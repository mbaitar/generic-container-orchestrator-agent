package control

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/mabaitar/gco/agent/internal/log"
	"github.com/mabaitar/gco/agent/internal/provider"
	"github.com/mabaitar/gco/agent/internal/state"
	"github.com/mabaitar/gco/agent/internal/state/diff"

	"golang.org/x/sync/semaphore"
)

// StateUpdateHandler defines a function which will be called when a state update has been
// received from the external container provider.
type StateUpdateHandler func(spec state.Spec)

// Control defines a structure which is responsible for keeping the system in the correct state
// by applying and observing the changes coming from the user and the external system.
type Control struct {
	provider   provider.Provider
	reconciler *diff.Reconciler

	apply   chan state.Spec
	observe chan state.Spec
	exit    chan struct{}

	sem      *semaphore.Weighted
	handlers map[string]StateUpdateHandler
}

// InitControl will initialize the control structure used for keeping the system in the correct state.
func InitControl(p provider.Provider) (*Control, error) {
	// fetch first actual state
	actual, err := p.ActualState()
	if err != nil {
		log.Warn("Unable to retrieve initial actual state from external provider")
		return nil, err
	}

	log.Infof("Retrieved current application state (applications=%d)", len(actual.Applications))
	reconciler := diff.InitReconciler(p).WithInitialActualState(actual)
	return &Control{
		provider:   p,
		reconciler: reconciler,

		apply:   make(chan state.Spec, 1),
		observe: make(chan state.Spec, 1),
		exit:    make(chan struct{}),

		sem:      semaphore.NewWeighted(1),
		handlers: make(map[string]StateUpdateHandler),
	}, nil
}

// Start defines a function which will start the control loop for keeping the system in the correct state.
// This method will block until the 'exit' signal has been received.
func (c *Control) Start() {
	log.Info("Resource control loop has been started")

	for {
		select {
		case desired := <-c.apply:
			log.Infof("Received signal from 'apply' channel (applications=%d)", len(desired.Applications))
			c.reconciler.Apply(&desired)
		case actual := <-c.observe:
			log.Infof("Received signal from 'observe' channel (applications=%d)", len(actual.Applications))
			c.reconciler.Observe(&actual)
		case <-c.exit:
			log.Debug("Received signal from 'exit' channel")
			return
		}
	}
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
