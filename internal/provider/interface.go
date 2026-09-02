package provider

import (
	"context"

	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
)

// Provider defines the external container system which will be used
// to apply the changes based on the desired and actual state of the system.
type Provider interface {
	// CreateApplication defines a function which will create a new application.
	CreateApplication(ctx context.Context, app *resource.Application) error

	// UpdateApplication defines a function which will update an existing application.
	UpdateApplication(ctx context.Context, app *resource.Application) error

	// RemoveApplication defines a function which will remove an existing application.
	RemoveApplication(ctx context.Context, app *resource.Application) error

	// CreateFeature defines a function which will create a new feature.
	CreateFeature(ctx context.Context, feat feature.Feature) error

	// UpdateFeature defines a function which will update an existing feature.
	UpdateFeature(ctx context.Context, feat feature.Feature) error

	// RemoveFeature defines a function which will remove an existing feature.
	RemoveFeature(ctx context.Context, feat feature.Feature) error

	// ActualState defines a function which will analyze the current state and return it in the form of a specification.
	ActualState(ctx context.Context) (*state.Spec, error)
}

// Watcher is an optional interface a Provider can implement to notify the control
// loop about changes happening in the external system (e.g. a container that
// crashed or was removed manually). The returned channel emits an empty struct
// whenever the externally managed state may have changed; the receiver is
// expected to fetch the ActualState and reconcile.
type Watcher interface {
	// Watch starts watching the external system until the context is cancelled.
	Watch(ctx context.Context) (<-chan struct{}, error)
}
