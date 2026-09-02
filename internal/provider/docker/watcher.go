package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/mbaitar/gco/agent/internal/log"
)

// reconnectDelay is the time waited before re-subscribing to the docker
// events API after the event stream returned an error.
const reconnectDelay = 5 * time.Second

// lifecycleActions lists the container events which can change the actual state
// of the system and therefore require a reconciliation.
var lifecycleActions = []events.Action{
	events.ActionCreate,
	events.ActionStart,
	events.ActionStop,
	events.ActionDie,
	events.ActionKill,
	events.ActionDestroy,
	events.ActionPause,
	events.ActionUnPause,
	events.ActionRename,
	events.ActionOOM,
}

// Watch subscribes to the docker events API and emits a signal whenever a
// container managed by GCO changes state (started, died, removed, ...).
// The returned channel is closed when the context is cancelled.
func (p *Provider) Watch(ctx context.Context) (<-chan struct{}, error) {
	out := make(chan struct{}, 1)

	go func() {
		defer close(out)
		p.streamEvents(ctx, out)
	}()

	return out, nil
}

// streamEvents keeps a subscription to the docker events API open until the
// context is cancelled, re-subscribing with a delay whenever the stream fails.
func (p *Provider) streamEvents(ctx context.Context, out chan<- struct{}) {
	for {
		opts := events.ListOptions{Filters: filters.NewArgs()}
		opts.Filters.Add("type", string(events.ContainerEventType))
		opts.Filters.Add("label", managedByLabel().string())
		for _, action := range lifecycleActions {
			opts.Filters.Add("event", string(action))
		}

		msgs, errs := p.client.Events(ctx, opts)

	stream:
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgs:
				log.Debugf("Received docker event (action=%s, container=%s)", msg.Action, msg.Actor.ID)

				// a single pending signal is enough, drop the rest
				select {
				case out <- struct{}{}:
				default:
				}
			case err := <-errs:
				if ctx.Err() != nil {
					return
				}

				log.Warnf("Docker event stream failed: %v (reconnecting in %s)", err, reconnectDelay)
				break stream
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}
