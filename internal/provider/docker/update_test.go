package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/mbaitar/gco/agent/pkg/resource"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

/* test helpers */

// orderedTestClient wraps the TestClient and additionally records the order of
// the container lifecycle calls across the different methods, so tests can
// assert e.g. that the old container is stopped before the new one is started.
type orderedTestClient struct {
	*TestClient
	sequence []string
}

func newOrderedTestClient() *orderedTestClient {
	return &orderedTestClient{TestClient: NewTestClient()}
}

func (o *orderedTestClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, containerName string) (container.CreateResponse, error) {
	o.sequence = append(o.sequence, "create")
	return o.TestClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

func (o *orderedTestClient) ContainerStart(ctx context.Context, id string, opts container.StartOptions) error {
	o.sequence = append(o.sequence, "start:"+id)
	return o.TestClient.ContainerStart(ctx, id, opts)
}

func (o *orderedTestClient) ContainerStop(ctx context.Context, id string, opts container.StopOptions) error {
	o.sequence = append(o.sequence, "stop:"+id)
	return o.TestClient.ContainerStop(ctx, id, opts)
}

func (o *orderedTestClient) ContainerRemove(ctx context.Context, id string, opts container.RemoveOptions) error {
	o.sequence = append(o.sequence, "remove:"+id)
	return o.TestClient.ContainerRemove(ctx, id, opts)
}

// updateTestApp returns an application publishing host port 8080.
func updateTestApp() *resource.Application {
	return &resource.Application{
		Name:  "web",
		Image: resource.Image{Name: "nginx", Tag: "latest"},
		Ports: []resource.Port{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		},
	}
}

// inspectWithState builds a minimal inspect response carrying the given
// container state, as consumed by awaitHealthy and fromDockerContainer.
func inspectWithState(status string, health *container.Health) container.InspectResponse {
	c := container.InspectResponse{}
	c.ContainerJSONBase = &container.ContainerJSONBase{
		ID: "container_id",
		State: &container.State{
			Status: status,
			Health: health,
		},
	}
	c.Config = &container.Config{
		Image:  "nginx:latest",
		Labels: map[string]string{},
	}
	c.HostConfig = &container.HostConfig{}
	return c
}

// inspectForApp builds an inspect response for a container that belongs to the
// application identified by the name label, carrying the given config hash.
func inspectForApp(id string, status string, name string, hash string) container.InspectResponse {
	c := inspectWithState(status, nil)
	c.ContainerJSONBase.ID = id
	c.Config.Labels = map[string]string{
		managedByLabelTag.string(): "gco",
		kindLabelTag.string():      string(resource.ApplicationKind),
		nameLabelTag.string():      name,
		hashLabelTag.string():      hash,
	}
	return c
}

/* rollingUpdate */

func TestProvider_rollingUpdate_blueGreenKeepsOldRunning(t *testing.T) {
	client := newOrderedTestClient()
	client.containerCreateReturnId = "new_id"
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", nil),
	}

	app := updateTestApp()
	provider := &Provider{
		client:         client,
		settleDuration: 5 * time.Millisecond,
		failedUpdates: map[string]updateFailure{
			// an expired failure memo for the same hash must not block the update
			app.Name: {hash: app.CalculateHash(), retryAt: time.Now().Add(-time.Minute)},
		},
	}

	// the old container publishes nothing, both versions can run at once
	old := &internalContainer{id: "old_id"}

	err := provider.rollingUpdate(context.Background(), old, app)
	assert.Nil(t, err, "should not have thrown an error")

	// the old container is never stopped, only removed once the new one settled
	assert.Equal(t, []string{"create", "start:new_id", "remove:old_id"}, client.sequence)
	assert.Equal(t, 0, len(client.containerStopArgs), "should not stop the old container")

	_, found := provider.failedUpdates[app.Name]
	assert.False(t, found, "should have cleared the failure memo after a successful update")
}

func TestProvider_rollingUpdate_sharedPortStopsOldFirst(t *testing.T) {
	client := newOrderedTestClient()
	client.containerCreateReturnId = "new_id"
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", nil),
	}

	provider := &Provider{client: client, settleDuration: 5 * time.Millisecond}

	// the old container publishes the same host port as the new one
	old := &internalContainer{
		id:    "old_id",
		ports: []containerPort{newContainerPort(80, 8080, "tcp")},
	}

	err := provider.rollingUpdate(context.Background(), old, updateTestApp())
	assert.Nil(t, err, "should not have thrown an error")

	// the old container is stopped after the replacement was created (image
	// pulled) but before the replacement is started
	assert.Equal(t, []string{"create", "stop:old_id", "start:new_id", "remove:old_id"}, client.sequence)
}

func TestProvider_rollingUpdate_rollbackRestartsStoppedOld(t *testing.T) {
	client := newOrderedTestClient()
	client.containerCreateReturnId = "new_id"

	// the replacement exits during the settle period
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("exited", nil),
	}

	app := updateTestApp()
	provider := &Provider{client: client, settleDuration: 5 * time.Millisecond}

	old := &internalContainer{
		id:    "old_id",
		ports: []containerPort{newContainerPort(80, 8080, "tcp")},
	}

	err := provider.rollingUpdate(context.Background(), old, app)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "rolled back")
	}

	// the failed replacement is removed and the stopped old container restarted
	assert.Equal(t, []string{"create", "stop:old_id", "start:new_id", "remove:new_id", "start:old_id"}, client.sequence)

	failure, found := provider.failedUpdates[app.Name]
	if assert.True(t, found, "should have recorded the failed update") {
		assert.Equal(t, app.CalculateHash(), failure.hash)
		assert.True(t, failure.retryAt.After(time.Now()), "the backoff window should lie in the future")
	}
}

func TestProvider_rollingUpdate_rollbackKeepsRunningOld(t *testing.T) {
	client := newOrderedTestClient()
	client.containerCreateReturnId = "new_id"
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("exited", nil),
	}

	app := updateTestApp()
	provider := &Provider{client: client, settleDuration: 5 * time.Millisecond}

	// nothing is shared, the old container keeps running during the update
	old := &internalContainer{id: "old_id"}

	err := provider.rollingUpdate(context.Background(), old, app)
	assert.NotNil(t, err, "should have thrown an error")

	// only the failed replacement is removed, the old container is untouched
	assert.Equal(t, []string{"create", "start:new_id", "remove:new_id"}, client.sequence)

	_, found := provider.failedUpdates[app.Name]
	assert.True(t, found, "should have recorded the failed update")
}

func TestProvider_rollingUpdate_stopError(t *testing.T) {
	client := newOrderedTestClient()
	client.containerCreateReturnId = "new_id"
	client.containerStopReturn = errors.New("test error")

	app := updateTestApp()
	provider := &Provider{client: client, settleDuration: 5 * time.Millisecond}

	old := &internalContainer{
		id:    "old_id",
		ports: []containerPort{newContainerPort(80, 8080, "tcp")},
	}

	err := provider.rollingUpdate(context.Background(), old, app)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "failed to stop old container")
	}

	// the stop may have partially taken effect, so the rollback tries to
	// restore the old container
	assert.Equal(t, []string{"create", "stop:old_id", "remove:new_id", "start:old_id"}, client.sequence)

	_, found := provider.failedUpdates[app.Name]
	assert.True(t, found, "should have recorded the failed update")
}

func TestProvider_rollingUpdate_createError(t *testing.T) {
	client := newOrderedTestClient()
	client.containerCreateReturnErr = errors.New("test error")

	app := updateTestApp()
	provider := &Provider{client: client, settleDuration: 5 * time.Millisecond}

	old := &internalContainer{id: "old_id"}

	err := provider.rollingUpdate(context.Background(), old, app)
	assert.NotNil(t, err, "should have thrown an error")

	// the old container is never touched when the replacement cannot be created
	assert.Equal(t, []string{"create"}, client.sequence)
	assert.Equal(t, 0, len(client.containerStartArgs))
	assert.Equal(t, 0, len(client.containerStopArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))

	_, found := provider.failedUpdates[app.Name]
	assert.True(t, found, "should have recorded the failed update")
}

/* failure memo */

func TestProvider_skipRecentlyFailed(t *testing.T) {
	provider := &Provider{}

	assert.Nil(t, provider.skipRecentlyFailed("web", "hash"),
		"should not skip without a recorded failure")

	provider.failedUpdates = map[string]updateFailure{
		"web": {hash: "hash", retryAt: time.Now().Add(time.Minute)},
	}

	assert.NotNil(t, provider.skipRecentlyFailed("web", "hash"),
		"should skip the same hash within the backoff window")
	assert.Nil(t, provider.skipRecentlyFailed("web", "other-hash"),
		"should not hold back a different hash")
	assert.Nil(t, provider.skipRecentlyFailed("api", "hash"),
		"should not hold back a different application")

	provider.failedUpdates["web"] = updateFailure{hash: "hash", retryAt: time.Now().Add(-time.Minute)}
	assert.Nil(t, provider.skipRecentlyFailed("web", "hash"),
		"should retry once the backoff window expired")
}

func TestProvider_recordUpdateFailure_initialisesMap(t *testing.T) {
	provider := &Provider{}
	provider.recordUpdateFailure("web", "hash")

	failure, found := provider.failedUpdates["web"]
	if assert.True(t, found, "should have recorded the failure on a nil map") {
		assert.Equal(t, "hash", failure.hash)
		assert.True(t, failure.retryAt.After(time.Now()), "retryAt should lie in the future")
	}
}

func TestProvider_UpdateApplication_failedUpdateBacksOff(t *testing.T) {
	client := NewTestClient()
	client.containerCreateReturnId = "new_id"
	client.containerListReturnContainers = []container.Summary{{ID: "old_id"}}

	app := updateTestApp()
	provider := &Provider{
		client:         client,
		settleDuration: 2 * time.Millisecond,
		updateBackoff:  200 * time.Millisecond,
	}

	// first update: resolves the running container, the replacement exits
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("old_id", "running", app.Name, "old-hash"),
		inspectWithState("exited", nil),
	}

	err := provider.UpdateApplication(context.Background(), app)
	assert.NotNil(t, err, "should have thrown an error")
	assert.Equal(t, 1, len(client.containerCreateArgs))
	if assert.Equal(t, 1, len(client.containerRemoveArgs), "rollback should only remove the replacement") {
		assert.Equal(t, "new_id", client.containerRemoveArgs[0][1])
	}

	// an immediate retry with the same configuration is refused before
	// touching the running container
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("old_id", "running", app.Name, "old-hash"),
	}

	err = provider.UpdateApplication(context.Background(), app)
	if assert.NotNil(t, err, "should have thrown an error during the backoff window") {
		assert.Contains(t, err.Error(), "failed recently")
	}
	assert.Equal(t, 1, len(client.containerCreateArgs), "should not have created another container")
	assert.Equal(t, 1, len(client.containerRemoveArgs), "should not have removed another container")

	// after the backoff expired the same configuration is retried
	time.Sleep(300 * time.Millisecond)

	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("old_id", "running", app.Name, "old-hash"),
		inspectWithState("running", nil),
	}

	err = provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should have retried after the backoff expired")
	assert.Equal(t, 2, len(client.containerCreateArgs))
	if assert.Equal(t, 2, len(client.containerRemoveArgs), "should have removed the old container") {
		assert.Equal(t, "old_id", client.containerRemoveArgs[1][1])
	}

	_, found := provider.failedUpdates[app.Name]
	assert.False(t, found, "should have cleared the failure memo")
}

func TestProvider_UpdateApplication_differentHashRetriesImmediately(t *testing.T) {
	client := NewTestClient()
	client.containerCreateReturnId = "new_id"
	client.containerListReturnContainers = []container.Summary{{ID: "old_id"}}

	app := updateTestApp()
	provider := &Provider{
		client:         client,
		settleDuration: 2 * time.Millisecond,
		failedUpdates: map[string]updateFailure{
			// a failure for a different desired hash is still backing off
			app.Name: {hash: "some-other-hash", retryAt: time.Now().Add(time.Hour)},
		},
	}

	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("old_id", "running", app.Name, "old-hash"),
		inspectWithState("running", nil),
	}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "a different configuration should not be held back by the memo")
	assert.Equal(t, 1, len(client.containerCreateArgs))
	if assert.Equal(t, 1, len(client.containerRemoveArgs)) {
		assert.Equal(t, "old_id", client.containerRemoveArgs[0][1])
	}
}

/* awaitHealthy */

func TestProvider_awaitHealthy_noHealthCheckSettles(t *testing.T) {
	client := NewTestClient()
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", nil),
	}

	provider := &Provider{client: client, settleDuration: 5 * time.Millisecond}

	err := provider.awaitHealthy(context.Background(), "container_id", nil)
	assert.Nil(t, err, "should not have thrown an error")
	assert.Equal(t, 1, len(client.containerInspectArgs), "should inspect once after the settle period")
}

func TestProvider_awaitHealthy_noHealthCheckExitedDuringStartup(t *testing.T) {
	client := NewTestClient()
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("exited", nil),
	}

	provider := &Provider{client: client, settleDuration: 5 * time.Millisecond}

	err := provider.awaitHealthy(context.Background(), "container_id", nil)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "exited during startup")
	}
}

func TestProvider_awaitHealthy_imageHealthCheckIsHonoured(t *testing.T) {
	client := NewTestClient()

	// no health check in the spec, but the image defines one
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", &container.Health{Status: "starting"}),
		inspectWithState("running", &container.Health{Status: "healthy"}),
	}

	provider := &Provider{
		client:             client,
		settleDuration:     2 * time.Millisecond,
		healthPollInterval: time.Millisecond,
	}

	err := provider.awaitHealthy(context.Background(), "container_id", nil)
	assert.Nil(t, err, "should not have thrown an error")
	assert.Equal(t, 2, len(client.containerInspectArgs), "should have polled until the health status resolved")
}

func TestProvider_awaitHealthy_healthy(t *testing.T) {
	client := NewTestClient()

	// the health status is not reported at first, then starting, then healthy;
	// a restarting container is tolerated while waiting
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", nil),
		inspectWithState("restarting", &container.Health{Status: "starting"}),
		inspectWithState("running", &container.Health{Status: "healthy"}),
	}

	provider := &Provider{client: client, healthPollInterval: time.Millisecond}

	hc := &resource.HealthCheck{Test: []string{"CMD", "true"}}
	err := provider.awaitHealthy(context.Background(), "container_id", hc)
	assert.Nil(t, err, "should not have thrown an error")
	assert.Equal(t, 3, len(client.containerInspectArgs))
}

func TestProvider_awaitHealthy_unhealthy(t *testing.T) {
	client := NewTestClient()
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", &container.Health{Status: "starting"}),
		inspectWithState("running", &container.Health{Status: "unhealthy"}),
	}

	provider := &Provider{client: client, healthPollInterval: time.Millisecond}

	hc := &resource.HealthCheck{Test: []string{"CMD", "true"}}
	err := provider.awaitHealthy(context.Background(), "container_id", hc)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "unhealthy")
	}
}

func TestProvider_awaitHealthy_exitedBeforeHealthy(t *testing.T) {
	client := NewTestClient()
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("exited", nil),
	}

	provider := &Provider{client: client, healthPollInterval: time.Millisecond}

	hc := &resource.HealthCheck{Test: []string{"CMD", "true"}}
	err := provider.awaitHealthy(context.Background(), "container_id", hc)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "exited before becoming healthy")
	}
}

func TestProvider_awaitHealthy_contextCancelled(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client, healthPollInterval: 2 * time.Millisecond}

	// the inspect queue is exhausted: every poll sees a response without a
	// state, so only the context can end the wait
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	hc := &resource.HealthCheck{Test: []string{"CMD", "true"}}
	err := provider.awaitHealthy(ctx, "container_id", hc)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestProvider_awaitHealthy_deadline(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the real minimum health deadline (~14s)")
	}

	client := NewTestClient()

	// the inspect queue is exhausted: every poll sees a response without a
	// state, so only the derived deadline can end the wait
	provider := &Provider{client: client, healthPollInterval: 100 * time.Millisecond}

	// the smallest possible deadline: (1s+1s)*(1+1) + the fixed 10s buffer
	hc := &resource.HealthCheck{
		Test:            []string{"CMD", "true"},
		IntervalSeconds: 1,
		TimeoutSeconds:  1,
		Retries:         1,
	}

	err := provider.awaitHealthy(context.Background(), "container_id", hc)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "did not become healthy in time")
	}
}

/* healthDeadline */

func TestHealthDeadline(t *testing.T) {
	// explicit values: start + (interval+timeout)*(retries+1) + 10s buffer
	hc := &resource.HealthCheck{
		IntervalSeconds:    5,
		TimeoutSeconds:     2,
		Retries:            2,
		StartPeriodSeconds: 10,
	}
	assert.Equal(t, 10*time.Second+7*time.Second*3+10*time.Second, healthDeadline(hc))

	// docker defaults are used for unset fields
	expected := (defaultHealthInterval+defaultHealthTimeout)*time.Duration(defaultHealthRetries+1) + 10*time.Second
	assert.Equal(t, expected, healthDeadline(nil))
	assert.Equal(t, expected, healthDeadline(&resource.HealthCheck{}))

	// the deadline is capped at five minutes
	long := &resource.HealthCheck{
		IntervalSeconds:    1,
		TimeoutSeconds:     1,
		Retries:            1,
		StartPeriodSeconds: 3600,
	}
	assert.Equal(t, maxHealthDeadline, healthDeadline(long))
}

/* sharesResourcesWith */

func TestInternalContainer_sharesResourcesWith_ports(t *testing.T) {
	oldJson := inspectWithState("running", nil)
	oldJson.HostConfig.PortBindings = nat.PortMap{
		"80/tcp": {{HostIP: "0.0.0.0", HostPort: "8080"}},
	}
	old := fromDockerContainer(oldJson)

	// the desired application publishes the same host port 8080
	next, err := fromApplicationResource(updateTestApp())
	assert.Nil(t, err, "should not have thrown an error")

	assert.True(t, old.sharesResourcesWith(next), "the same published host port should conflict")
	assert.True(t, next.sharesResourcesWith(&old), "the conflict should be symmetric")

	disjoint := &internalContainer{ports: []containerPort{newContainerPort(80, 9090, "tcp")}}
	assert.False(t, old.sharesResourcesWith(disjoint), "different host ports should not conflict")

	unpublished := &internalContainer{ports: []containerPort{newContainerPort(80, 0, "tcp")}}
	other := &internalContainer{ports: []containerPort{newContainerPort(81, 0, "tcp")}}
	assert.False(t, unpublished.sharesResourcesWith(other), "unpublished ports should never conflict")
}

func TestInternalContainer_sharesResourcesWith_volumes(t *testing.T) {
	writableJson := inspectWithState("running", nil)
	writableJson.HostConfig.Binds = []string{"/data:/var/lib/data"}
	writable := fromDockerContainer(writableJson)

	// the same source with at least one writable side conflicts
	readonlyMount := &internalContainer{
		volumes: []volumeMount{{source: "/data", destination: "/var/lib/data", readonly: true}},
	}
	assert.True(t, writable.sharesResourcesWith(readonlyMount), "a writable shared source should conflict")
	assert.True(t, readonlyMount.sharesResourcesWith(&writable), "the conflict should be symmetric")

	// both sides readonly is fine
	readonlyJson := inspectWithState("running", nil)
	readonlyJson.HostConfig.Binds = []string{"/data:/var/lib/data:ro"}
	readonly := fromDockerContainer(readonlyJson)
	assert.False(t, readonly.sharesResourcesWith(readonlyMount), "two readonly mounts should not conflict")

	// different sources are fine
	other := &internalContainer{
		volumes: []volumeMount{{source: "/other", destination: "/var/lib/data", readonly: false}},
	}
	assert.False(t, writable.sharesResourcesWith(other), "different sources should not conflict")
}

/* UpdateApplication housekeeping */

func TestProvider_UpdateApplication_removesStaleContainers(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "running_id"}, {ID: "stale_id"}}

	app := updateTestApp()
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("running_id", "running", app.Name, app.CalculateHash()),
		inspectForApp("stale_id", "exited", app.Name, "older-hash"),
	}

	provider := &Provider{client: client}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	// the exited leftover is removed, the converged running container is kept
	if assert.Equal(t, 1, len(client.containerRemoveArgs)) {
		assert.Equal(t, "stale_id", client.containerRemoveArgs[0][1])
	}
	assert.Equal(t, 0, len(client.containerCreateArgs), "a matching hash should not trigger an update")
	assert.Equal(t, 0, len(client.containerStopArgs))
}

func TestProvider_UpdateApplication_convergedIsNoop(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "running_id"}}

	app := updateTestApp()
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("running_id", "running", app.Name, app.CalculateHash()),
	}

	provider := &Provider{client: client}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
	assert.Equal(t, 0, len(client.containerStopArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))
}

func TestProvider_UpdateApplication_recreatesWhenNothingRuns(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "stale_id"}}

	app := updateTestApp()

	// only an exited leftover exists (e.g. after a crash): it is removed and
	// the application recreated, even though its hash label matches
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("stale_id", "exited", app.Name, app.CalculateHash()),
	}

	provider := &Provider{client: client}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(client.containerRemoveArgs)) {
		assert.Equal(t, "stale_id", client.containerRemoveArgs[0][1])
	}
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))
	assert.Equal(t, 0, len(client.containerStopArgs))
}

/* RemoveApplication and ActualState */

func TestProvider_RemoveApplication_removesAllMatches(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "one"}, {ID: "two"}}

	app := updateTestApp()
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("one", "running", app.Name, "h1"),
		inspectForApp("two", "exited", app.Name, "h2"),
	}

	provider := &Provider{client: client}

	err := provider.RemoveApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 2, len(client.containerRemoveArgs), "should remove every matching container") {
		assert.Equal(t, "one", client.containerRemoveArgs[0][1])
		assert.Equal(t, "two", client.containerRemoveArgs[1][1])
	}
}

func TestProvider_ActualState_prefersRunningDuplicate(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "stale_id"}, {ID: "live_id"}}

	// the same list is replayed for the application and the feature lookup
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("stale_id", "exited", "web", "h1"),
		inspectForApp("live_id", "running", "web", "h2"),
		inspectForApp("stale_id", "exited", "web", "h1"),
		inspectForApp("live_id", "running", "web", "h2"),
	}

	provider := &Provider{client: client}

	spec, err := provider.ActualState(context.Background())
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(spec.Applications), "duplicate names should collapse into one application") {
		assert.Equal(t, "web", spec.Applications[0].Name)
		assert.Equal(t, 1, spec.Applications[0].Instances, "the running container should win")
	}
}

func TestProvider_ActualState_prefersRunningDuplicate_runningListedFirst(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "live_id"}, {ID: "stale_id"}}

	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("live_id", "running", "web", "h2"),
		inspectForApp("stale_id", "exited", "web", "h1"),
		inspectForApp("live_id", "running", "web", "h2"),
		inspectForApp("stale_id", "exited", "web", "h1"),
	}

	provider := &Provider{client: client}

	spec, err := provider.ActualState(context.Background())
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(spec.Applications), "duplicate names should collapse into one application") {
		assert.Equal(t, 1, spec.Applications[0].Instances, "the running container should win regardless of order")
	}
}
