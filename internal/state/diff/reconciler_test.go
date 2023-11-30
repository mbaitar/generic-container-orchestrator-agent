package diff

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"revengy.io/gco/agent/internal/state"
	"revengy.io/gco/agent/pkg/feature"
	"revengy.io/gco/agent/pkg/resource"
	"testing"
)

func TestInitReconciler(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	if assert.NotNil(t, reconciler, "should not be nil") {
		assert.Equal(t, provider, reconciler.provider)
		assert.NotNil(t, reconciler.actual, "actual state should not be nil")
		assert.NotNil(t, reconciler.desired, "desired state should not be nil")
	}
}

func TestReconciler_Apply(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	desired := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}

	// it will verify the changes by pulling the actual state from the provider
	provider.actualReturn = desired
	reconciler.Apply(desired)
	assert.Equal(t, 1, len(provider.createCalls), "should have created one new app")
	assert.Equal(t, 1, provider.actualCalls, "should have called ActualState()")

	provider.reset()

	// actual state equals requested desired state
	reconciler.Apply(desired)
	assert.Equal(t, 0, len(provider.createCalls), "should not have tried to re add the same application")
	assert.Equal(t, 0, provider.actualCalls, "should not have called ActualState()")

	provider.reset()

	// remove the created application
	provider.actualReturn = state.EmptySpec()
	reconciler.Apply(state.EmptySpec())
	assert.Equal(t, 1, len(provider.removeCalls), "should have removed the application")
	assert.Equal(t, 1, provider.actualCalls, "should have called ActualState()")

	provider.reset()

	// next create call will fail
	provider.createErr = errors.New("test error")
	reconciler.Apply(desired)
	assert.Equal(t, 1, len(provider.createCalls), "should have tried to create an application")
	assert.Equal(t, 0, provider.actualCalls, "should not have called ActualState()")
}

func TestReconciler_Apply_actualStateError(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}

	reconciler.WithInitialActualState(actual)

	provider.actualReturn = state.EmptySpec()
	provider.actualErr = errors.New("test error")
	reconciler.Apply(state.EmptySpec())

	assert.Equal(t, 1, len(provider.removeCalls), "should have removed an application")
	assert.Equal(t, 1, provider.actualCalls, "should have called ActualState()")

	assert.Equal(t, actual, reconciler.actual, "should not have updated the actual state as it failed")
}

func TestReconciler_Apply_removalError(t *testing.T) {
	actual := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}

	provider := &TestProvider{}
	reconciler := InitReconciler(provider).WithInitialActualState(actual)

	provider.removeErr = errors.New("test error")
	reconciler.Apply(state.EmptySpec())

	assert.Equal(t, 1, len(provider.removeCalls), "should have tried to remove an application")
	assert.Equal(t, 0, provider.actualCalls, "should not have called ActualState()")
}

func TestReconciler_Apply_update(t *testing.T) {
	appOneActual := SampleApp("app-1")
	appOneDesired := SampleApp("app-1")
	appOneDesired.Image.Tag = "v1.0.0"

	actual := &state.Spec{
		Applications: []resource.Application{
			*appOneActual,
		},
	}

	desired := &state.Spec{
		Applications: []resource.Application{
			*appOneDesired,
		},
	}

	provider := &TestProvider{}
	provider.actualReturn = desired
	reconciler := InitReconciler(provider).WithInitialActualState(actual)
	reconciler.Apply(desired)

	assert.Equal(t, 1, len(provider.updateCalls), "should have updated the application")
	assert.Equal(t, 1, provider.actualCalls, "should have called ActualState()")

	provider.reset()

	provider.updateErr = errors.New("test error")
	reconciler.Apply(actual)

	assert.Equal(t, 1, len(provider.updateCalls), "should have tried to update the application")
	assert.Equal(t, 0, provider.actualCalls, "should not have called ActualState()")

}

func TestReconciler_Observe(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}

	desired := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
			*SampleApp("app-2"),
		},
	}

	reconciler.desired = desired
	reconciler.Observe(actual)

	assert.Equal(t, 1, len(provider.createCalls), "should have created a new application")
	assert.Equal(t, 0, provider.actualCalls, "should not have to re-fetch ActualState()")
}

func TestReconciler_Apply_creatingFeature(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := state.EmptySpec()
	desired := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "debug"},
		},
	}

	reconciler.WithInitialActualState(actual)
	reconciler.Apply(desired)

	assert.Equal(t, 1, len(provider.createFeatCalls), "should have called #CreateFeature()")
	assert.Equal(t, 1, provider.actualCalls, "should have called #ActualState()")
}

func TestReconciler_Apply_creatingFeatureWithError(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := state.EmptySpec()
	desired := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "debug"},
		},
	}

	provider.createFeatErr = errors.New("unable to create feature")

	reconciler.WithInitialActualState(actual)
	reconciler.Apply(desired)

	assert.Equal(t, 1, len(provider.createFeatCalls), "should have called #CreateFeature()")
	assert.Equal(t, 0, provider.actualCalls, "should not have called #ActualState()")
}

func TestReconciler_Apply_updateFeature(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "info"},
		},
	}
	desired := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "debug"},
		},
	}

	reconciler.WithInitialActualState(actual)
	reconciler.Apply(desired)

	assert.Equal(t, 1, len(provider.updateFeatCalls), "should have called #UpdateFeature()")
	assert.Equal(t, 1, provider.actualCalls, "should have called #ActualState()")
}

func TestReconciler_Apply_updateFeatureWithError(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "info"},
		},
	}
	desired := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "debug"},
		},
	}

	provider.updateFeatErr = errors.New("unable to update feature")

	reconciler.WithInitialActualState(actual)
	reconciler.Apply(desired)

	assert.Equal(t, 1, len(provider.updateFeatCalls), "should have called #UpdateFeature()")
	assert.Equal(t, 0, provider.actualCalls, "should not have called #ActualState()")
}

func TestReconciler_Apply_removeFeature(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "info"},
		},
	}
	desired := state.EmptySpec()

	reconciler.WithInitialActualState(actual)
	reconciler.Apply(desired)

	assert.Equal(t, 1, len(provider.removeFeatCalls), "should have called #RemoveFeature()")
	assert.Equal(t, 1, provider.actualCalls, "should have called #ActualState()")
}

func TestReconciler_Apply_removeFeatureWithError(t *testing.T) {
	provider := &TestProvider{}
	reconciler := InitReconciler(provider)

	actual := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{LogLevel: "info"},
		},
	}
	desired := state.EmptySpec()

	provider.removeFeatErr = errors.New("unable to update feature")

	reconciler.WithInitialActualState(actual)
	reconciler.Apply(desired)

	assert.Equal(t, 1, len(provider.removeFeatCalls), "should have called #RemoveFeature()")
	assert.Equal(t, 0, provider.actualCalls, "should not have called #ActualState()")
}
