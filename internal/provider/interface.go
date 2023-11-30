package provider

import (
	"revengy.io/gco/agent/internal/state"
	"revengy.io/gco/agent/pkg/feature"
	"revengy.io/gco/agent/pkg/resource"
)

// Provider defines the external container system which will be used
// to apply the changes based on the desired and actual state of the system.
type Provider interface {
	// CreateApplication defines a function which will create a new application.
	CreateApplication(app *resource.Application) error

	// UpdateApplication defines a function which will update an existing application.
	UpdateApplication(app *resource.Application) error

	// RemoveApplication defines a function which will remove an existing application.
	RemoveApplication(app *resource.Application) error

	// CreateFeature defines a function which will create a new feature.
	CreateFeature(feat feature.Feature) error

	// UpdateFeature defines a function which will update an existing feature.
	UpdateFeature(feat feature.Feature) error

	// RemoveFeature defines a function which will remove an existing feature.
	RemoveFeature(feat feature.Feature) error

	// ActualState defines a function which will analyze the current state and return it in the form of a specification.
	ActualState() (*state.Spec, error)
}
