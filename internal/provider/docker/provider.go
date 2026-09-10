package docker

import (
	"context"
	"time"

	docker "github.com/docker/docker/client"
	"github.com/mbaitar/gco/agent/internal/config"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/internal/provider"
	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
)

// Provider defines a docker provider which can communicate with the local docker socket.
type Provider struct {
	// client represents the Docker SDK client
	client docker.APIClient
	// addComposeLabel adds the docker compose project label.
	addComposeLabel bool

	// failedUpdates remembers failed rolling updates per application so a
	// broken specification is not retried on every resync.
	failedUpdates map[string]updateFailure

	// networkReady caches whether the managed docker network exists.
	networkReady bool

	// healthPollInterval, settleDuration, updateBackoff and
	// healthDeadlineOverride tune the rolling update behaviour, zero values
	// fall back to the package defaults.
	healthPollInterval     time.Duration
	settleDuration         time.Duration
	updateBackoff          time.Duration
	healthDeadlineOverride time.Duration
}

func NewDockerProvider() *Provider {
	client := newDockerClient()
	return &Provider{
		client:          client,
		addComposeLabel: false,
		failedUpdates:   make(map[string]updateFailure),
	}
}

func (p *Provider) WithConfig(conf config.DockerProvider) *Provider {
	p.addComposeLabel = conf.UseDockerComposeGrouping
	return p
}

func (p *Provider) CreateApplication(ctx context.Context, app *resource.Application) error {
	if effectiveInstances(app) > 1 {
		// the reconcile path also handles the initial creation and cleans up
		// any leftover containers whose names would otherwise conflict
		return p.reconcileInstances(ctx, app)
	}

	container, err := fromApplicationResource(app)
	if err != nil {
		return err
	}

	id, err := p.createContainer(ctx, container)
	if err != nil {
		return err
	}

	err = p.startContainer(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) UpdateApplication(ctx context.Context, app *resource.Application) error {
	if effectiveInstances(app) > 1 {
		return p.reconcileInstances(ctx, app)
	}

	containers, err := p.getContainersByName(ctx, app.Name)
	if err != nil {
		return err
	}

	// pick the running container as the current version, everything else is
	// stale (exited containers, leftovers from an interrupted update)
	var current *internalContainer
	for i := range containers {
		if current == nil && containers[i].state == "running" {
			current = &containers[i]
			continue
		}

		if err := p.removeContainer(ctx, containers[i].id); err != nil {
			log.Warnf("Failed to remove stale container of application '%s': %v", app.Name, err)
		}
	}

	if current == nil {
		// nothing is running (e.g. self-healing after a crash), a plain
		// create is the fastest way back up
		return p.CreateApplication(ctx, app)
	}

	if current.getLabel(hashLabelTag) == app.CalculateHash() {
		// already converged, e.g. after cleaning up a duplicate
		return nil
	}

	return p.rollingUpdate(ctx, current, app)
}

func (p *Provider) RemoveApplication(ctx context.Context, app *resource.Application) error {
	containers, err := p.getContainersByName(ctx, app.Name)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return provider.ErrAppNotFound
	}

	for i := range containers {
		if err := p.removeContainer(ctx, containers[i].id); err != nil {
			return err
		}
	}

	// a removed application should not inherit an old backoff window when it
	// is recreated later
	delete(p.failedUpdates, app.Name)

	return nil
}

func (p *Provider) CreateFeature(ctx context.Context, feat feature.Feature) error {

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

	id, err := p.createContainer(ctx, container)
	if err != nil {
		return err
	}

	err = p.startContainer(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) UpdateFeature(ctx context.Context, feat feature.Feature) error {
	if err := p.RemoveFeature(ctx, feat); err != nil {
		return err
	}

	return p.CreateFeature(ctx, feat)
}

func (p *Provider) RemoveFeature(ctx context.Context, feat feature.Feature) error {
	container, err := p.getFeatureByName(ctx, feat.Name())
	if err != nil {
		return err
	}

	if container == nil {
		return provider.ErrFeatureNotFound
	}

	err = p.removeContainer(ctx, container.id)
	if err != nil {
		return err
	}

	p.cleanUpBinds(container)

	return nil
}

func (p *Provider) ActualState(ctx context.Context) (*state.Spec, error) {

	// extract applications
	appContainers, err := p.getApplicationContainers(ctx)
	if err != nil {
		return nil, err
	}

	// group the containers per application: the instances of an application
	// all carry the same name label
	grouped := make(map[string][]internalContainer)
	order := make([]string, 0, len(appContainers))
	for _, container := range appContainers {
		name := container.getLabel(nameLabelTag)
		if _, seen := grouped[name]; !seen {
			order = append(order, name)
		}
		grouped[name] = append(grouped[name], container)
	}

	applications := make([]resource.Application, 0, len(order))
	for _, name := range order {
		group := grouped[name]

		// remove containers which are not running, they are leftovers of
		// crashed instances or interrupted updates; the reconciler recreates
		// whatever the desired state still needs
		running := make([]internalContainer, 0, len(group))
		for i := range group {
			if group[i].state == "running" || group[i].state == "restarting" {
				running = append(running, group[i])
				continue
			}

			log.Warnf("Cleaning up stale container '%s' of application '%s'", group[i].name, name)
			if err := p.removeContainer(ctx, group[i].id); err != nil {
				log.Warnf("Failed to remove stale container of application '%s': %v", name, err)
			}
		}

		if len(running) == 0 {
			// absent from the actual state, the reconciler recreates it
			continue
		}

		// the observed version is the configuration hash the majority of the
		// running instances carry, ties go to the newest container (docker
		// lists newest first)
		counts := make(map[string]int)
		for i := range running {
			counts[running[i].getLabel(hashLabelTag)]++
		}

		representative := 0
		for i := range running {
			if counts[running[i].getLabel(hashLabelTag)] > counts[running[representative].getLabel(hashLabelTag)] {
				representative = i
			}
		}

		// instances counts every running container, so surplus containers of
		// any version show up as an instance mismatch to the reconciler
		app := running[representative].toApplicationResource()
		app.Instances = len(running)

		// mixed versions can otherwise look converged (majority hash equals
		// the desired one at the desired count), force a reconcile instead
		if len(counts) > 1 {
			app.SetHash("inconsistent")
		}

		applications = append(applications, app)
	}

	// extract features
	featContainers, err := p.getFeatureContainers(ctx)
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
