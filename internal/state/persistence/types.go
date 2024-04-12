package persistence

import (
	"github.com/mabaitar/gco/agent/internal/state"
)

// ChangeChannel defines a channel used for read only messages of a changed State.
type ChangeChannel <-chan state.Spec

// Controller defines the public facing interface for how to communicate with the persistence storage of the application.
// Regardless of which storage device is being used: files, k8s resources or any other means of storage.
type Controller interface {

	// GetChangeChannel returns a read only channel for receiving changes to the persistent state.
	GetChangeChannel() ChangeChannel

	// Persist performs a persistent action on the storage device for the given state specification.
	Persist(spec *state.Spec) error

	// Read defines a function which will try to read the current persisted state from the controller.
	Read() (*state.Spec, error)
}
