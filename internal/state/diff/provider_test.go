package diff

import (
	"github.com/mabaitar/gco/agent/internal/state"
	"github.com/mabaitar/gco/agent/pkg/feature"
	"github.com/mabaitar/gco/agent/pkg/resource"
)

type TestProvider struct {
	createCalls []resource.Application
	createErr   error

	updateCalls []resource.Application
	updateErr   error

	removeCalls []resource.Application
	removeErr   error

	createFeatCalls []feature.Feature
	createFeatErr   error

	updateFeatCalls []feature.Feature
	updateFeatErr   error

	removeFeatCalls []feature.Feature
	removeFeatErr   error

	actualReturn *state.Spec
	actualErr    error
	actualCalls  int
}

func (t *TestProvider) reset() {
	t.createCalls = make([]resource.Application, 0)
	t.createErr = nil
	t.updateCalls = make([]resource.Application, 0)
	t.updateErr = nil
	t.removeCalls = make([]resource.Application, 0)
	t.removeErr = nil
	t.createFeatCalls = make([]feature.Feature, 0)
	t.createFeatErr = nil
	t.updateFeatCalls = make([]feature.Feature, 0)
	t.updateFeatErr = nil
	t.removeFeatCalls = make([]feature.Feature, 0)
	t.removeFeatErr = nil
	t.actualReturn = nil
	t.actualErr = nil
	t.actualCalls = 0
}

func (t *TestProvider) CreateApplication(app *resource.Application) error {
	t.createCalls = append(t.createCalls, *app)
	return t.createErr
}

func (t *TestProvider) UpdateApplication(app *resource.Application) error {
	t.updateCalls = append(t.updateCalls, *app)
	return t.updateErr
}

func (t *TestProvider) RemoveApplication(app *resource.Application) error {
	t.removeCalls = append(t.removeCalls, *app)
	return t.removeErr
}

func (t *TestProvider) CreateFeature(feat feature.Feature) error {
	t.createFeatCalls = append(t.createFeatCalls, feat)
	return t.createFeatErr
}

func (t *TestProvider) UpdateFeature(feat feature.Feature) error {
	t.updateFeatCalls = append(t.updateFeatCalls, feat)
	return t.updateFeatErr
}

func (t *TestProvider) RemoveFeature(feat feature.Feature) error {
	t.removeFeatCalls = append(t.removeFeatCalls, feat)
	return t.removeFeatErr
}

func (t *TestProvider) ActualState() (*state.Spec, error) {
	t.actualCalls += 1
	return t.actualReturn, t.actualErr
}
