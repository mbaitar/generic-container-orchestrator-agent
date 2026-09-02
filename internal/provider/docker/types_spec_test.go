package docker

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/mbaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

func exampleSpecApplication() *resource.Application {
	return &resource.Application{
		Name: "web",
		Image: resource.Image{
			Name: "nginx",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 80, ContainerPort: 8080, Protocol: "tcp"},
		},
	}
}

func TestInternalContainer_fromApplicationResource_envSorted(t *testing.T) {
	app := exampleSpecApplication()
	app.Env = map[string]string{
		"ZEBRA":  "last",
		"ALPHA":  "first",
		"MIDDLE": "between",
	}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	expected := []string{
		"ALPHA=first",
		"MIDDLE=between",
		"ZEBRA=last",
	}
	assert.Equal(t, expected, ic.env, "env should be rendered sorted as K=V")
}

func TestInternalContainer_fromApplicationResource_customLabels(t *testing.T) {
	app := exampleSpecApplication()
	app.Labels = map[string]string{
		"com.example.team": "platform",
	}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	assert.Equal(t, "platform", ic.labels["com.example.team"], "custom label should be applied")
	assert.Equal(t, string(resource.ApplicationKind), ic.labels[kindLabelTag.string()])
	assert.Equal(t, "web", ic.labels[nameLabelTag.string()])
}

func TestInternalContainer_fromApplicationResource_reservedLabelsWin(t *testing.T) {
	app := exampleSpecApplication()
	app.Labels = map[string]string{
		kindLabelTag.string(): "spoofed-kind",
		nameLabelTag.string(): "spoofed-name",
		hashLabelTag.string(): "spoofed-hash",
	}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	assert.Equal(t, string(resource.ApplicationKind), ic.labels[kindLabelTag.string()], "kind label should not be overridable")
	assert.Equal(t, "web", ic.labels[nameLabelTag.string()], "name label should not be overridable")
	assert.Equal(t, app.CalculateHash(), ic.labels[hashLabelTag.string()], "hash label should not be overridable")
}

func TestInternalContainer_fromApplicationResource_hashLabel(t *testing.T) {
	app := exampleSpecApplication()
	app.Env = map[string]string{"KEY": "value"}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	assert.Equal(t, app.CalculateHash(), ic.labels[hashLabelTag.string()], "hash label should equal the application hash")
}

func TestInternalContainer_fromApplicationResource_volumes(t *testing.T) {
	app := exampleSpecApplication()
	app.Volumes = []resource.Volume{
		{Source: "/host/data", Destination: "/data"},
		{Source: "/host/config", Destination: "/etc/config", ReadOnly: true},
	}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	if assert.Equal(t, 2, len(ic.volumes), "should have mapped two volumes") {
		assert.Equal(t, "/host/data", ic.volumes[0].source)
		assert.Equal(t, "/data", ic.volumes[0].destination)
		assert.False(t, ic.volumes[0].readonly)

		assert.Equal(t, "/host/config", ic.volumes[1].source)
		assert.Equal(t, "/etc/config", ic.volumes[1].destination)
		assert.True(t, ic.volumes[1].readonly)
	}

	binds := ic.hostConfig().Binds
	if assert.Equal(t, 2, len(binds), "should have two bind entries") {
		assert.Equal(t, "/host/data:/data", binds[0])
		assert.Equal(t, "/host/config:/etc/config:ro", binds[1])
	}
}

func TestInternalContainer_fromApplicationResource_networkAndRestart(t *testing.T) {
	app := exampleSpecApplication()
	app.NetworkMode = "host"
	app.RestartPolicy = resource.RestartPolicyUnlessStopped

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	hostConfig := ic.hostConfig()
	assert.Equal(t, container.NetworkMode("host"), hostConfig.NetworkMode)
	assert.Equal(t, container.RestartPolicyMode("unless-stopped"), hostConfig.RestartPolicy.Name)
}

func TestInternalContainer_fromApplicationResource_noRestartPolicy(t *testing.T) {
	app := exampleSpecApplication()

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	hostConfig := ic.hostConfig()
	assert.Equal(t, container.RestartPolicy{}, hostConfig.RestartPolicy, "should not set a restart policy by default")
}

func TestInternalContainer_fromApplicationResource_resources(t *testing.T) {
	app := exampleSpecApplication()
	app.Resources = &resource.Resources{
		Memory: "256m",
		Cpus:   0.5,
	}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	hostConfig := ic.hostConfig()
	assert.Equal(t, int64(268435456), hostConfig.Resources.Memory, "256m should equal 268435456 bytes")
	assert.Equal(t, int64(500000000), hostConfig.Resources.NanoCPUs, "0.5 cpus should equal 500000000 NanoCPUs")
}

func TestInternalContainer_fromApplicationResource_invalidMemory(t *testing.T) {
	app := exampleSpecApplication()
	app.Resources = &resource.Resources{
		Memory: "lots-of-ram",
	}

	ic, err := fromApplicationResource(app)
	assert.NotNil(t, err, "should have returned an error for an invalid memory limit")
	assert.Nil(t, ic, "should not have returned a container")
}

func TestInternalContainer_fromApplicationResource_healthCheck(t *testing.T) {
	app := exampleSpecApplication()
	app.HealthCheck = &resource.HealthCheck{
		Test:               []string{"CMD", "curl", "-f", "http://localhost/"},
		IntervalSeconds:    10,
		TimeoutSeconds:     5,
		Retries:            3,
		StartPeriodSeconds: 30,
	}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	if assert.NotNil(t, ic.healthCheck, "should have mapped the health check") {
		assert.Equal(t, []string{"CMD", "curl", "-f", "http://localhost/"}, ic.healthCheck.Test)
		assert.Equal(t, 10*time.Second, ic.healthCheck.Interval)
		assert.Equal(t, 5*time.Second, ic.healthCheck.Timeout)
		assert.Equal(t, 30*time.Second, ic.healthCheck.StartPeriod)
		assert.Equal(t, 3, ic.healthCheck.Retries)
	}
}

func TestInternalContainer_config_includesEnvAndHealthcheck(t *testing.T) {
	app := exampleSpecApplication()
	app.Env = map[string]string{"KEY": "value"}
	app.HealthCheck = &resource.HealthCheck{
		Test:            []string{"CMD-SHELL", "true"},
		IntervalSeconds: 15,
	}

	ic, err := fromApplicationResource(app)
	assert.Nil(t, err, "should not have returned an error")

	config := ic.config()
	assert.Equal(t, []string{"KEY=value"}, config.Env, "config should carry the env")
	if assert.NotNil(t, config.Healthcheck, "config should carry the health check") {
		assert.Equal(t, []string{"CMD-SHELL", "true"}, config.Healthcheck.Test)
		assert.Equal(t, 15*time.Second, config.Healthcheck.Interval)
	}
}

func TestInternalContainer_toApplicationResource_storedHash(t *testing.T) {
	con := exampleDockerContainerJson()
	con.Config.Labels = map[string]string{
		nameLabelTag.string(): "web",
		hashLabelTag.string(): "stored-hash-value",
	}

	ic := fromDockerContainer(con)
	app := ic.toApplicationResource()

	assert.Equal(t, "stored-hash-value", app.CalculateHash(), "stored hash label should be authoritative")
}

func TestInternalContainer_toApplicationResource_withoutStoredHash(t *testing.T) {
	con := exampleDockerContainerJson()
	con.Config.Labels = map[string]string{
		nameLabelTag.string(): "web",
	}
	// keep a single port binding so the reconstructed port order is deterministic
	con.HostConfig.PortBindings = map[nat.Port][]nat.PortBinding{
		"8080/tcp": {
			{HostPort: "80", HostIP: "0.0.0.0"},
		},
	}

	ic := fromDockerContainer(con)
	app := ic.toApplicationResource()

	expected := resource.Application{
		Name: "web",
		Image: resource.Image{
			Name: "nginx",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 80, ContainerPort: 8080, Protocol: "tcp"},
		},
		Instances: 1,
	}

	assert.Equal(t, "web", app.Name)
	assert.Equal(t, expected.CalculateHash(), app.CalculateHash(), "hash should fall back to the reconstructed fields")
}
