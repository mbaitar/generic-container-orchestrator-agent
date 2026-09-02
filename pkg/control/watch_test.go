package control

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

// WatchingProvider is a fake provider which implements the optional Watcher
// interface and records the reconciliation calls it receives.
type WatchingProvider struct {
	mu     sync.Mutex
	actual *state.Spec
	events chan struct{}

	createCalls int
	updateCalls int
	removeCalls int
}

func NewWatchingProvider() *WatchingProvider {
	return &WatchingProvider{
		actual: state.EmptySpec(),
		events: make(chan struct{}, 1),
	}
}

// SetActualState simulates an external change to the system.
func (w *WatchingProvider) SetActualState(spec *state.Spec) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.actual = spec
}

// EmitEvent simulates the provider noticing an external change.
func (w *WatchingProvider) EmitEvent() {
	w.events <- struct{}{}
}

func (w *WatchingProvider) Calls() (create int, update int, remove int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.createCalls, w.updateCalls, w.removeCalls
}

func (w *WatchingProvider) CreateApplication(ctx context.Context, app *resource.Application) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.createCalls++
	return nil
}

func (w *WatchingProvider) UpdateApplication(ctx context.Context, app *resource.Application) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.updateCalls++
	return nil
}

func (w *WatchingProvider) RemoveApplication(ctx context.Context, app *resource.Application) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.removeCalls++
	return nil
}

func (w *WatchingProvider) CreateFeature(ctx context.Context, feat feature.Feature) error {
	return nil
}

func (w *WatchingProvider) UpdateFeature(ctx context.Context, feat feature.Feature) error {
	return nil
}

func (w *WatchingProvider) RemoveFeature(ctx context.Context, feat feature.Feature) error {
	return nil
}

func (w *WatchingProvider) ActualState(ctx context.Context) (*state.Spec, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.actual, nil
}

func (w *WatchingProvider) Watch(ctx context.Context) (<-chan struct{}, error) {
	return w.events, nil
}

func exampleApp(instances int) resource.Application {
	return resource.Application{
		Name:      "test-app",
		Image:     resource.Image{Name: "nginx", Tag: "latest"},
		Instances: instances,
	}
}

func specWith(app resource.Application) *state.Spec {
	spec := state.EmptySpec()
	spec.Applications = append(spec.Applications, app)
	return spec
}

func TestControl_SelfHealsAfterProviderEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider := NewWatchingProvider()
	control, err := InitControl(ctx, provider)
	assert.Nil(t, err, "should have initialized control")

	// speed up the watch behaviour for testing
	control.debounceDelay = 10 * time.Millisecond
	control.resyncInterval = time.Hour

	go control.Start(ctx)

	// apply the desired state, the application should be created
	control.Apply(*specWith(exampleApp(1)))
	assert.Eventually(t, func() bool {
		create, _, _ := provider.Calls()
		return create == 1
	}, 2*time.Second, 10*time.Millisecond, "should have created the application")

	// simulate the container dying externally (present, but no longer running)
	provider.SetActualState(specWith(exampleApp(0)))
	provider.EmitEvent()

	assert.Eventually(t, func() bool {
		_, update, _ := provider.Calls()
		return update == 1
	}, 2*time.Second, 10*time.Millisecond, "should have restarted the application after the event")
}

func TestControl_SelfHealsAfterResyncTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider := NewWatchingProvider()
	control, err := InitControl(ctx, provider)
	assert.Nil(t, err, "should have initialized control")

	// no events, only the fallback resync ticker
	control.debounceDelay = time.Hour
	control.resyncInterval = 20 * time.Millisecond

	go control.Start(ctx)

	// apply the desired state, the application should be created
	control.Apply(*specWith(exampleApp(1)))
	assert.Eventually(t, func() bool {
		create, _, _ := provider.Calls()
		return create == 1
	}, 2*time.Second, 10*time.Millisecond, "should have created the application")

	// simulate the container being removed externally without an event
	provider.SetActualState(state.EmptySpec())

	assert.Eventually(t, func() bool {
		create, _, _ := provider.Calls()
		return create == 2
	}, 2*time.Second, 10*time.Millisecond, "should have recreated the application after a resync")
}
