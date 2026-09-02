package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/mbaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

func TestProvider_getFilteredContainers_skipsFailedInspects(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	client.containerListReturnContainers = []container.Summary{exampleDockerContainer()}
	client.containerInspectErr = errors.New("inspect failed")

	containers, err := provider.getFilteredContainers(context.Background(), &container.ListOptions{})
	assert.Nil(t, err, "should not have returned an error")
	assert.Equal(t, 0, len(containers), "failed inspects should be skipped, not returned as zero values")
}

func TestProvider_CreateApplication_propagatesConversionError(t *testing.T) {
	client := NewTestClient()
	provider := &Provider{client: client}

	app := &resource.Application{
		Name:      "bad",
		Image:     resource.Image{Name: "nginx", Tag: "latest"},
		Resources: &resource.Resources{Memory: "12abc"},
	}

	err := provider.CreateApplication(context.Background(), app)
	assert.NotNil(t, err, "should have returned the conversion error")
	assert.Equal(t, 0, len(client.containerCreateArgs), "should not have attempted to create a container")
}
