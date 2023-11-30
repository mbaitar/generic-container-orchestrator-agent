package state

import (
	"revengy.io/gco/agent/pkg/feature"
)

// Evaluate updates the current application specifications based on the enabled features.
func (s *Spec) Evaluate() {
	isFluentBitEnabled := s.IsFeatureEnabled(feature.NameFluentBit)

	for i := range s.Applications {
		app := &s.Applications[i]
		hasLogConfig := app.LogConfig != nil

		// evaluate each app based on the enabled features
		if !hasLogConfig && isFluentBitEnabled {
			app.LogConfig = s.Feature.FluentBit.DefaultLogConfig()
		}

	}
}
