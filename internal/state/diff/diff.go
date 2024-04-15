package diff

import (
	"github.com/mbaitar/gco/agent/internal/flag"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
)

// changes defines a structure which holds the comparison results of to states.
type changes struct {
	apps struct {
		unchanged []resource.Application
		added     []resource.Application
		changed   []resource.Application
		removed   []resource.Application
	}

	features struct {
		unchanged []feature.Feature
		added     []feature.Feature
		changed   []feature.Feature
		removed   []feature.Feature
	}
}

type specMap struct {
	spec          *state.Spec
	appLookup     map[string]resource.Application
	featureLookup map[string]feature.Feature
}

func newSpecMap(spec *state.Spec) *specMap {
	appLookup := make(map[string]resource.Application)
	featureLookup := make(map[string]feature.Feature)

	for _, app := range spec.Applications {
		name := app.Name
		appLookup[name] = app
	}

	if spec.Feature.FluentBit != nil {
		name := spec.Feature.FluentBit.Name()
		featureLookup[name] = spec.Feature.FluentBit
	}

	return &specMap{
		spec:          spec,
		appLookup:     appLookup,
		featureLookup: featureLookup,
	}
}

func (sm *specMap) HasApp(name string) *resource.Application {
	res, match := sm.appLookup[name]
	if !match {
		return nil
	} else {
		return &res
	}
}

func (sm *specMap) HasFeature(name string) feature.Feature {
	res, match := sm.featureLookup[name]
	if !match {
		return nil
	} else {
		return res
	}
}

func (sm *specMap) RemoveApp(name string) {
	delete(sm.appLookup, name)
}

func (sm *specMap) RemoveFeature(name string) {
	delete(sm.featureLookup, name)
}

// compare defines a function which will calculate the state changes between the actual and desired system.
func compare(desired *state.Spec, actual *state.Spec) *changes {
	output := &changes{}

	if desired == nil {
		desired = state.EmptySpec()
	}

	if actual == nil {
		actual = state.EmptySpec()
	}

	// evaluate both state specifications before comparing
	log.Debug("Evaluating state specifications before comparing")
	desired.Evaluate()
	actual.Evaluate()

	desiredMap := newSpecMap(desired)
	actualMap := newSpecMap(actual)

	output.apps.unchanged = make([]resource.Application, 0, len(desired.Applications))
	output.apps.added = make([]resource.Application, 0, len(desired.Applications))
	output.apps.changed = make([]resource.Application, 0, len(desired.Applications))
	output.apps.removed = make([]resource.Application, 0, len(actual.Applications))

	output.features.unchanged = make([]feature.Feature, 0)
	output.features.added = make([]feature.Feature, 0)
	output.features.changed = make([]feature.Feature, 0)
	output.features.removed = make([]feature.Feature, 0)

	// determine added, changed and unchanged applications
	for name, app := range desiredMap.appLookup {
		match := actualMap.HasApp(name)

		if match == nil {
			// new resource
			output.apps.added = append(output.apps.added, app)
		} else {

			if flag.Has(flag.IgnoreInstanceDiff) {
				app.Instances = 1
			}

			actualHash := match.CalculateHash()
			desiredHash := app.CalculateHash()

			hashMismatch := actualHash != desiredHash
			instanceMismatch := app.Instances != match.Instances
			log.Debugf("Difference calculation for app '%s' (hash=%v, instance=%v)", app.Name, hashMismatch, instanceMismatch)

			if hashMismatch || instanceMismatch {
				output.apps.changed = append(output.apps.changed, app)
			} else {
				output.apps.unchanged = append(output.apps.unchanged, app)
			}
		}

		actualMap.RemoveApp(name) // remove processed application
	}

	// add remaining applications in actual state as removed
	for _, app := range actualMap.appLookup {
		output.apps.removed = append(output.apps.removed, app)
	}

	// determine added, changed and unchanged features
	for name, feat := range desiredMap.featureLookup {
		match := actualMap.HasFeature(name)

		if match == nil {
			output.features.added = append(output.features.added, feat)
		} else {

			actualHash := match.ConfigHash()
			desiredHash := feat.ConfigHash()

			hashMismatch := actualHash != desiredHash
			log.Debugf("Difference calculation for feature '%s' (hash=%v)", name, hashMismatch)

			if hashMismatch {
				output.features.changed = append(output.features.changed, feat)
			} else {
				output.features.unchanged = append(output.features.unchanged, feat)
			}

		}

		actualMap.RemoveFeature(name) // remove processed feature
	}

	// add remaining features in actual state as removed
	for _, feat := range actualMap.featureLookup {
		output.features.removed = append(output.features.removed, feat)
	}

	return output
}
