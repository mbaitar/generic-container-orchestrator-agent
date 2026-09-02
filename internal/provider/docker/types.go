package docker

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/mbaitar/gco/agent/pkg/resource"
)

// containerPort describes a port exposed by a container.
type containerPort string

func newContainerPort(private uint16, public uint16, protocol string) containerPort {
	formatted := fmt.Sprintf("%d:%d/%s", private, public, protocol)
	return containerPort(formatted)
}

func newContainerPortFromBinding(port nat.Port, binding nat.PortBinding) containerPort {
	formatted := fmt.Sprintf("%s:%s/%s", port.Port(), binding.HostPort, port.Proto())
	return containerPort(formatted)
}

func (c containerPort) privatePort() string {
	ports := strings.Split(string(c), "/")[0]
	return strings.Split(ports, ":")[0]
}

func (c containerPort) publicPort() string {
	ports := strings.Split(string(c), "/")[0]
	return strings.Split(ports, ":")[1]
}

func (c containerPort) protocol() string {
	return strings.Split(string(c), "/")[1]
}

func (c containerPort) exposedPort() string {
	return fmt.Sprintf("%s/%s", c.privatePort(), c.protocol())
}

// volumeMount represents a volume that needs to be mounted to a container
type volumeMount struct {
	destination string
	source      string
	readonly    bool
}

func (v *volumeMount) asBind() string {
	bind := fmt.Sprintf("%s:%s", v.source, v.destination)
	if v.readonly {
		return bind + ":ro"
	} else {
		return bind
	}
}

// volumeMountFromBind creates a volumeMount structure based on the incoming bind string.
func volumeMountFromBind(bind string) *volumeMount {
	parts := strings.Split(bind, ":")
	if len(parts) < 2 {
		// invalid volume mount
		return nil
	}

	source := parts[0]
	dest := parts[1]

	if len(parts) == 3 && parts[2] == "ro" {
		return &volumeMount{destination: dest, source: source, readonly: true}
	} else {
		return &volumeMount{destination: dest, source: source, readonly: false}
	}
}

// containerName builds the docker container name for an application. The name
// carries a short configuration hash so that during a rolling update the new
// container can exist next to the old one; the application is identified by
// the name label, not by the docker name.
func containerName(app *resource.Application) string {
	h := app.CalculateHash()
	if len(h) < 8 {
		return app.Name
	}

	return fmt.Sprintf("%s-%s", app.Name, h[:8])
}

// internalContainer describes the internal structure on how the docker provider handles container data.
type internalContainer struct {
	id         string
	name       string
	image      string
	labels     map[string]string
	ports      []containerPort
	volumes    []volumeMount
	state      string
	logConfig  container.LogConfig
	pullPolicy imagePullPolicy

	env           []string
	networkMode   string
	restartPolicy string
	healthCheck   *container.HealthConfig
	memoryBytes   int64
	nanoCpus      int64
}

func fromDockerContainer(c container.InspectResponse) internalContainer {
	ic := &internalContainer{
		id:          c.ID,
		name:        c.Name,
		image:       c.Config.Image,
		labels:      c.Config.Labels,
		ports:       make([]containerPort, 0),
		state:       c.State.Status,
		volumes:     make([]volumeMount, 0),
		networkMode: string(c.HostConfig.NetworkMode),
	}

	if ic.image == "" {
		ic.image = c.Config.Image
	}

	for port, bindings := range c.HostConfig.PortBindings {
		for _, binding := range bindings {
			ic.ports = append(ic.ports, newContainerPortFromBinding(port, binding))
		}
	}

	// PortBindings is a map, sort for a deterministic order so the
	// reconstructed application hashes consistently
	sort.Slice(ic.ports, func(a, b int) bool {
		return ic.ports[a] < ic.ports[b]
	})

	for _, binding := range c.HostConfig.Binds {
		mount := volumeMountFromBind(binding)
		if mount != nil {
			ic.volumes = append(ic.volumes, *mount)
		}
	}

	return *ic
}

func fromApplicationResource(app *resource.Application) (*internalContainer, error) {
	ic := &internalContainer{
		name:   containerName(app),
		image:  fmt.Sprintf("%s:%s", app.Image.Name, app.Image.Tag),
		ports:  make([]containerPort, len(app.Ports)),
		labels: make(map[string]string),
	}

	for i, port := range app.Ports {
		ic.ports[i] = newContainerPort(port.ContainerPort, port.HostPort, string(port.Protocol))
	}

	// map environment variables in a deterministic order
	if len(app.Env) > 0 {
		keys := make([]string, 0, len(app.Env))
		for key := range app.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		ic.env = make([]string, 0, len(keys))
		for _, key := range keys {
			ic.env = append(ic.env, fmt.Sprintf("%s=%s", key, app.Env[key]))
		}
	}

	// map volumes
	for _, volume := range app.Volumes {
		ic.volumes = append(ic.volumes, volumeMount{
			source:      volume.Source,
			destination: volume.Destination,
			readonly:    volume.ReadOnly,
		})
	}

	// map network and restart configuration
	ic.networkMode = app.NetworkMode
	ic.restartPolicy = app.RestartPolicy

	// map health check
	if app.HealthCheck != nil {
		ic.healthCheck = &container.HealthConfig{
			Test:        app.HealthCheck.Test,
			Interval:    time.Duration(app.HealthCheck.IntervalSeconds) * time.Second,
			Timeout:     time.Duration(app.HealthCheck.TimeoutSeconds) * time.Second,
			StartPeriod: time.Duration(app.HealthCheck.StartPeriodSeconds) * time.Second,
			Retries:     int(app.HealthCheck.Retries),
		}
	}

	// map resource limits
	if app.Resources != nil {
		if app.Resources.Memory != "" {
			bytes, err := units.RAMInBytes(app.Resources.Memory)
			if err != nil {
				return nil, fmt.Errorf("invalid memory limit '%s': %v", app.Resources.Memory, err)
			}
			ic.memoryBytes = bytes
		}

		ic.nanoCpus = int64(app.Resources.Cpus * 1e9)
	}

	// set custom labels before the reserved labels so they cannot be overridden
	for key, value := range app.Labels {
		ic.addLabel(customLabel(key, value))
	}

	// set default label
	ic.addLabel(kindLabel(resource.ApplicationKind))
	ic.addLabel(nameLabel(app.Name))
	ic.addLabel(hashLabel(app.CalculateHash()))

	// parse log config
	if app.LogConfig != nil {
		if app.LogConfig.Driver == resource.FluentdLogDriver {
			ic.logConfig = container.LogConfig{
				Type: "fluentd",
				Config: map[string]string{
					"labels":          strings.Join([]string{kindLabelTag.string(), managedByLabelTag.string(), nameLabelTag.string()}, ","),
					"fluentd-async":   "true",
					"fluentd-address": app.LogConfig.Config["address"],
				},
			}
		}
	}

	// parse pull policy
	switch strings.ToLower(app.Image.PullPolicy) {
	case "always":
		ic.pullPolicy = alwaysPullPolicy
	default:
		ic.pullPolicy = whenNotPresentPolicy
	}

	return ic, nil
}

func (i *internalContainer) config() *container.Config {
	ports := make(map[nat.Port]struct{})
	for _, port := range i.ports {
		ports[nat.Port(port.exposedPort())] = struct{}{}
	}

	return &container.Config{
		Labels:       i.labels,
		Image:        i.image,
		ExposedPorts: ports,
		Env:          i.env,
		Healthcheck:  i.healthCheck,
	}
}

func (i *internalContainer) hostConfig() *container.HostConfig {
	// map ports
	ports := nat.PortMap{}
	for _, port := range i.ports {
		key, _ := nat.NewPort(port.protocol(), port.privatePort())
		binding := nat.PortBinding{
			HostIP:   "0.0.0.0", // bind to all addresses
			HostPort: port.publicPort(),
		}

		ports[key] = []nat.PortBinding{binding}
	}

	binds := make([]string, len(i.volumes))
	for idx, volume := range i.volumes {
		binds[idx] = volume.asBind()
	}

	hostConfig := &container.HostConfig{
		PortBindings: ports,
		LogConfig:    i.logConfig,
		Binds:        binds,
		NetworkMode:  container.NetworkMode(i.networkMode),
		Resources: container.Resources{
			Memory:   i.memoryBytes,
			NanoCPUs: i.nanoCpus,
		},
	}

	if i.restartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(i.restartPolicy),
		}
	}

	return hostConfig
}

// publishedHostPorts returns the set of host port/protocol pairs the container publishes.
func (i *internalContainer) publishedHostPorts() map[string]struct{} {
	ports := make(map[string]struct{})
	for _, port := range i.ports {
		public := port.publicPort()
		if public == "" || public == "0" {
			continue
		}

		ports[fmt.Sprintf("%s/%s", public, port.protocol())] = struct{}{}
	}

	return ports
}

// sharesResourcesWith reports whether starting the other container while this
// one is still running would conflict: either runs on the host network, both
// publish the same host port, or both mount the same volume source with at
// least one of them writable.
func (i *internalContainer) sharesResourcesWith(other *internalContainer) bool {
	// on the host network the processes themselves bind host ports, two
	// versions of the same application would always collide
	if i.networkMode == "host" || other.networkMode == "host" {
		return true
	}

	ports := i.publishedHostPorts()
	for public := range other.publishedHostPorts() {
		if _, conflict := ports[public]; conflict {
			return true
		}
	}

	sources := make(map[string]bool)
	for _, volume := range i.volumes {
		sources[volume.source] = volume.readonly
	}

	for _, volume := range other.volumes {
		if readonly, shared := sources[volume.source]; shared {
			if !readonly || !volume.readonly {
				return true
			}
		}
	}

	return false
}

func (i *internalContainer) addLabel(label label) {
	i.labels[label.tag.string()] = label.value
}

func (i *internalContainer) getLabel(tag labelTag) string {
	return i.labels[tag.string()]
}

func (i *internalContainer) getImageResource() resource.Image {
	split := strings.Split(i.image, ":")
	return resource.Image{
		Name: split[0],
		Tag:  split[1],
	}
}

func (i *internalContainer) getPortResources() []resource.Port {
	if i.ports == nil || len(i.ports) == 0 {
		return make([]resource.Port, 0)
	}

	ports := make([]resource.Port, 0, len(i.ports))
	for _, port := range i.ports {
		private, _ := strconv.ParseUint(port.privatePort(), 10, 16)
		public, _ := strconv.ParseUint(port.publicPort(), 10, 16)

		if public <= 0 {
			// skip non exposed ports
			continue
		}

		ports = append(ports, resource.Port{
			ContainerPort: uint16(private),
			HostPort:      uint16(public),
			Protocol:      resource.Protocol(port.protocol()),
		})
	}

	return ports
}

func (i *internalContainer) toApplicationResource() resource.Application {
	instances := 0

	// a container in the 'restarting' state is being managed by a docker
	// native restart policy, counting it as down would make the agent and
	// docker fight over the same container in an endless remove/create loop
	if i.state == "running" || i.state == "restarting" {
		instances = 1
	}

	app := resource.Application{
		Name:      i.getLabel(nameLabelTag),
		Image:     i.getImageResource(),
		Ports:     i.getPortResources(),
		Instances: instances,
	}

	// the configuration hash stored on the container is authoritative for
	// change detection, reconstructing every field from the container
	// (e.g. env vars merged with image defaults) would never be exact
	if h := i.getLabel(hashLabelTag); h != "" {
		app.SetHash(h)
	}

	return app
}
