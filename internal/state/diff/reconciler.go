package diff

import (
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/provider"
	"github.com/mbaitar/gco/agent/internal/state"
)

// Reconciler defines a structure that is responsible for the reconciliation process
// in keeping the desired state as close as possible to the actual state.
type Reconciler struct {
	// provider provides an interface to the external containerized system
	provider provider.Provider
	// desired defines the desired state in which the system should be at any given time
	desired *state.Spec
	// actual defines the actual state in which the system currently is
	actual *state.Spec
}

// InitReconciler initialises a new reconciler with the associated external container provider.
func InitReconciler(provider provider.Provider) *Reconciler {
	return &Reconciler{
		provider: provider,
		desired:  state.EmptySpec(),
		actual:   state.EmptySpec(),
	}
}

// WithInitialActualState sets the actual state of the reconciler without triggering any update.
func (r *Reconciler) WithInitialActualState(actual *state.Spec) *Reconciler {
	r.actual = actual
	return r
}

func (r *Reconciler) Apply(desired *state.Spec) {
	r.desired = desired
	r.update(true)
}

func (r *Reconciler) Observe(actual *state.Spec) {
	r.actual = actual
	r.update(false)
}

func (r *Reconciler) update(triggerFetch bool) {
	modified := false
	result := compare(r.desired, r.actual)

	// adding features (before applications), supporting infrastructure -> some applications might rely on it
	for _, feat := range result.features.added {
		err := r.provider.CreateFeature(feat)
		if err != nil {
			log.Errorf("Error while creating feature=%s: %v", feat.Name(), err)
		} else {
			log.Debugf("Created feature=%s with hash=%s", feat.Name(), feat.ConfigHash())
			modified = true
		}
	}

	// updating features (before applications)
	for _, feat := range result.features.changed {
		err := r.provider.UpdateFeature(feat)
		if err != nil {
			log.Errorf("Error while creating feature=%s: %v", feat.Name(), err)
		} else {
			log.Debugf("Updated feature=%s to hash=%s", feat.Name(), feat.ConfigHash())
			modified = true
		}
	}

	// remove applications -> first
	for _, app := range result.apps.removed {
		err := r.provider.RemoveApplication(&app)
		if err != nil {
			log.Errorf("Error while removing application=%s: %v", app.Name, err)
		} else {
			log.Debugf("Removed application=%s from state", app.Name)
			modified = true
		}
	}

	// update applications -> second
	for _, app := range result.apps.changed {
		err := r.provider.UpdateApplication(&app)
		if err != nil {
			log.Errorf("Error while updating application=%s: %v", app.Name, err)
		} else {
			log.Debugf("Updated application=%s to hash=%s", app.Name, app.CalculateHash())
			modified = true
		}
	}

	// create new applications -> last
	for _, app := range result.apps.added {
		err := r.provider.CreateApplication(&app)
		if err != nil {
			log.Errorf("Error while creating application=%s: %v", app.Name, err)
		} else {
			log.Debugf("Created application=%s with hash=%s", app.Name, app.CalculateHash())
			modified = true
		}
	}

	// remove features (after applications)
	for _, feat := range result.features.removed {
		err := r.provider.RemoveFeature(feat)
		if err != nil {
			log.Errorf("Error while removing feature=%s: %v", feat.Name(), err)
		} else {
			log.Debugf("Removed feature=%s from state", feat.Name())
			modified = true
		}
	}

	if modified && triggerFetch {
		log.Debug("Changes detected to external system, pulling latest actual state")
		actual, err := r.provider.ActualState()
		if err != nil {
			log.Errorf("Unable to get actual state from external system: %v", err)
		} else {
			r.actual = actual
		}
	}
}
