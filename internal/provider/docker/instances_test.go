package docker

import (
	"context"
	"fmt"
	"strings"
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

// multiTestClient wraps the TestClient, hands out a distinct id for every
// created container and records the order of the lifecycle calls, so tests
// can assert capacity was grown before an old instance was retired.
type multiTestClient struct {
	*TestClient
	sequence []string
	created  int
}

func newMultiTestClient() *multiTestClient {
	return &multiTestClient{TestClient: NewTestClient()}
}

func (m *multiTestClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, containerName string) (container.CreateResponse, error) {
	if _, err := m.TestClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName); err != nil {
		return container.CreateResponse{}, err
	}

	m.created++
	id := fmt.Sprintf("new-%d", m.created)
	m.sequence = append(m.sequence, "create:"+id)
	return container.CreateResponse{ID: id}, nil
}

func (m *multiTestClient) ContainerStart(ctx context.Context, id string, opts container.StartOptions) error {
	m.sequence = append(m.sequence, "start:"+id)
	return m.TestClient.ContainerStart(ctx, id, opts)
}

func (m *multiTestClient) ContainerStop(ctx context.Context, id string, opts container.StopOptions) error {
	m.sequence = append(m.sequence, "stop:"+id)
	return m.TestClient.ContainerStop(ctx, id, opts)
}

func (m *multiTestClient) ContainerRemove(ctx context.Context, id string, opts container.RemoveOptions) error {
	m.sequence = append(m.sequence, "remove:"+id)
	return m.TestClient.ContainerRemove(ctx, id, opts)
}

// multiInstanceApp returns an application desiring several instances; it does
// not publish fixed host ports so the instances can run next to each other.
func multiInstanceApp(instances int) *resource.Application {
	return &resource.Application{
		Name:      "web",
		Image:     resource.Image{Name: "nginx", Tag: "latest"},
		Instances: instances,
	}
}

// instanceBaseName returns the '<name>-<hash8>' prefix instance names use.
func instanceBaseName(app *resource.Application) string {
	return fmt.Sprintf("%s-%s", app.Name, app.CalculateHash()[:8])
}

// inspectForInstance builds an inspect response for one instance of an
// application, carrying the docker container name the taken-name set is
// built from.
func inspectForInstance(id string, name string, status string, appName string, hash string) container.InspectResponse {
	c := inspectForApp(id, status, appName, hash)
	c.ContainerJSONBase.Name = "/" + name
	return c
}

/* createInstances */

func TestProvider_createInstances_createsAllOnManagedNetwork(t *testing.T) {
	client := newMultiTestClient()
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", nil),
		inspectWithState("running", nil),
		inspectWithState("running", nil),
	}

	app := multiInstanceApp(3)
	provider := &Provider{client: client, settleDuration: 2 * time.Millisecond}

	err := provider.CreateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	base := instanceBaseName(app)
	if assert.Equal(t, 3, len(client.containerCreateArgs), "should create every desired instance") {
		for i, args := range client.containerCreateArgs {
			assert.Equal(t, fmt.Sprintf("%s-%d", base, i+1), args[5],
				"instances should be numbered from one")

			config := args[1].(*container.Config)
			assert.Equal(t, app.Name, config.Labels[nameLabelTag.string()])
			assert.Equal(t, app.CalculateHash(), config.Labels[hashLabelTag.string()])

			hostConfig := args[2].(*container.HostConfig)
			assert.Equal(t, managedNetworkName, string(hostConfig.NetworkMode),
				"instances should default to the managed network")

			networking := args[3].(*network.NetworkingConfig)
			if assert.NotNil(t, networking, "instances should carry a networking config") {
				endpoint := networking.EndpointsConfig[managedNetworkName]
				if assert.NotNil(t, endpoint, "the endpoint should target the managed network") {
					assert.Equal(t, []string{app.Name}, endpoint.Aliases,
						"the DNS alias should equal the application name")
				}
			}
		}
	}

	assert.Equal(t, 3, len(client.containerStartArgs), "should start every instance")
	assert.Equal(t, 0, len(client.containerRemoveArgs), "should not remove anything")

	_, found := provider.failedUpdates[app.Name]
	assert.False(t, found, "should not have recorded a failure")
}

func TestProvider_createInstances_failureRemovesFailedKeepsHealthy(t *testing.T) {
	client := newMultiTestClient()

	// the first instance settles, the second exits during startup
	client.containerInspectReturn = []container.InspectResponse{
		inspectWithState("running", nil),
		inspectWithState("exited", nil),
	}

	app := multiInstanceApp(3)
	provider := &Provider{client: client, settleDuration: 2 * time.Millisecond}

	err := provider.CreateApplication(context.Background(), app)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "failed to scale application")
	}

	assert.Equal(t, 2, len(client.containerCreateArgs), "should stop creating after the failure")

	// only the failed second instance is removed, the healthy first one is kept
	if assert.Equal(t, 1, len(client.containerRemoveArgs)) {
		assert.Equal(t, "new-2", client.containerRemoveArgs[0][1])
	}

	failure, found := provider.failedUpdates[app.Name]
	if assert.True(t, found, "should have recorded the failed create") {
		assert.Equal(t, app.CalculateHash(), failure.hash)
		assert.True(t, failure.retryAt.After(time.Now()), "the backoff window should lie in the future")
	}
}

/* reconcileInstances */

func TestProvider_reconcileInstances_scaleUpSkipsTakenNames(t *testing.T) {
	client := newMultiTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "run-1"}, {ID: "run-2"}}

	app := multiInstanceApp(4)
	hash := app.CalculateHash()
	base := instanceBaseName(app)

	// two current instances run under the names '-1' and '-3'
	client.containerInspectReturn = []container.InspectResponse{
		inspectForInstance("run-1", base+"-1", "running", app.Name, hash),
		inspectForInstance("run-2", base+"-3", "running", app.Name, hash),
		inspectWithState("running", nil),
		inspectWithState("running", nil),
	}

	provider := &Provider{client: client, settleDuration: 2 * time.Millisecond}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	// the free names '-2' and '-4' are picked, the taken ones are skipped
	if assert.Equal(t, 2, len(client.containerCreateArgs), "should only create the missing instances") {
		assert.Equal(t, base+"-2", client.containerCreateArgs[0][5])
		assert.Equal(t, base+"-4", client.containerCreateArgs[1][5])
	}

	assert.Equal(t, 0, len(client.containerRemoveArgs), "should not remove anything while scaling up")
	assert.Equal(t, 0, len(client.containerStopArgs))
}

func TestProvider_reconcileInstances_scaleDownRemovesNewestFirst(t *testing.T) {
	client := newMultiTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "run-1"}, {ID: "run-2"}, {ID: "run-3"}, {ID: "run-4"},
	}

	app := multiInstanceApp(2)
	hash := app.CalculateHash()
	base := instanceBaseName(app)

	// docker lists the newest containers first
	client.containerInspectReturn = []container.InspectResponse{
		inspectForInstance("run-1", base+"-1", "running", app.Name, hash),
		inspectForInstance("run-2", base+"-2", "running", app.Name, hash),
		inspectForInstance("run-3", base+"-3", "running", app.Name, hash),
		inspectForInstance("run-4", base+"-4", "running", app.Name, hash),
	}

	provider := &Provider{client: client, settleDuration: 2 * time.Millisecond}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 0, len(client.containerCreateArgs), "should not create anything while scaling down")

	// the surplus instances listed first (the newest) are removed
	if assert.Equal(t, 2, len(client.containerRemoveArgs)) {
		assert.Equal(t, "run-1", client.containerRemoveArgs[0][1])
		assert.Equal(t, "run-2", client.containerRemoveArgs[1][1])
	}
}

func TestProvider_reconcileInstances_healReplacesExitedInstance(t *testing.T) {
	client := newMultiTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "run-1"}, {ID: "dead-2"}, {ID: "run-3"},
	}

	app := multiInstanceApp(3)
	hash := app.CalculateHash()
	base := instanceBaseName(app)

	// one instance exited, its hash still matches the desired configuration
	client.containerInspectReturn = []container.InspectResponse{
		inspectForInstance("run-1", base+"-1", "running", app.Name, hash),
		inspectForInstance("dead-2", base+"-2", "exited", app.Name, hash),
		inspectForInstance("run-3", base+"-3", "running", app.Name, hash),
		inspectWithState("running", nil),
	}

	provider := &Provider{client: client, settleDuration: 2 * time.Millisecond}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	// the exited instance is removed as stale and one replacement is created
	if assert.Equal(t, 1, len(client.containerRemoveArgs), "should only remove the exited instance") {
		assert.Equal(t, "dead-2", client.containerRemoveArgs[0][1])
	}

	if assert.Equal(t, 1, len(client.containerCreateArgs), "should scale back up to the desired count") {
		// the name of the removed instance stays taken during the reconcile
		assert.Equal(t, base+"-4", client.containerCreateArgs[0][5])
	}
}

func TestProvider_reconcileInstances_rollingReplaceKeepsCapacity(t *testing.T) {
	client := newMultiTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "old-1"}, {ID: "old-2"}, {ID: "old-3"},
	}

	app := multiInstanceApp(3)

	// every running instance carries an outdated configuration hash
	client.containerInspectReturn = []container.InspectResponse{
		inspectForInstance("old-1", "web-old-1", "running", app.Name, "old-hash"),
		inspectForInstance("old-2", "web-old-2", "running", app.Name, "old-hash"),
		inspectForInstance("old-3", "web-old-3", "running", app.Name, "old-hash"),
		inspectWithState("running", nil),
		inspectWithState("running", nil),
		inspectWithState("running", nil),
	}

	provider := &Provider{
		client:         client,
		settleDuration: 2 * time.Millisecond,
		failedUpdates: map[string]updateFailure{
			// an expired failure memo for the same hash must not block the update
			app.Name: {hash: app.CalculateHash(), retryAt: time.Now().Add(-time.Minute)},
		},
	}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	// every old instance is replaced one by one, growing capacity first
	assert.Equal(t, []string{
		"create:new-1", "start:new-1", "remove:old-1",
		"create:new-2", "start:new-2", "remove:old-2",
		"create:new-3", "start:new-3", "remove:old-3",
	}, client.sequence)

	// before every removal at least as many replacements were created, so the
	// running capacity never drops below the desired instance count
	creates, removes := 0, 0
	for _, step := range client.sequence {
		if strings.HasPrefix(step, "create:") {
			creates++
		}
		if strings.HasPrefix(step, "remove:") {
			removes++
			assert.GreaterOrEqual(t, creates, removes,
				"an old instance may only be retired after a replacement was created")
		}
	}

	_, found := provider.failedUpdates[app.Name]
	assert.False(t, found, "should have cleared the failure memo after a successful replace")
}

func TestProvider_reconcileInstances_replaceFailureKeepsOldInstances(t *testing.T) {
	client := newMultiTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "old-1"}, {ID: "old-2"}, {ID: "old-3"},
	}

	app := multiInstanceApp(3)

	// the first replacement exits during startup
	client.containerInspectReturn = []container.InspectResponse{
		inspectForInstance("old-1", "web-old-1", "running", app.Name, "old-hash"),
		inspectForInstance("old-2", "web-old-2", "running", app.Name, "old-hash"),
		inspectForInstance("old-3", "web-old-3", "running", app.Name, "old-hash"),
		inspectWithState("exited", nil),
	}

	provider := &Provider{client: client, settleDuration: 2 * time.Millisecond}

	err := provider.UpdateApplication(context.Background(), app)
	if assert.NotNil(t, err, "should have thrown an error") {
		assert.Contains(t, err.Error(), "rolled back")
	}

	// only the failed replacement is removed, every old instance keeps running
	assert.Equal(t, []string{"create:new-1", "start:new-1", "remove:new-1"}, client.sequence)

	failure, found := provider.failedUpdates[app.Name]
	if assert.True(t, found, "should have recorded the failed update") {
		assert.Equal(t, app.CalculateHash(), failure.hash)
	}
}

/* effectiveInstances and nextInstanceName */

func TestEffectiveInstances(t *testing.T) {
	assert.Equal(t, 1, effectiveInstances(&resource.Application{Instances: 0}),
		"a missing instance count should mean a single instance")
	assert.Equal(t, 1, effectiveInstances(&resource.Application{Instances: -1}),
		"a negative instance count should mean a single instance")
	assert.Equal(t, 3, effectiveInstances(&resource.Application{Instances: 3}))
}

func TestNextInstanceName_skipsTakenCandidates(t *testing.T) {
	app := multiInstanceApp(3)
	base := instanceBaseName(app)

	taken := make(map[string]struct{})
	assert.Equal(t, base+"-1", nextInstanceName(app, taken),
		"the first instance name should be numbered one")

	_, marked := taken[base+"-1"]
	assert.True(t, marked, "the picked name should be marked as taken")

	taken[base+"-2"] = struct{}{}
	taken[base+"-3"] = struct{}{}
	assert.Equal(t, base+"-4", nextInstanceName(app, taken),
		"taken candidates should be skipped")

	// freed names are reused, the lowest free number wins
	delete(taken, base+"-2")
	assert.Equal(t, base+"-2", nextInstanceName(app, taken))
}

/* ActualState */

func TestProvider_ActualState_countsRunningInstances(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "i1"}, {ID: "i2"}, {ID: "i3"},
	}

	// the same list is replayed for the application and the feature lookup
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("i1", "running", "web", "h1"),
		inspectForApp("i2", "running", "web", "h1"),
		inspectForApp("i3", "running", "web", "h1"),
		inspectForApp("i1", "running", "web", "h1"),
		inspectForApp("i2", "running", "web", "h1"),
		inspectForApp("i3", "running", "web", "h1"),
	}

	provider := &Provider{client: client}

	spec, err := provider.ActualState(context.Background())
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(spec.Applications), "instances should collapse into one application") {
		assert.Equal(t, "web", spec.Applications[0].Name)
		assert.Equal(t, 3, spec.Applications[0].Instances, "every running instance should be counted")
		assert.Equal(t, "h1", spec.Applications[0].CalculateHash())
	}

	assert.Equal(t, 0, len(client.containerRemoveArgs), "running instances should not be removed")
}

func TestProvider_ActualState_majorityHashWins(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "new-1"}, {ID: "old-1"}, {ID: "old-2"},
	}

	// a partial update left one new instance next to two old ones, the new
	// one is listed first (docker lists newest first)
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("new-1", "running", "web", "h-new"),
		inspectForApp("old-1", "running", "web", "h-old"),
		inspectForApp("old-2", "running", "web", "h-old"),
		inspectForApp("new-1", "running", "web", "h-new"),
		inspectForApp("old-1", "running", "web", "h-old"),
		inspectForApp("old-2", "running", "web", "h-old"),
	}

	provider := &Provider{client: client}

	spec, err := provider.ActualState(context.Background())
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(spec.Applications)) {
		assert.Equal(t, 3, spec.Applications[0].Instances,
			"instances of every version should be counted")
		assert.Equal(t, "inconsistent", spec.Applications[0].CalculateHash(),
			"mixed versions should force a reconcile instead of looking converged")
	}
}

func TestProvider_ActualState_hashTieGoesToNewest(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "new-1"}, {ID: "old-1"},
	}

	// one instance per version: the newest container is listed first and wins
	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("new-1", "running", "web", "h-new"),
		inspectForApp("old-1", "running", "web", "h-old"),
		inspectForApp("new-1", "running", "web", "h-new"),
		inspectForApp("old-1", "running", "web", "h-old"),
	}

	provider := &Provider{client: client}

	spec, err := provider.ActualState(context.Background())
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(spec.Applications)) {
		assert.Equal(t, 2, spec.Applications[0].Instances)
		assert.Equal(t, "inconsistent", spec.Applications[0].CalculateHash(),
			"mixed versions should force a reconcile instead of looking converged")
	}
}

func TestProvider_ActualState_omitsGroupWithoutRunningInstances(t *testing.T) {
	client := NewTestClient()
	client.containerListReturnContainers = []container.Summary{
		{ID: "dead-1"}, {ID: "dead-2"},
	}

	client.containerInspectReturn = []container.InspectResponse{
		inspectForApp("dead-1", "exited", "web", "h1"),
		inspectForApp("dead-2", "exited", "web", "h1"),
		inspectForApp("dead-1", "exited", "web", "h1"),
		inspectForApp("dead-2", "exited", "web", "h1"),
	}

	provider := &Provider{client: client}

	spec, err := provider.ActualState(context.Background())
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 0, len(spec.Applications),
		"an application without running instances should be omitted so it is recreated")

	// the exited leftovers are cleaned up
	if assert.Equal(t, 2, len(client.containerRemoveArgs)) {
		assert.Equal(t, "dead-1", client.containerRemoveArgs[0][1])
		assert.Equal(t, "dead-2", client.containerRemoveArgs[1][1])
	}
}

/* single instance path */

func TestProvider_UpdateApplication_singleInstanceKeepsStopFirstUpdate(t *testing.T) {
	client := newMultiTestClient()
	client.containerListReturnContainers = []container.Summary{{ID: "old_id"}}

	// a single instance application publishing a fixed host port
	app := updateTestApp()
	app.Instances = 1

	old := inspectForApp("old_id", "running", app.Name, "old-hash")
	old.HostConfig.PortBindings = nat.PortMap{
		"80/tcp": {{HostIP: "0.0.0.0", HostPort: "8080"}},
	}

	client.containerInspectReturn = []container.InspectResponse{
		old,
		inspectWithState("running", nil),
	}

	provider := &Provider{client: client, settleDuration: 2 * time.Millisecond}

	err := provider.UpdateApplication(context.Background(), app)
	assert.Nil(t, err, "should not have thrown an error")

	// the single instance path still runs the stop-first rolling update for a
	// shared host port; reconcileInstances would never stop a container
	assert.Equal(t, []string{"create:new-1", "stop:old_id", "start:new-1", "remove:old_id"}, client.sequence)

	// the replacement uses the plain rolling update name without an instance number
	if assert.Equal(t, 1, len(client.containerCreateArgs)) {
		assert.Equal(t, containerName(app), client.containerCreateArgs[0][5])
	}
}
