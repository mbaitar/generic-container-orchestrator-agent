package resource

import (
	applicationv1 "dsync.io/gco/agent/gen/proto/application/v1"
	"dsync.io/gco/agent/internal/hash"
)

// Application defines a structure which describes everything the application needs
// to be translated to a container management system.
type Application struct {
	hash string

	Name  string `json:"name"`
	Image Image  `json:"image"`
	Ports []Port `json:"ports,omitempty"`

	Instances int `json:"instances"`

	LogConfig *LogConfig `json:"logConfig,omitempty"`

	// TODO: custom labels
}

func (a *Application) CalculateHash() string {
	if a.hash == "" {
		// create map for collection properties relevant for hashing
		m := make(map[string]interface{})
		m["name"] = a.Name

		// include image name and tag
		m["image_name"] = a.Image.Name
		m["image_tag"] = a.Image.Tag

		if a.Ports != nil && len(a.Ports) > 0 {
			m["ports"] = a.Ports
		}

		a.hash = hash.CalculateHash(m)
	}

	return a.hash
}

func (a *Application) ToApplicationV1() *applicationv1.Application {
	return &applicationv1.Application{
		Name:      a.Name,
		Image:     a.Image.ToImageV1(),
		Ports:     ToPortsV1(a.Ports),
		Instances: uint32(a.Instances),
	}
}

func FromApplicationV1(v1 *applicationv1.Application) *Application {
	if v1 == nil {
		return nil
	}

	return &Application{
		Name:      v1.Name,
		Image:     *FromImageV1(v1.Image),
		Ports:     FromPortsV1(v1.Ports),
		Instances: int(v1.Instances),
	}
}
