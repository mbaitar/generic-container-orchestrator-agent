package resource

import applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"

// Resources describes the resource limits for an application instance.
type Resources struct {
	// Memory is a human readable memory limit, e.g. "512m" or "1g".
	Memory string `json:"memory,omitempty"`
	// Cpus limits the amount of CPU available to the container, e.g. 0.5.
	Cpus float64 `json:"cpus,omitempty"`
}

func (r *Resources) ToResourcesV1() *applicationv1.Resources {
	if r == nil {
		return nil
	}

	return &applicationv1.Resources{
		Memory: r.Memory,
		Cpus:   r.Cpus,
	}
}

func FromResourcesV1(v1 *applicationv1.Resources) *Resources {
	if v1 == nil {
		return nil
	}

	return &Resources{
		Memory: v1.Memory,
		Cpus:   v1.Cpus,
	}
}
