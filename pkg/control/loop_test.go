package control

import (
	"log"
	"sync"
	"testing"

	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
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

func TestControl_RegisterAndRemoveHandler(t *testing.T) {
	control, _ := InitControl(&NilProvider{})

	n := 100
	wg := sync.WaitGroup{}
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			handler := func(spec state.Spec) {}
			sign := control.RegisterHandler(handler)
			assert.NotEqual(t, "", sign, "should include a real signature")
			control.RemoveHandler(sign)
		}()
	}

	wg.Wait()
	log.Printf("hello")
}
