package docker

import (
	docker "github.com/docker/docker/client"
	"revengy.io/gco/agent/internal/config"
	"revengy.io/gco/agent/internal/provider"
	"revengy.io/gco/agent/internal/state"
	"revengy.io/gco/agent/pkg/feature"
	"revengy.io/gco/agent/pkg/resource"
)

// Provider defines a docker provider which can communicate with the local docker socket.
type Provider struct {
	// client represents the Docker SDK client
	client docker.CommonAPIClient
	// addComposeLabel adds the docker compose project label.
	addComposeLabel bool
}

func NewDockerProvider() *Provider {
	client := newDockerClient()
	return &Provider{
		client:          client,
		addComposeLabel: false,
	}
}

func (p *Provider) WithConfig(conf config.DockerProvider) *Provider {
	p.addComposeLabel = conf.UseDockerComposeGrouping
	return p
}

func (p *Provider) CreateApplication(app *resource.Application) error {
	container := fromApplicationResource(app)

	// TODO: retrieve configuration hash for this resource?

	id, err := p.createContainer(container)
	if err != nil {
		return err
	}

	err = p.startContainer(id)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) UpdateApplication(app *resource.Application) error {
	if err := p.RemoveApplication(app); err != nil {
		return err
	}

	return p.CreateApplication(app)
}

func (p *Provider) RemoveApplication(app *resource.Application) error {
	container, err := p.getContainerByName(app.Name)
	if err != nil {
		return err
	}

	if container == nil {
		return provider.ErrAppNotFound
	} else {
		return p.removeContainer(container.id)
	}
}

func (p *Provider) CreateFeature(feat feature.Feature) error {

	var container *internalContainer

	switch v := feat.(type) {
	case *feature.FluentBit:
		{
			fluentBitContainer, err := p.createFluentBitContainer(v)
			if err != nil {
				return err
			}

			container = fluentBitContainer
		}
	default:
		return provider.ErrFeatureNotSupported
	}

	id, err := p.createContainer(container)
	if err != nil {
		return err
	}

	err = p.startContainer(id)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) UpdateFeature(feat feature.Feature) error {
	if err := p.RemoveFeature(feat); err != nil {
		return err
	}

	return p.CreateFeature(feat)
}

func (p *Provider) RemoveFeature(feat feature.Feature) error {
	container, err := p.getFeatureByName(feat.Name())
	if err != nil {
		return err
	}

	if container == nil {
		return provider.ErrFeatureNotFound
	}

	err = p.removeContainer(container.id)
	if err != nil {
		return err
	}

	p.cleanUpBinds(container)

	return nil
}

func (p *Provider) ActualState() (*state.Spec, error) {

	// extract applications
	appContainers, err := p.getApplicationContainers()
	if err != nil {
		return nil, err
	}

	applications := make([]resource.Application, 0)
	for _, container := range appContainers {
		app := container.toApplicationResource()
		applications = append(applications, app)
	}

	// extract features
	featContainers, err := p.getFeatureContainers()
	if err != nil {
		return nil, err
	}

	features := state.Feature{}
	for _, container := range featContainers {
		featureName := container.getLabel(featureLabelTag)
		configValue := container.getLabel(configLabelTag)

		switch featureName {
		case feature.NameFluentBit:
			{
				features.FluentBit = &feature.FluentBit{}
				feature.DecodeFeature(configValue, features.FluentBit)
			}
		}

	}

	spec := &state.Spec{
		Applications: applications,
		Feature:      features,
	}

	return spec, nil
}
