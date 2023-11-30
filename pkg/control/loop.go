package control

import (
	"revengy.io/gco/agent/internal/concurrency"
	"revengy.io/gco/agent/internal/log"
	"revengy.io/gco/agent/internal/provider"
	"revengy.io/gco/agent/internal/state"
	"revengy.io/gco/agent/internal/state/diff"
)

// StateUpdateHandler defines a function which will be called when a state update has been
// received from the external container provider.
type StateUpdateHandler func(spec state.Spec)

// Control defines a structure which is responsible for keeping the system in the correct state
// by applying and observing the changes coming from the user and the external system.
type Control struct {
	provider   provider.Provider
	reconciler *diff.Reconciler

	apply     chan state.Spec
	observe   chan state.Spec
	errors    chan []error
	exit      chan struct{}
	semaphore concurrency.Semaphore
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

		apply:     make(chan state.Spec, 1),
		observe:   make(chan state.Spec, 1),
		errors:    make(chan []error, 1),
		exit:      make(chan struct{}),
		semaphore: concurrency.NewSemaphore(1), // only allow a single call to be in progress concurrently
	}, nil
}

// Start defines a function which will start the control loop for keeping the system in the correct state.
// This method will block until the 'exit' signal has been received.
func (c *Control) Start() {
	log.Info("Resource control loop has been started")

	for {

		log.Debugf("Applying lock on control loop")
		c.semaphore.Lock()

		select {
		case desired := <-c.apply:
			log.Infof("Received signal from 'apply' channel (applications=%d)", len(desired.Applications))
			c.errors <- c.reconciler.Apply(&desired)
		case actual := <-c.observe:
			log.Infof("Received signal from 'observe' channel (applications=%d)", len(actual.Applications))
			c.errors <- c.reconciler.Observe(&actual)
		case <-c.exit:
			log.Debug("Received signal from 'exit' channel")
			return
		}

		log.Debugf("Releasing lock on control loop")
		c.semaphore.Unlock()
	}
}

// Stop halts the control loop and stops handling state updates.
func (c *Control) Stop() {
	log.Debug("Closing 'exit' channel")
	close(c.exit)
}

// Apply will apply the desired state to the reconciler and make sure the system stays up to date.
func (c *Control) Apply(spec state.Spec) []error {
	c.apply <- spec
	return <-c.errors
}

// Observe will observe a change from the external system and propagate it to the reconciler to decide what needs to happen.
func (c *Control) Observe(spec state.Spec) []error {
	c.observe <- spec
	return <-c.errors
}
