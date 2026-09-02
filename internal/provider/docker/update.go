package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/mbaitar/gco/agent/internal/log"
	"github.com/mbaitar/gco/agent/pkg/resource"
)

const (
	// defaultHealthPollInterval is how often the state of a starting container is checked.
	defaultHealthPollInterval = 500 * time.Millisecond

	// defaultSettleDuration is how long a container without a health check gets
	// to prove it does not exit immediately after starting.
	defaultSettleDuration = 3 * time.Second

	// defaultUpdateBackoff is how long a failed update is not retried, so a
	// broken specification does not disrupt the running container on every resync.
	defaultUpdateBackoff = 5 * time.Minute

	// maxHealthDeadline caps how long an update waits for a container to become healthy.
	maxHealthDeadline = 5 * time.Minute

	// docker defaults used when health check fields are not set.
	defaultHealthInterval = 30 * time.Second
	defaultHealthTimeout  = 30 * time.Second
	defaultHealthRetries  = 3
)

// updateFailure remembers a failed update so it is not retried on every resync.
type updateFailure struct {
	hash    string
	retryAt time.Time
}

// skipRecentlyFailed reports whether the given desired hash recently failed to
// deploy for this application and is still within its backoff window.
func (p *Provider) skipRecentlyFailed(name string, hash string) error {
	failure, found := p.failedUpdates[name]
	if !found || failure.hash != hash {
		return nil
	}

	if time.Now().After(failure.retryAt) {
		return nil
	}

	return fmt.Errorf("update of application '%s' failed recently, retrying after %s", name, failure.retryAt.Format(time.TimeOnly))
}

// recordUpdateFailure starts the backoff window for the given desired hash.
func (p *Provider) recordUpdateFailure(name string, hash string) {
	if p.failedUpdates == nil {
		p.failedUpdates = make(map[string]updateFailure)
	}

	backoff := p.updateBackoff
	if backoff <= 0 {
		backoff = defaultUpdateBackoff
	}

	p.failedUpdates[name] = updateFailure{hash: hash, retryAt: time.Now().Add(backoff)}
}

// rollingUpdate replaces the running container of an application with as
// little downtime as possible: the new container is created (pulling the
// image up front) and started next to the old one, and the old container is
// only removed once the new one is healthy. When both containers would fight
// over the same host port or writable volume the old one is stopped, not
// removed, right before starting the new one, so it remains available for a
// rollback. Any failure removes the new container and restores the old one.
func (p *Provider) rollingUpdate(ctx context.Context, old *internalContainer, app *resource.Application) error {
	desiredHash := app.CalculateHash()
	if err := p.skipRecentlyFailed(app.Name, desiredHash); err != nil {
		return err
	}

	next, err := fromApplicationResource(app)
	if err != nil {
		return err
	}

	// create (and pull) before touching the old container
	newID, err := p.createContainer(ctx, next)
	if err != nil {
		p.recordUpdateFailure(app.Name, desiredHash)
		return fmt.Errorf("failed to create replacement container: %w", err)
	}

	// stop the old container first when running both at once is impossible
	stoppedOld := false
	if old.sharesResourcesWith(next) {
		log.Debugf("Application '%s' shares ports or volumes between versions, stopping old container first", app.Name)
		if err = p.stopContainer(ctx, old.id); err != nil {
			// the stop may have partially taken effect, try to restore the
			// old container during the rollback
			p.rollback(ctx, app.Name, newID, old, true)
			p.recordUpdateFailure(app.Name, desiredHash)
			return fmt.Errorf("failed to stop old container: %w", err)
		}
		stoppedOld = true
	}

	if err = p.startContainer(ctx, newID); err == nil {
		err = p.awaitHealthy(ctx, newID, app.HealthCheck)
	}

	if err != nil {
		p.rollback(ctx, app.Name, newID, old, stoppedOld)
		p.recordUpdateFailure(app.Name, desiredHash)
		return fmt.Errorf("update of application '%s' was rolled back: %w", app.Name, err)
	}

	// the new container is healthy, retire the old one
	if err = p.removeContainer(ctx, old.id); err != nil {
		// the next update or resync cleans up leftovers, do not fail the update
		log.Warnf("Failed to remove old container of application '%s': %v", app.Name, err)
	}

	delete(p.failedUpdates, app.Name)
	log.Infof("Application '%s' was updated without downtime (hash=%s)", app.Name, desiredHash)
	return nil
}

// rollback removes the failed replacement container and restarts the old one
// when it was stopped to free shared resources.
func (p *Provider) rollback(ctx context.Context, name string, newID string, old *internalContainer, restartOld bool) {
	log.Warnf("Rolling back failed update of application '%s'", name)

	if err := p.removeContainer(ctx, newID); err != nil {
		log.Errorf("Failed to remove replacement container during rollback of '%s': %v", name, err)
	}

	if restartOld {
		if err := p.startContainer(ctx, old.id); err != nil {
			log.Errorf("Failed to restart old container during rollback of '%s', application is down: %v", name, err)
		}
	}
}

// awaitHealthy blocks until the container proves it is up: containers with a
// health check must report 'healthy', containers without one must still be
// running after a short settle period. An image defined health check is
// honoured as well.
func (p *Provider) awaitHealthy(ctx context.Context, id string, hc *resource.HealthCheck) error {
	poll := p.healthPollInterval
	if poll <= 0 {
		poll = defaultHealthPollInterval
	}

	// a 'NONE' health check disables probing, docker will never report a
	// health status for it
	if hc != nil && len(hc.Test) > 0 && hc.Test[0] == "NONE" {
		hc = nil
	}

	wait := healthDeadline(hc)

	if hc == nil {
		settle := p.settleDuration
		if settle <= 0 {
			settle = defaultSettleDuration
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(settle):
		}

		inspect, err := p.client.ContainerInspect(ctx, id)
		if err != nil {
			return err
		}

		if inspect.State == nil || inspect.State.Status != "running" {
			return fmt.Errorf("container exited during startup")
		}

		// a restart during the settle period means the container crashed and
		// a docker restart policy brought it back, that is not a stable start
		if inspect.RestartCount > 0 {
			return fmt.Errorf("container restarted during startup")
		}

		// no spec health check, but the image may define one
		if inspect.State.Health == nil {
			return nil
		}

		// derive the deadline from the image defined health check timings
		if inspect.Config != nil && inspect.Config.Healthcheck != nil {
			wait = healthDeadlineFromConfig(inspect.Config.Healthcheck)
		}
	}

	if p.healthDeadlineOverride > 0 {
		wait = p.healthDeadlineOverride
	}

	deadline := time.NewTimer(wait)
	defer deadline.Stop()

	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("container did not become healthy in time")
		case <-ticker.C:
		}

		inspect, err := p.client.ContainerInspect(ctx, id)
		if err != nil {
			return err
		}

		if inspect.State == nil {
			continue
		}

		if inspect.State.Status != "running" && inspect.State.Status != "restarting" {
			return fmt.Errorf("container exited before becoming healthy")
		}

		if inspect.State.Health == nil {
			// health status not reported yet
			continue
		}

		switch inspect.State.Health.Status {
		case "healthy":
			return nil
		case "unhealthy":
			return fmt.Errorf("container reported an unhealthy status")
		}
	}
}

// healthDeadlineFromConfig derives the health deadline from a docker health
// check configuration, e.g. one defined by the image.
func healthDeadlineFromConfig(hc *container.HealthConfig) time.Duration {
	converted := &resource.HealthCheck{}

	if hc.Interval > 0 {
		converted.IntervalSeconds = uint32(hc.Interval / time.Second)
	}
	if hc.Timeout > 0 {
		converted.TimeoutSeconds = uint32(hc.Timeout / time.Second)
	}
	if hc.Retries > 0 {
		converted.Retries = uint32(hc.Retries)
	}
	if hc.StartPeriod > 0 {
		converted.StartPeriodSeconds = uint32(hc.StartPeriod / time.Second)
	}

	return healthDeadline(converted)
}

// healthDeadline derives how long to wait for a healthy status from the
// health check configuration, falling back to the docker defaults.
func healthDeadline(hc *resource.HealthCheck) time.Duration {
	interval := defaultHealthInterval
	timeout := defaultHealthTimeout
	retries := defaultHealthRetries
	start := time.Duration(0)

	if hc != nil {
		if hc.IntervalSeconds > 0 {
			interval = time.Duration(hc.IntervalSeconds) * time.Second
		}
		if hc.TimeoutSeconds > 0 {
			timeout = time.Duration(hc.TimeoutSeconds) * time.Second
		}
		if hc.Retries > 0 {
			retries = int(hc.Retries)
		}
		if hc.StartPeriodSeconds > 0 {
			start = time.Duration(hc.StartPeriodSeconds) * time.Second
		}
	}

	deadline := start + (interval+timeout)*time.Duration(retries+1) + 10*time.Second
	if deadline > maxHealthDeadline {
		return maxHealthDeadline
	}

	return deadline
}
