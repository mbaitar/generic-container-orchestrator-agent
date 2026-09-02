package docker

import (
	"context"
	"errors"
	"github.com/docker/docker/api/types/container"
	"strings"
	"testing"

	"github.com/mbaitar/gco/agent/internal/files"
	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
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

	err := provider.CreateApplication(context.Background(), app)
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

	err := provider.CreateApplication(context.Background(), app)
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

	err := provider.CreateApplication(context.Background(), app)
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

	err := provider.CreateApplication(context.Background(), app)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}

	err := provider.UpdateApplication(context.Background(), app)
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

	client.containerListReturnContainers = []container.Summary{}

	err := provider.UpdateApplication(context.Background(), app)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}
	client.containerRemoveReturn = errors.New("testing error")

	err := provider.UpdateApplication(context.Background(), app)
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

	err := provider.UpdateApplication(context.Background(), app)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}

	err := provider.RemoveApplication(context.Background(), app)
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

	client.containerListReturnContainers = []container.Summary{}

	err := provider.RemoveApplication(context.Background(), app)
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

	err := provider.RemoveApplication(context.Background(), app)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}

	client.containerRemoveReturn = errors.New("test error")

	err := provider.RemoveApplication(context.Background(), app)
	assert.NotNil(t, err, "should have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))
}

func TestProvider_ActualState(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	spec, err := provider.ActualState(context.Background())
	assert.Nil(t, err, "should not have thrown")
	assert.NotNil(t, spec, "should have returned a state spec")

	assert.Equal(t, 2, len(client.containerListArgs))

	// first call for applications
	appCallArgs := client.containerListArgs[0]
	if assert.NotNil(t, appCallArgs) {
		opts := appCallArgs[1].(container.ListOptions)
		assert.True(t, opts.All, "should have used the All flag")
		labels := opts.Filters.Get("label")
		ShouldIncludeLabel(t, "gco.io/kind=app", labels)
		ShouldIncludeLabel(t, "gco.io/managed-by=gco", labels)
	}

	// second call for features
	featCallArgs := client.containerListArgs[1]
	if assert.NotNil(t, featCallArgs) {
		opts := featCallArgs[1].(container.ListOptions)
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

	spec, err := provider.ActualState(context.Background())
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

	err := provider.CreateFeature(context.Background(), fluentBit)
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
	err := provider.CreateFeature(context.Background(), fluentBit)
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
	err := provider.CreateFeature(context.Background(), fluentBit)
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
	err := provider.CreateFeature(context.Background(), fluentBit)
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

	err := provider.CreateFeature(context.Background(), unsupported)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}

	err := provider.UpdateFeature(context.Background(), fluentBit)
	assert.Nil(t, err, "should not have thrown an error")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))

	// all default create calls
	assert.Equal(t, 1, len(client.imagePullArgs))
	assert.Equal(t, 1, len(client.containerCreateArgs))
	assert.Equal(t, 1, len(client.containerStartArgs))

	// verify container list args used correct labels
	opts := client.containerListArgs[0][1].(container.ListOptions)
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

	client.containerListReturnContainers = []container.Summary{}

	err := provider.UpdateFeature(context.Background(), fluentBit)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}
	client.containerRemoveReturn = errors.New("testing error")

	err := provider.UpdateFeature(context.Background(), fluentBit)
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

	err := provider.UpdateFeature(context.Background(), fluentBit)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}

	err := provider.RemoveFeature(context.Background(), fluentBit)
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

	client.containerListReturnContainers = []container.Summary{}

	err := provider.RemoveFeature(context.Background(), fluentBit)
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

	err := provider.RemoveFeature(context.Background(), fluentBit)
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

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectReturn = []container.InspectResponse{exampleDockerContainerJson()}
	client.containerRemoveReturn = errors.New("test error")

	err := provider.RemoveFeature(context.Background(), fluentBit)
	assert.NotNil(t, err, "should have thrown an error when failing to remove")

	assert.Equal(t, 1, len(client.containerListArgs))
	assert.Equal(t, 1, len(client.containerRemoveArgs))
}
