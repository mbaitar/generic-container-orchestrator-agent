package control

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/provider"
	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/internal/state/diff"

	"golang.org/x/sync/semaphore"
)

const (
	// resyncInterval is the fallback interval at which the actual state is re-fetched
	// from the provider, in case an external change was missed by the watcher.
	resyncInterval = 30 * time.Second

	// debounceDelay is how long the control loop waits after the last provider event
	// before fetching the actual state, so bursts of events result in a single resync.
	debounceDelay = 2 * time.Second
)

// StateUpdateHandler defines a function which will be called when a state update has been
// received from the external container provider.
type StateUpdateHandler func(spec state.Spec)

// Control defines a structure which is responsible for keeping the system in the correct state
// by applying and observing the changes coming from the user and the external system.
type Control struct {
	provider   provider.Provider
	reconciler *diff.Reconciler

	apply          chan state.Spec
	observe        chan state.Spec
	resyncRequests chan struct{}
	exit           chan struct{}

	sem      *semaphore.Weighted
	handlers map[string]StateUpdateHandler

	// resyncInterval and debounceDelay control the watch behaviour and are
	// overridable for testing purposes.
	resyncInterval time.Duration
	debounceDelay  time.Duration
}

// InitControl will initialize the control structure used for keeping the system in the correct state.
func InitControl(ctx context.Context, p provider.Provider) (*Control, error) {
	// fetch first actual state
	actual, err := p.ActualState(ctx)
	if err != nil {
		log.Warn("Unable to retrieve initial actual state from external provider")
		return nil, err
	}

	log.Infof("Retrieved current application state (applications=%d)", len(actual.Applications))
	reconciler := diff.InitReconciler(p).WithInitialActualState(actual)
	return &Control{
		provider:   p,
		reconciler: reconciler,

		apply:          make(chan state.Spec, 1),
		observe:        make(chan state.Spec, 1),
		resyncRequests: make(chan struct{}, 1),
		exit:           make(chan struct{}),

		sem:      semaphore.NewWeighted(1),
		handlers: make(map[string]StateUpdateHandler),

		resyncInterval: resyncInterval,
		debounceDelay:  debounceDelay,
	}, nil
}

// Start defines a function which will start the control loop for keeping the system in the correct state.
// This method will block until the context is cancelled or the 'exit' signal has been received.
func (c *Control) Start(ctx context.Context) {
	log.Info("Resource control loop has been started")

	// watch the provider for external changes and periodically resync
	go c.watchProvider(ctx)

	for {
		select {
		case desired := <-c.apply:
			log.Infof("Received signal from 'apply' channel (applications=%d)", len(desired.Applications))
			c.reconciler.Apply(ctx, &desired)
		case actual := <-c.observe:
			log.Infof("Received signal from 'observe' channel (applications=%d)", len(actual.Applications))
			c.reconciler.Observe(ctx, &actual)
		case <-c.resyncRequests:
			c.resync(ctx)
		case <-ctx.Done():
			log.Debug("Control loop context has been cancelled")
			return
		case <-c.exit:
			log.Debug("Received signal from 'exit' channel")
			return
		}
	}
}

// watchProvider listens for change events from the provider (when supported) and
// periodically triggers a resync so the actual state never drifts unnoticed.
func (c *Control) watchProvider(ctx context.Context) {
	var events <-chan struct{}
	if watcher, ok := c.provider.(provider.Watcher); ok {
		ch, err := watcher.Watch(ctx)
		if err != nil {
			log.Warnf("Unable to watch provider for external changes: %v", err)
		} else {
			log.Info("Watching provider for external state changes")
			events = ch
		}
	}

	ticker := time.NewTicker(c.resyncInterval)
	defer ticker.Stop()

	// debounce timer starts inactive
	debounce := time.NewTimer(c.debounceDelay)
	if !debounce.Stop() {
		<-debounce.C
	}
	defer debounce.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.exit:
			return
		case _, ok := <-events:
			if !ok {
				// watcher stopped, fall back to periodic resync only
				log.Warn("Provider watcher has stopped, falling back to periodic resync")
				events = nil
				continue
			}
			debounce.Reset(c.debounceDelay)
		case <-debounce.C:
			c.requestResync()
		case <-ticker.C:
			c.requestResync()
		}
	}
}

// requestResync asks the control loop to fetch the actual state, a single
// pending request is enough. The fetch happens on the control loop goroutine
// itself: ActualState also cleans up stale containers, so it must never run
// concurrently with an in-flight update.
func (c *Control) requestResync() {
	select {
	case c.resyncRequests <- struct{}{}:
	default:
	}
}

// resync fetches the actual state from the provider and reconciles it.
func (c *Control) resync(ctx context.Context) {
	actual, err := c.provider.ActualState(ctx)
	if err != nil {
		log.Warnf("Unable to fetch actual state during resync: %v", err)
		return
	}

	log.Infof("Resynced actual state from provider (applications=%d)", len(actual.Applications))
	c.reconciler.Observe(ctx, actual)
}

// Stop halts the control loop and stops handling state updates.
func (c *Control) Stop() {
	log.Debug("Closing 'exit' channel")
	close(c.exit)
}

// Apply will apply the desired state to the reconciler and make sure the system stays up to date.
func (c *Control) Apply(spec state.Spec) {
	c.apply <- spec
}

// Observe will observe a change from the external system and propagate it to the reconciler to decide what needs to happen.
func (c *Control) Observe(spec state.Spec) {
	c.observe <- spec
}

// RegisterHandler registers a new handler and returns the handler signature for optional removal
func (c *Control) RegisterHandler(handler StateUpdateHandler) string {
	ctx := context.Background()
	if err := c.sem.Acquire(ctx, 1); err != nil {
		log.Errorf("unable to acquire handler lock: %v", err)
		os.Exit(1)
	}
	defer c.sem.Release(1)

	addr := uuid.NewString()
	c.handlers[addr] = handler
	return addr
}

// RemoveHandler removes the handler using the signature received from the RegisterHandler method.
func (c *Control) RemoveHandler(signature string) {
	ctx := context.Background()
	if err := c.sem.Acquire(ctx, 1); err != nil {
		log.Errorf("unable to acquire handler lock: %v", err)
		os.Exit(1)
	}
	defer c.sem.Release(1)

	delete(c.handlers, signature)
}
