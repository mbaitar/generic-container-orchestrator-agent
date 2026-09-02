package control

import (
	"context"
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

func (n NilProvider) CreateFeature(ctx context.Context, feat feature.Feature) error {
	return nil
}

func (n NilProvider) UpdateFeature(ctx context.Context, feat feature.Feature) error {
	return nil
}

func (n NilProvider) RemoveFeature(ctx context.Context, feat feature.Feature) error {
	return nil
}

func (n NilProvider) CreateApplication(ctx context.Context, app *resource.Application) error {
	return nil
}

func (n NilProvider) UpdateApplication(ctx context.Context, app *resource.Application) error {
	return nil
}

func (n NilProvider) RemoveApplication(ctx context.Context, app *resource.Application) error {
	return nil
}

func (n NilProvider) ActualState(ctx context.Context) (*state.Spec, error) {
	empty := state.EmptySpec()
	return empty, nil
}

func TestControl_RegisterAndRemoveHandler(t *testing.T) {
	control, _ := InitControl(context.Background(), &NilProvider{})

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
