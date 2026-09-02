package state

import (
	"errors"
	"fmt"

	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
)

var (
	// ErrApplicationExists is returned when an application with the same name already exists.
	ErrApplicationExists = errors.New("application already exists")
	// ErrApplicationNotFound is returned when no application matches the given name.
	ErrApplicationNotFound = errors.New("application not found")
)

// Spec describes the application specification in an 'as is' or 'should be' state.
type Spec struct {
	// Applications lists the available applications in the current state specification.
	Applications []resource.Application `json:"applications"`

	// Feature contains all the enabled features for the agent.
	Feature Feature `json:"feature,omitempty"`
}

// EmptySpec returns a new empty state specification
func EmptySpec() *Spec {
	return &Spec{
		Applications: make([]resource.Application, 0),
	}
}

// GetApplication tries to find the application matching the given name.
func (s *Spec) GetApplication(name string) *resource.Application {
	for _, app := range s.Applications {
		if app.Name == name {
			return &app
		}
	}

	return nil
}

// AddApplication appends a new application to the state if no other application exists with the same name.
func (s *Spec) AddApplication(app resource.Application) error {
	match := s.GetApplication(app.Name)
	if match != nil {
		return fmt.Errorf("%w: '%s'", ErrApplicationExists, app.Name)
	}

	s.Applications = append(s.Applications, app)
	return nil
}

// UpdateApplication tries to find the matching application and updates the resource.Application.
func (s *Spec) UpdateApplication(update resource.Application) error {
	for i, app := range s.Applications {
		if app.Name == update.Name {
			s.Applications[i] = update
			return nil
		}
	}

	return fmt.Errorf("no application found to update with name '%s': %w", update.Name, ErrApplicationNotFound)
}

// RemoveApplication tries to find the matching application by name and removes it from the current spec.
func (s *Spec) RemoveApplication(name string) error {
	idx := -1

	for i, app := range s.Applications {
		if app.Name == name {
			idx = i
			break
		}
	}

	if idx < 0 {
		return fmt.Errorf("no application found to remove with name '%s': %w", name, ErrApplicationNotFound)
	}

	s.Applications = append(s.Applications[:idx], s.Applications[idx+1:]...)
	return nil
}

// Validate checks every application in the specification and rejects duplicate
// application names. It guards the ingestion paths which do not go through the
// service layer, such as the persisted state file.
func (s *Spec) Validate() error {
	names := make(map[string]struct{})

	for i := range s.Applications {
		app := &s.Applications[i]

		if err := app.Validate(); err != nil {
			return fmt.Errorf("application '%s': %w", app.Name, err)
		}

		if _, duplicate := names[app.Name]; duplicate {
			return fmt.Errorf("duplicate application name '%s'", app.Name)
		}
		names[app.Name] = struct{}{}
	}

	return nil
}

// IsFeatureEnabled returns true if the specified name of the feature can be found in the state specification.
func (s *Spec) IsFeatureEnabled(name string) bool {
	if feature.NameFluentBit == name {
		return s.Feature.FluentBit != nil
	}

	return false
}
