package docker

import (
	"dsync.io/gco/agent/internal/files"
	"dsync.io/gco/agent/pkg/feature"
	"dsync.io/gco/agent/pkg/resource"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func SetupForTests(t *testing.T) {
	files.SetDirectory(t.TempDir())
}

func ShouldIncludeLabel(t *testing.T, label string, labels []string) {
	for _, l := range labels {
		if l == label {
			return
		}
	}

	t.Errorf("label '%s' not found in '%s'", label, strings.Join(labels, ","))
}

func TestProvider_CreateApplication(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	err := provider.CreateApplication(app)
	assert.Nil(t, err, "should not have thrown an error")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))
}

func TestProvider_CreateApplication_createError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.containerCreateReturnErr = errors.New("test error")

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	err := provider.CreateApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_CreateApplication_startError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.containerStartReturn = errors.New("test error")

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	err := provider.CreateApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))
}

func TestProvider_CreateApplication_pullError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.imagePullReturnErr = errors.New("test error")

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	err := provider.CreateApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_UpdateApplication(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}

	err := provider.UpdateApplication(app)
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))
}

func TestProvider_UpdateApplication_noMatchingContainer(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnContainers = []types.Container{}

	err := provider.UpdateApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_UpdateApplication_removeError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}
	client.containerRemoveReturn = errors.New("testing error")

	err := provider.UpdateApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_UpdateApplication_getContainerError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnErr = errors.New("testing error")

	err := provider.UpdateApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_RemoveApplication(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}

	err := provider.RemoveApplication(app)
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))
}

func TestProvider_RemoveApplication_notFound(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnContainers = []types.Container{}

	err := provider.RemoveApplication(app)
	assert.NotNil(t, err, "should have thrown an error when not found")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))
}

func TestProvider_RemoveApplication_findError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnErr = errors.New("test error")

	err := provider.RemoveApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))
}

func TestProvider_RemoveApplication_removeError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name: "postgres",
		Image: resource.Image{
			Name: "postgres",
			Tag:  "latest",
		},
		Ports: []resource.Port{
			{HostPort: 5432, ContainerPort: 5432, Protocol: "tcp"},
		},
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}

	client.containerRemoveReturn = errors.New("test error")

	err := provider.RemoveApplication(app)
	assert.NotNil(t, err, "should have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))
}

func TestProvider_ActualState(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	spec, err := provider.ActualState()
	assert.Nil(t, err, "should not have thrown")
	assert.NotNil(t, spec, "should have returned a state spec")

	assert.Equal(t, 2, len(client.containerListArgs))

	// first call for applications
	appCallArgs := client.containerListArgs[0]
	if assert.NotNil(t, appCallArgs) {
		opts := appCallArgs[1].(types.ContainerListOptions)
		assert.True(t, opts.All, "should have used the All flag")
		labels := opts.Filters.Get("label")
		ShouldIncludeLabel(t, "gco.io/kind=app", labels)
		ShouldIncludeLabel(t, "gco.io/managed-by=gco", labels)
	}

	// second call for features
	featCallArgs := client.containerListArgs[1]
	if assert.NotNil(t, featCallArgs) {
		opts := featCallArgs[1].(types.ContainerListOptions)
		assert.True(t, opts.All, "should have used the All flag")
		labels := opts.Filters.Get("label")
		ShouldIncludeLabel(t, "gco.io/kind=feature", labels)
		ShouldIncludeLabel(t, "gco.io/managed-by=gco", labels)
	}

}

func TestProvider_ActualState_listError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.containerListReturnErr = errors.New("test error")

	spec, err := provider.ActualState()
	assert.NotNil(t, err, "should have thrown")
	assert.Nil(t, spec, "should not have returned a state spec")

	assert.Equal(t, 1, len(client.containerListArgs))
}

func TestProvider_CreateFeature(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	err := provider.CreateFeature(fluentBit)
	assert.Nil(t, err, "should not have thrown an error")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))
}

func TestProvider_CreateFeature_createError(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerCreateReturnErr = errors.New("test error")
	err := provider.CreateFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error upon creation")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_CreateFeature_startError(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerStartReturn = errors.New("test error")
	err := provider.CreateFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error upon starting the container")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))
}

func TestProvider_CreateFeature_pullError(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.imagePullReturnErr = errors.New("test error")
	err := provider.CreateFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error when failing to pull image")

	// verify calls made
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_CreateFeature_unsupported(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := Provider{client: client}

	unsupported := &UnsupportedFeature{}

	err := provider.CreateFeature(unsupported)
	assert.NotNil(t, err, "should have thrown an error")

	// verify calls made
	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_UpdateFeature(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}

	err := provider.UpdateFeature(fluentBit)
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))

	// verify container list args used correct labels
	opts := client.containerListArgs[0][1].(types.ContainerListOptions)
	if assert.NotNil(t, opts, "should have used container list options") {
		labels := opts.Filters.Get("label")
		ShouldIncludeLabel(t, "gco.io/feature=fluent-bit", labels)
	}
}

func TestProvider_UpdateFeature_noMatchingContainer(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnContainers = []types.Container{}

	err := provider.UpdateFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error when not finding a matching container")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_UpdateFeature_removeError(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}
	client.containerRemoveReturn = errors.New("testing error")

	err := provider.UpdateFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error upon failing to remove")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_UpdateFeature_getFeatureError(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnErr = errors.New("testing error")

	err := provider.UpdateFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error when unable to get feature")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 0, len(client.imagePullArgs))
	assert.Equal(t, 0, len(client.containerCreateArgs))
	assert.Equal(t, 0, len(client.containerStartArgs))
}

func TestProvider_RemoveFeature(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}

	err := provider.RemoveFeature(fluentBit)
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))
}

func TestProvider_RemoveFeature_notFound(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnContainers = []types.Container{}

	err := provider.RemoveFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error when not found")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))
}

func TestProvider_RemoveFeature_findError(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnErr = errors.New("test error")

	err := provider.RemoveFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error when list command fails")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 0, len(client.containerRemoveArgs))
}

func TestProvider_removeFeature_removeError(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
	}

	client.containerListReturnContainers = []types.Container{exampleDockerContainer()}
	client.containerInspectReturn = []types.ContainerJSON{exampleDockerContainerJson()}
	client.containerRemoveReturn = errors.New("test error")

	err := provider.RemoveFeature(fluentBit)
	assert.NotNil(t, err, "should have thrown an error when failing to remove")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))
}
