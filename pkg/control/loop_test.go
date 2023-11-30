package control

import (
	"dsync.io/gco/agent/internal/state"
	"dsync.io/gco/agent/pkg/feature"
	"dsync.io/gco/agent/pkg/resource"
)

type NilProvider struct {
}

func (n NilProvider) CreateFeature(feat feature.Feature) error {
	return nil
}

func (n NilProvider) UpdateFeature(feat feature.Feature) error {
	return nil
}

func (n NilProvider) RemoveFeature(feat feature.Feature) error {
	return nil
}

func (n NilProvider) CreateApplication(app *resource.Application) error {
	return nil
}

func (n NilProvider) UpdateApplication(app *resource.Application) error {
	return nil
}

func (n NilProvider) RemoveApplication(app *resource.Application) error {
	return nil
}

func (n NilProvider) ActualState() (*state.Spec, error) {
	empty := state.EmptySpec()
	return empty, nil
}
