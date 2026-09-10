package resource

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/docker/go-units"
	applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"
	"github.com/mbaitar/gco/agent/internal/hash"
)

const (
	RestartPolicyNo            = "no"
	RestartPolicyAlways        = "always"
	RestartPolicyOnFailure     = "on-failure"
	RestartPolicyUnlessStopped = "unless-stopped"
)

// reservedLabelNamespace is the label namespace used by the agent itself and
// can therefore not be used for custom labels.
const reservedLabelNamespace = "gco.io"

// minMemoryBytes is the smallest memory limit accepted by the docker daemon.
const minMemoryBytes = 6 * 1024 * 1024

// volumeNamePattern matches valid docker volume names.
var volumeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// Application defines a structure which describes everything the application needs
// to be translated to a container management system.
type Application struct {
	hash string

	Name  string `json:"name"`
	Image Image  `json:"image"`
	Ports []Port `json:"ports,omitempty"`

	Instances int `json:"instances"`

	// Env lists the environment variables passed to the container.
	Env map[string]string `json:"env,omitempty"`

	// Volumes lists the volumes mounted into the container.
	Volumes []Volume `json:"volumes,omitempty"`

	// Labels lists custom labels added to the container. Keys within the
	// reserved 'gco.io' namespace are rejected.
	Labels map[string]string `json:"labels,omitempty"`

	// NetworkMode configures the container network, e.g. "bridge", "host",
	// "none" or the name of an existing docker network.
	NetworkMode string `json:"networkMode,omitempty"`

	// RestartPolicy configures the provider native restart behaviour:
	// "no", "always", "on-failure" or "unless-stopped".
	RestartPolicy string `json:"restartPolicy,omitempty"`

	// HealthCheck configures the container health check.
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`

	// Resources configures the resource limits for the application.
	Resources *Resources `json:"resources,omitempty"`

	LogConfig *LogConfig `json:"logConfig,omitempty"`
}

// CalculateHash calculates the configuration hash for the application.
// Fields are only included when set, keeping the hash stable for
// applications which do not use them.
func (a *Application) CalculateHash() string {
	if a.hash == "" {
		// create map for collection properties relevant for hashing
		m := make(map[string]interface{})
		m["name"] = a.Name

		// include image name and tag
		m["image_name"] = a.Image.Name
		m["image_tag"] = a.Image.Tag

		if len(a.Ports) > 0 {
			m["ports"] = a.Ports
		}

		if len(a.Env) > 0 {
			m["env"] = a.Env
		}

		if len(a.Volumes) > 0 {
			m["volumes"] = a.Volumes
		}

		if len(a.Labels) > 0 {
			m["labels"] = a.Labels
		}

		if a.NetworkMode != "" {
			m["network_mode"] = a.NetworkMode
		}

		if a.RestartPolicy != "" {
			m["restart_policy"] = a.RestartPolicy
		}

		if a.HealthCheck != nil {
			m["health_check"] = a.HealthCheck
		}

		if a.Resources != nil {
			m["resources"] = a.Resources
		}

		a.hash = hash.CalculateHash(m)
	}

	return a.hash
}

// SetHash overrides the configuration hash of the application. It is used by
// providers which store the configuration hash on the external resource
// (e.g. as a container label) when reconstructing the actual state.
func (a *Application) SetHash(h string) {
	a.hash = h
}

// Validate checks whether the application specification is complete and
// internally consistent before it is accepted into the desired state.
func (a *Application) Validate() error {
	if a.Name == "" {
		return errors.New("application name is required")
	}

	if a.Image.Name == "" {
		return errors.New("image name is required")
	}

	if a.Image.Tag == "" {
		return errors.New("image tag is required")
	}

	if a.Instances < 0 {
		return errors.New("instances cannot be negative")
	}

	if a.Instances > 1 {
		// multiple instances cannot share a fixed host port or the host network
		for _, port := range a.Ports {
			if port.HostPort > 0 {
				return fmt.Errorf("fixed host port %d cannot be combined with multiple instances", port.HostPort)
			}
		}

		if a.NetworkMode == "host" {
			return errors.New("host networking cannot be combined with multiple instances")
		}
	}

	for key := range a.Env {
		if key == "" {
			return errors.New("environment variable names cannot be empty")
		}

		if strings.Contains(key, "=") {
			return fmt.Errorf("environment variable name '%s' cannot contain '='", key)
		}
	}

	destinations := make(map[string]struct{})
	for _, volume := range a.Volumes {
		if volume.Source == "" || volume.Destination == "" {
			return errors.New("volumes require both a source and a destination")
		}

		if strings.Contains(volume.Source, ":") || strings.Contains(volume.Destination, ":") {
			return errors.New("volume paths cannot contain ':'")
		}

		if !strings.HasPrefix(volume.Source, "/") && !volumeNamePattern.MatchString(volume.Source) {
			return fmt.Errorf("volume source '%s' must be an absolute path or a valid volume name", volume.Source)
		}

		if !strings.HasPrefix(volume.Destination, "/") {
			return fmt.Errorf("volume destination '%s' must be an absolute path", volume.Destination)
		}

		if volume.Destination == "/" {
			return errors.New("volume destination cannot be '/'")
		}

		if _, duplicate := destinations[volume.Destination]; duplicate {
			return fmt.Errorf("duplicate volume destination '%s'", volume.Destination)
		}
		destinations[volume.Destination] = struct{}{}
	}

	for key := range a.Labels {
		if key == "" {
			return errors.New("label names cannot be empty")
		}

		if key == reservedLabelNamespace || strings.HasPrefix(key, reservedLabelNamespace+"/") {
			return fmt.Errorf("label '%s' is within the reserved '%s' namespace", key, reservedLabelNamespace)
		}
	}

	switch a.RestartPolicy {
	case "", RestartPolicyNo, RestartPolicyAlways, RestartPolicyOnFailure, RestartPolicyUnlessStopped:
	default:
		return fmt.Errorf("invalid restart policy '%s'", a.RestartPolicy)
	}

	if a.HealthCheck != nil {
		if len(a.HealthCheck.Test) == 0 {
			return errors.New("health checks require a test command")
		}

		// docker silently disables probes with any other type
		switch a.HealthCheck.Test[0] {
		case "CMD", "CMD-SHELL", "NONE":
		default:
			return fmt.Errorf("health check test must start with 'CMD', 'CMD-SHELL' or 'NONE', got '%s'", a.HealthCheck.Test[0])
		}
	}

	if a.Resources != nil && a.Resources.Memory != "" {
		bytes, err := units.RAMInBytes(a.Resources.Memory)
		if err != nil {
			return fmt.Errorf("invalid memory limit '%s': %v", a.Resources.Memory, err)
		}

		// the docker daemon rejects limits below 6MB at container create
		if bytes < minMemoryBytes {
			return fmt.Errorf("memory limit '%s' is below the minimum of 6m", a.Resources.Memory)
		}
	}

	if a.Resources != nil && a.Resources.Cpus < 0 {
		return errors.New("cpu limit cannot be negative")
	}

	return nil
}

func (a *Application) ToApplicationV1() *applicationv1.Application {
	return &applicationv1.Application{
		Name:          a.Name,
		Image:         a.Image.ToImageV1(),
		Ports:         ToPortsV1(a.Ports),
		Instances:     uint32(a.Instances),
		Env:           a.Env,
		Volumes:       ToVolumesV1(a.Volumes),
		Labels:        a.Labels,
		NetworkMode:   a.NetworkMode,
		RestartPolicy: a.RestartPolicy,
		HealthCheck:   a.HealthCheck.ToHealthCheckV1(),
		Resources:     a.Resources.ToResourcesV1(),
	}
}

func FromApplicationV1(v1 *applicationv1.Application) *Application {
	if v1 == nil {
		return nil
	}

	// guard against a missing image, validation will reject it later
	image := FromImageV1(v1.Image)
	if image == nil {
		image = &Image{}
	}

	app := &Application{
		Name:          v1.Name,
		Image:         *image,
		Ports:         FromPortsV1(v1.Ports),
		Instances:     int(v1.Instances),
		Volumes:       FromVolumesV1(v1.Volumes),
		NetworkMode:   v1.NetworkMode,
		RestartPolicy: v1.RestartPolicy,
		HealthCheck:   FromHealthCheckV1(v1.HealthCheck),
		Resources:     FromResourcesV1(v1.Resources),
	}

	// only keep non-empty maps so the hash stays stable between
	// absent and empty values
	if len(v1.Env) > 0 {
		app.Env = v1.Env
	}

	if len(v1.Labels) > 0 {
		app.Labels = v1.Labels
	}

	return app
}
