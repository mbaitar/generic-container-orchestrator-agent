package docker

import (
	"dsync.io/gco/agent/pkg/resource"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"strconv"
	"strings"
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
}

func fromDockerContainer(c types.ContainerJSON) internalContainer {
	ic := &internalContainer{
		id:      c.ID,
		name:    c.Name,
		image:   c.Config.Image,
		labels:  c.Config.Labels,
		ports:   make([]containerPort, 0),
		state:   c.State.Status,
		volumes: make([]volumeMount, 0),
	}

	if ic.image == "" {
		ic.image = c.Config.Image
	}

	for port, bindings := range c.HostConfig.PortBindings {
		for _, binding := range bindings {
			ic.ports = append(ic.ports, newContainerPortFromBinding(port, binding))
		}
	}

	for _, binding := range c.HostConfig.Binds {
		mount := volumeMountFromBind(binding)
		if mount != nil {
			ic.volumes = append(ic.volumes, *mount)
		}
	}

	return *ic
}

func fromApplicationResource(app *resource.Application) *internalContainer {
	ic := &internalContainer{
		name:   app.Name,
		image:  fmt.Sprintf("%s:%s", app.Image.Name, app.Image.Tag),
		ports:  make([]containerPort, len(app.Ports)),
		labels: make(map[string]string),
	}

	for i, port := range app.Ports {
		ic.ports[i] = newContainerPort(port.ContainerPort, port.HostPort, string(port.Protocol))
	}

	// set default label
	ic.addLabel(kindLabel(resource.ApplicationKind))
	ic.addLabel(nameLabel(app.Name))

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

	return ic
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

	return &container.HostConfig{
		PortBindings: ports,
		LogConfig:    i.logConfig,
		Binds:        binds,
	}
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
	if i.state == "running" {
		instances = 1
	}

	return resource.Application{
		Name:      i.getLabel(nameLabelTag),
		Image:     i.getImageResource(),
		Ports:     i.getPortResources(),
		Instances: instances,
	}
}
