package resource

import applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"

// Volume describes a volume that is mounted into the container. The source can
// either be an absolute path on the host or the name of a docker volume.
type Volume struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"readOnly,omitempty"`
}

func (v *Volume) ToVolumeV1() *applicationv1.Volume {
	return &applicationv1.Volume{
		Source:      v.Source,
		Destination: v.Destination,
		ReadOnly:    v.ReadOnly,
	}
}

func ToVolumesV1(volumes []Volume) []*applicationv1.Volume {
	v1s := make([]*applicationv1.Volume, 0, len(volumes))
	for _, v := range volumes {
		v1s = append(v1s, v.ToVolumeV1())
	}
	return v1s
}

func FromVolumeV1(v1 *applicationv1.Volume) *Volume {
	if v1 == nil {
		return nil
	}

	return &Volume{
		Source:      v1.Source,
		Destination: v1.Destination,
		ReadOnly:    v1.ReadOnly,
	}
}

func FromVolumesV1(v1 []*applicationv1.Volume) []Volume {
	if len(v1) == 0 {
		return nil
	}

	volumes := make([]Volume, 0, len(v1))
	for _, v := range v1 {
		if converted := FromVolumeV1(v); converted != nil {
			volumes = append(volumes, *converted)
		}
	}
	return volumes
}
