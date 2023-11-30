package resource

import applicationv1 "dsync.io/gco/agent/gen/proto/application/v1"

// Image defines a containerized image.
type Image struct {
	Name       string `json:"name"`
	Tag        string `json:"tag"`
	PullPolicy string `json:"pullPolicy,omitempty"`
}

func (i *Image) ToImageV1() *applicationv1.Image {
	return &applicationv1.Image{
		Name:       i.Name,
		Tag:        i.Tag,
		PullPolicy: i.PullPolicy,
	}
}

func FromImageV1(v1 *applicationv1.Image) *Image {
	if v1 == nil {
		return nil
	}
	return &Image{
		Name:       v1.Name,
		Tag:        v1.Tag,
		PullPolicy: v1.PullPolicy,
	}
}
