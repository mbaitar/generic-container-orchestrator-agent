package docker

import (
	"context"
	"dsync.io/gco/agent/pkg/resource"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProvider_startContainer(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{
		client: client,
	}

	err := provider.startContainer("container_id")
	assert.Nil(t, err, "should not have returned an error")

	// verify arguments
	if assert.Equal(t, 1, len(client.containerStartArgs), "should have called client.ContainerStart()") {
		args := client.containerStartArgs[0]
		ctx := args[0].(context.Context)
		containerId := args[1].(string)
		opts := args[2].(types.ContainerStartOptions)

		assert.NotNilf(t, ctx, "should have included a context")
		assert.Equal(t, "container_id", containerId)
		assert.NotNilf(t, opts, "should have included ContainerStartOptions")
	}
}

func TestProvider_startContainer_clientError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{
		client: client,
	}

	client.containerStartReturn = errors.New("test error")
	err := provider.startContainer("container_id")
	assert.NotNilf(t, err, "should have returned an error")

	// verify arguments
	if assert.Equal(t, 1, len(client.containerStartArgs), "should have called client.ContainerStart()") {
		args := client.containerStartArgs[0]
		ctx := args[0].(context.Context)
		containerId := args[1].(string)
		opts := args[2].(types.ContainerStartOptions)

		assert.NotNilf(t, ctx, "should have included a context")
		assert.Equal(t, "container_id", containerId)
		assert.NotNilf(t, opts, "should have included ContainerStartOptions")
	}
}

func TestProvider_createContainer(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	con := &internalContainer{
		name:   "my-container",
		image:  "nginx:latest",
		labels: map[string]string{"custom-labelTag": "value"},
		ports: []containerPort{
			newContainerPort(8080, 80, "tcp"),
			newContainerPort(9000, 9000, "tcp"),
		},
	}

	client.containerCreateReturnId = "container_id"
	id, err := provider.createContainer(con)
	assert.Nil(t, err, "should not have thrown an error")
	assert.Equal(t, "container_id", id)

	if assert.Equal(t, 1, len(client.containerCreateArgs), "should have called client.ContainerCreate()") {
		args := client.containerCreateArgs[0]
		ctx := args[0].(context.Context)
		config := args[1].(*container.Config)
		hostConfig := args[2].(*container.HostConfig)
		//networkConfig := args[3].(*network.NetworkingConfig)
		//platform := args[4].(*v1.Platform)
		containerName := args[5].(string)

		assert.NotNilf(t, ctx, "should have included context")
		assert.Equal(t, con.name, containerName, "should not have an empty container name")

		if assert.NotNil(t, config, "should have included container config") {
			assert.Equal(t, con.image, config.Image)
			assert.Equal(t, con.labels, config.Labels)

			// verify that platform labels are present
			assert.Equal(t, "gco", con.labels[managedByLabelTag.string()])
			assert.Equal(t, "my-container", con.labels[nameLabelTag.string()])
		}

		if assert.NotNil(t, hostConfig, "should have included container host config") {
			assert.Equal(t, 2, len(hostConfig.PortBindings))

			webPort, _ := nat.NewPort("tcp", "8080")
			assert.Equal(t, "0.0.0.0", hostConfig.PortBindings[webPort][0].HostIP)
			assert.Equal(t, "80", hostConfig.PortBindings[webPort][0].HostPort)

			metricPort, _ := nat.NewPort("tcp", "9000")
			assert.Equal(t, "0.0.0.0", hostConfig.PortBindings[metricPort][0].HostIP)
			assert.Equal(t, "9000", hostConfig.PortBindings[metricPort][0].HostPort)
		}
	}
}

func TestProvider_createContainer_clientError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	con := &internalContainer{
		name:   "my-container",
		image:  "nginx:latest",
		labels: map[string]string{"custom-labelTag": "value"},
		ports: []containerPort{
			newContainerPort(8080, 80, "tcp"),
			newContainerPort(9000, 9000, "tcp"),
		},
	}

	client.containerCreateReturnId = ""
	client.containerCreateReturnErr = errors.New("test error")
	id, err := provider.createContainer(con)
	assert.NotNil(t, err, "should have thrown an error")
	assert.Equal(t, "", id)
}

func TestProvider_removeContainer(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	err := provider.removeContainer("container_id")
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(client.containerRemoveArgs), "should have called client.ContainerRemove()") {
		args := client.containerRemoveArgs[0]
		ctx := args[0].(context.Context)
		containerId := args[1].(string)
		opts := args[2].(types.ContainerRemoveOptions)

		assert.NotNil(t, ctx, "should have included context")
		assert.Equal(t, "container_id", containerId)
		assert.NotNilf(t, opts, "should have included remove options")
		assert.True(t, opts.Force, "should have toggled force flag")
	}
}

func TestProvider_removeContainer_clientError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.containerRemoveReturn = errors.New("test error")
	err := provider.removeContainer("container_id")
	assert.NotNil(t, err, "should have thrown an error")
}

func TestProvider_getFilteredContainers(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}
	containers, err := provider.getFilteredContainers(&types.ContainerListOptions{})
	assert.Nil(t, err, "should not have thrown an error")
	assert.NotNil(t, containers, "should not have returned a nil list")
	assert.Equal(t, 1, len(containers), "should have returned 1 container")

	// verify arguments
	if assert.Equal(t, 1, len(client.containerListArgs), "should have called client.ContainerList()") {
		args := client.containerListArgs[0]
		ctx := args[0].(context.Context)
		opts := args[1].(types.ContainerListOptions)

		assert.NotNil(t, ctx, "should have included context")
		if assert.NotNil(t, opts, "should have included list options") {

			// should include the managed by label
			if assert.NotNil(t, opts.Filters, "should include filters") {
				labels := opts.Filters.Get("label")
				if assert.Equal(t, 1, len(labels), "should include 1 label") {
					assert.Equal(t, managedByLabel().string(), labels[0])
				}
			}
		}
	}
}

func TestProvider_getFilteredContainers_clientError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.containerListReturnErr = errors.New("test error")
	containers, err := provider.getFilteredContainers(&types.ContainerListOptions{})
	assert.NotNil(t, err, "should have thrown an error")
	assert.Nil(t, containers, "should have returned a nil list")
}

func TestProvider_getContainerByName(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	dc := exampleDockerContainer()

	client.containerListReturnContainers = []types.Container{dc}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}

	c, err := provider.getContainerByName(dc.Names[0])
	assert.Nil(t, err, "should not have thrown an error")
	assert.NotNil(t, c, "should have returned a container")

	// verify args
	if assert.Equal(t, 1, len(client.containerListArgs), "should have called client.ContainerList()") {
		args := client.containerListArgs[0]
		opts := args[1].(types.ContainerListOptions)

		assert.True(t, opts.All, "should have enabled all flag")

		hasNameLabel, hasManagedByLabel := false, false
		labels := opts.Filters.Get("label")

		for _, l := range labels {
			if l == managedByLabel().string() {
				hasManagedByLabel = true
			}

			if l == nameLabel(dc.Names[0]).string() {
				hasNameLabel = true
			}
		}

		assert.True(t, hasNameLabel, "should have included the name label")
		assert.True(t, hasManagedByLabel, "should have included the managed by label")
	}
}

func TestProvider_getContainerByName_noMatch(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	c, err := provider.getContainerByName("no matches")
	assert.Nil(t, err, "should not have thrown an error")
	assert.Nil(t, c, "should not have returned a container")
}

func TestProvider_verifyImage_always(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	err := provider.verifyImage("postgres:latest", alwaysPullPolicy)
	assert.Nil(t, err, "should not have thrown an error")

	if assert.Equal(t, 1, len(client.imagePullArgs), "should have called client.ImagePull()") {
		args := client.imagePullArgs[0]
		ctx := args[0].(context.Context)
		ref := args[1].(string)
		//options := args[2].(types.ImagePullOptions)

		assert.NotNil(t, ctx, "should have included context")
		assert.Equal(t, "postgres:latest", ref)
	}
}

func TestProvider_getApplicationContainers(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	containers, err := provider.getApplicationContainers()
	assert.Nil(t, err, "should not have thrown an error")
	assert.NotNil(t, containers, "should have returned an array")

	if assert.Equal(t, 1, len(client.containerListArgs), "should have called client.ContainerList()") {
		args := client.containerListArgs[0]
		ctx := args[0].(context.Context)
		opts := args[1].(types.ContainerListOptions)

		assert.NotNil(t, ctx, "should have included a context")
		assert.True(t, opts.All, "should list all the containers")
		if assert.NotEqual(t, 0, opts.Filters.Len(), "should have included filters") {
			labels := opts.Filters.Get("label")

			managedFound, kindFound := false, false
			for _, value := range labels {
				if value == managedByLabel().string() {
					managedFound = true
					continue
				}

				if value == kindLabel(resource.ApplicationKind).string() {
					kindFound = true
					continue
				}
			}

			assert.True(t, managedFound, "should have included the managed-by label")
			assert.True(t, kindFound, "should have included the kind label")
		}
	}
}
