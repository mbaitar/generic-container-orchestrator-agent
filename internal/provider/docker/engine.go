package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"io"
	"revengy.io/gco/agent/internal/files"
	"revengy.io/gco/agent/internal/log"
	"revengy.io/gco/agent/pkg/resource"
)

// startContainer allows the provider to start a container using their known container id.
func (p *Provider) startContainer(id string) error {
	ctx := context.Background()
	return p.client.ContainerStart(ctx, id, types.ContainerStartOptions{})
}

// createContainer creates a new container and returns the container id if successful.
func (p *Provider) createContainer(c *internalContainer) (string, error) {
	ctx := context.Background()

	err := p.verifyImage(c.image, c.pullPolicy)
	if err != nil {
		log.Debugf("Unable to pull image '%s' for application '%s'", c.image, c.name)
		return "", err
	}

	// set required labels
	c.addLabel(managedByLabel())

	// set name label if not given
	if c.getLabel(nameLabelTag) == "" {
		c.addLabel(nameLabel(c.name))
	}

	if p.addComposeLabel {
		c.addLabel(composeProjectLabel())
	}

	config := c.config()
	hostConfig := c.hostConfig()

	body, err := p.client.ContainerCreate(ctx, config, hostConfig, nil, nil, c.name)
	return body.ID, err
}

// removeContainer removes a container on the system with the referenced id.
func (p *Provider) removeContainer(id string) error {
	ctx := context.Background()
	return p.client.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
}

// cleanUpBinds removes the bound volumes from the host if the container had any volumes configured.
func (p *Provider) cleanUpBinds(ic *internalContainer) {
	if len(ic.volumes) == 0 {
		return
	}

	log.Debugf("Cleaning up volume binds for container with id '%s'", ic.id)
	for _, volume := range ic.volumes {
		files.RemoveConfigFile(volume.source)
	}
}

// getFilteredContainers returns the list of filtered containers managed by the system.
func (p *Provider) getFilteredContainers(opts *types.ContainerListOptions) ([]internalContainer, error) {
	ctx := context.Background()

	if opts.Filters.Len() == 0 {
		opts.Filters = filters.NewArgs()
	}

	opts.Filters.Add("label", managedByLabel().string())
	containers, err := p.client.ContainerList(ctx, *opts)
	if err != nil {
		return nil, err
	}

	parsed := make([]internalContainer, len(containers))
	for i, c := range containers {
		// inject with bind info
		containerJson, inspectErr := p.client.ContainerInspect(ctx, c.ID)
		if inspectErr != nil {
			log.Warnf("Failed to inspect container=%s: %v", c.ID, inspectErr)
			continue
		}

		parsed[i] = fromDockerContainer(containerJson)
	}

	return parsed, nil
}

// getContainerByName searched for a container with a matching name label.
func (p *Provider) getContainerByName(name string) (*internalContainer, error) {
	opts := &types.ContainerListOptions{All: true}
	opts.Filters = filters.NewArgs()
	opts.Filters.Add("label", nameLabel(name).string())

	containers, err := p.getFilteredContainers(opts)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, nil
	} else {
		return &containers[0], nil
	}
}

// getFeatureByName searches for a feature with the matching name using the feature label.
func (p *Provider) getFeatureByName(name string) (*internalContainer, error) {
	opts := &types.ContainerListOptions{All: true}
	opts.Filters = filters.NewArgs()
	opts.Filters.Add("label", featureLabel(name).string())

	containers, err := p.getFilteredContainers(opts)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, nil
	} else {
		return &containers[0], nil
	}
}

// verifyImage verify if the image adheres to the requested imagePullPolicy.
func (p *Provider) verifyImage(image string, policy imagePullPolicy) error {
	ctx := context.Background()

	if policy == whenNotPresentPolicy {
		// check to see if image is present
		opts := types.ImageListOptions{}
		opts.Filters = filters.NewArgs()
		opts.Filters.Add("reference", image)

		result, err := p.client.ImageList(ctx, opts)
		if err != nil {
			return err
		}

		if len(result) >= 1 {
			// image is already present
			log.Debugf("Image '%s' found, not performing pull", image)
			return nil
		}
	}

	log.Debugf("Pulling image '%s' (policy=%s)", image, policy)
	reader, err := p.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if reader != nil {
		defer reader.Close()
	}
	if err != nil {
		return err
	}

	_, err = io.ReadAll(reader)
	if err != nil {
		return err
	}

	log.Debugf("Successfully pulled image '%s'", image)
	return nil
}

// getApplicationContainers searches for all containers which are known to be applications.
func (p *Provider) getApplicationContainers() ([]internalContainer, error) {
	opts := &types.ContainerListOptions{All: true}
	opts.Filters = filters.NewArgs()

	// append kind label filter
	opts.Filters.Add("label", kindLabel(resource.ApplicationKind).string())
	return p.getFilteredContainers(opts)
}

// getFeatureContainers searches for all containers which are known to be features.
func (p *Provider) getFeatureContainers() ([]internalContainer, error) {
	opts := &types.ContainerListOptions{All: true}
	opts.Filters = filters.NewArgs()

	// append kind label filter
	opts.Filters.Add("label", kindLabel(resource.FeatureKind).string())
	return p.getFilteredContainers(opts)
}
