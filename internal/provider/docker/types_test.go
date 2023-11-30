package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"revengy.io/gco/agent/pkg/resource"
	"strings"
	"testing"
)

func exampleDockerContainer() types.Container {
	return types.Container{
		ID:     "container_id",
		Names:  []string{"container_name"},
		Image:  "nginx:latest",
		Labels: map[string]string{"key": "value"},
		State:  "running",
		Ports: []types.Port{
			{Type: "tcp", PrivatePort: 8080, PublicPort: 80},
			{Type: "tcp", PrivatePort: 9000, PublicPort: 9000},
		},
	}
}

func exampleDockerContainerJson() types.ContainerJSON {
	c := types.ContainerJSON{}
	c.ContainerJSONBase = &types.ContainerJSONBase{
		ID:   "container_id",
		Name: "container_name",
		State: &types.ContainerState{
			Status: "running",
		},
	}

	c.Config = &container.Config{
		Image: "nginx:latest",
		Labels: map[string]string{
			"key": "value",
		},
		ExposedPorts: map[nat.Port]struct{}{
			"80/tcp":   {},
			"9000/tcp": {},
		},
	}

	c.HostConfig = &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			"8080/tcp": {
				{HostPort: "80", HostIP: "0.0.0.0"},
			},
			"9000/tcp": {
				{HostPort: "9000", HostIP: "0.0.0.0"},
			},
		},
	}

	return c
}

func TestContainerPort_parse(t *testing.T) {
	port := newContainerPort(8080, 80, "tcp")
	assert.Equal(t, "8080:80/tcp", string(port))
	assert.Equal(t, "8080", port.privatePort(), "should have private port 8080")
	assert.Equal(t, "80", port.publicPort(), "should have public port 80")
	assert.Equal(t, "tcp", port.protocol(), "should have protocol tcp")
	assert.Equal(t, "8080/tcp", port.exposedPort(), "should have '8080/tcp' as exposed port")
}

func TestInternalContainer_fromDockerContainer_basic(t *testing.T) {
	con := exampleDockerContainerJson()

	c := fromDockerContainer(con)
	assert.Equal(t, "container_id", c.id)
	assert.Equal(t, "container_name", c.name)
	assert.Equal(t, "nginx:latest", c.image)
	assert.Equal(t, "value", c.labels["key"])

	if assert.Equal(t, 2, len(c.ports), "should have 2 ports") {
		web := c.ports[0]
		assert.Equal(t, "tcp", web.protocol())
		assert.Equal(t, "8080", web.privatePort())
		assert.Equal(t, "80", web.publicPort())

		metrics := c.ports[1]
		assert.Equal(t, "tcp", metrics.protocol())
		assert.Equal(t, "9000", metrics.privatePort())
		assert.Equal(t, "9000", metrics.publicPort())
	}
}

func TestInternalContainer_fromApplicationResource(t *testing.T) {
	application := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	c := fromApplicationResource(application)
	assert.Equal(t, "postgres", c.name)
	assert.Equal(t, "postgres:latest", c.image)
	assert.Equal(t, "5432", c.ports[0].publicPort())
	assert.Equal(t, "5432", c.ports[0].privatePort())
	assert.Equal(t, "tcp", c.ports[0].protocol())
}

func TestInternalContainer_toApplicationResource(t *testing.T) {
	original := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	ic := fromApplicationResource(original)
	parsed := ic.toApplicationResource()
	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.Image.Name, parsed.Image.Name)
	assert.Equal(t, original.Image.Tag, parsed.Image.Tag)
	assert.Equal(t, original.Ports[0].HostPort, parsed.Ports[0].HostPort)
	assert.Equal(t, original.Ports[0].ContainerPort, parsed.Ports[0].ContainerPort)
	assert.Equal(t, original.Ports[0].Protocol, parsed.Ports[0].Protocol)
}

func TestInternalContainer_fromApplicationResource_withLogConfig(t *testing.T) {
	application := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		LogConfig: &resource.LogConfig{
			Driver:   "fluentd",
			Disabled: false,
			Config: map[string]string{
				"address": "127.0.0.1:24224",
			},
		},
	}

	ic := fromApplicationResource(application)
	config := ic.hostConfig()

	if assert.NotNil(t, config, "should not have returned a nil config") {
		assert.Equal(t, "fluentd", config.LogConfig.Type)
		assert.Equal(t, "true", config.LogConfig.Config["fluentd-async"])
		assert.Equal(t, "127.0.0.1:24224", config.LogConfig.Config["fluentd-address"])

		allowedLabels := strings.Split(config.LogConfig.Config["labels"], ",")
		ShouldIncludeLabel(t, kindLabelTag.string(), allowedLabels)
		ShouldIncludeLabel(t, managedByLabelTag.string(), allowedLabels)
		ShouldIncludeLabel(t, nameLabelTag.string(), allowedLabels)
	}
}

func TestInternalContainer_fromDockerContainer_withBinds(t *testing.T) {
	con := exampleDockerContainerJson()
	con.HostConfig.Binds = []string{
		"/target:/source",
		"/target/readonly:/source/readonly:ro",
	}

	ic := fromDockerContainer(con)
	assert.Equal(t, 2, len(ic.volumes), "should have parsed two volume bindings")

	m1 := ic.volumes[0]
	assert.Equal(t, "/target", m1.source)
	assert.Equal(t, "/source", m1.destination)
	assert.False(t, m1.readonly)

	m2 := ic.volumes[1]
	assert.Equal(t, "/target/readonly", m2.source)
	assert.Equal(t, "/source/readonly", m2.destination)
	assert.True(t, m2.readonly)
}
