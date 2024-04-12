package state

import (
	"errors"
	"fmt"

	"github.com/mabaitar/gco/agent/pkg/feature"
	"github.com/mabaitar/gco/agent/pkg/resource"
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
		return errors.New("application already exists")
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

	return fmt.Errorf("no application found to update with name '%s'", update.Name)
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
		return fmt.Errorf("no application found to remove with name '%s'", name)
	}

	s.Applications = append(s.Applications[:idx], s.Applications[idx+1:]...)
	return nil
}

// IsFeatureEnabled returns true if the specified name of the feature can be found in the state specification.
func (s *Spec) IsFeatureEnabled(name string) bool {
	if feature.NameFluentBit == name {
		return s.Feature.FluentBit != nil
	}

	return false
}
