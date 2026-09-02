package resource

import (
	"testing"

	applicationv1 "github.com/mbaitar/gco/agent/gen/proto/application/v1"
	"github.com/stretchr/testify/assert"
)

/* Application conversions */

func Test_ApplicationRoundtrip_preservesEveryField(t *testing.T) {
	original := &Application{
		Name: "app",
		Image: Image{
			Name:       "nginx",
			Tag:        "1.27",
			PullPolicy: "always",
		},
		Ports: []Port{
			{ContainerPort: 80, HostPort: 8080, Protocol: TcpProtocol},
			{ContainerPort: 53, HostPort: 5353, Protocol: UdpProtocol},
		},
		Instances: 3,
		Env:       map[string]string{"FOO": "bar", "BAZ": "qux"},
		Volumes: []Volume{
			{Source: "data", Destination: "/var/lib/data", ReadOnly: true},
			{Source: "/host/config", Destination: "/etc/config"},
		},
		Labels:        map[string]string{"team": "platform"},
		NetworkMode:   "bridge",
		RestartPolicy: RestartPolicyUnlessStopped,
		HealthCheck: &HealthCheck{
			Test:               []string{"CMD", "curl", "-f", "http://localhost/"},
			IntervalSeconds:    30,
			TimeoutSeconds:     5,
			Retries:            3,
			StartPeriodSeconds: 10,
		},
		Resources: &Resources{Memory: "512m", Cpus: 1.5},
	}

	converted := FromApplicationV1(original.ToApplicationV1())

	assert.Equal(t, original, converted, "roundtrip should preserve every field")
}

func Test_ApplicationRoundtrip_minimalApplication(t *testing.T) {
	original := &Application{
		Name: "app",
		Image: Image{
			Name: "nginx",
			Tag:  "latest",
		},
		Ports:     []Port{},
		Instances: 1,
	}

	converted := FromApplicationV1(original.ToApplicationV1())

	assert.Equal(t, original, converted,
		"roundtrip of a minimal application should not introduce values")
}

func Test_FromApplicationV1_nilApplication(t *testing.T) {
	assert.Nil(t, FromApplicationV1(nil))
}

func Test_FromApplicationV1_nilImage(t *testing.T) {
	v1 := &applicationv1.Application{Name: "app"}

	app := FromApplicationV1(v1)

	if assert.NotNil(t, app) {
		assert.Equal(t, Image{}, app.Image, "a missing image should yield an empty image")
	}
}

func Test_FromApplicationV1_dropsEmptyEnvAndLabels(t *testing.T) {
	v1 := &applicationv1.Application{
		Name:   "app",
		Image:  &applicationv1.Image{Name: "nginx", Tag: "latest"},
		Env:    map[string]string{},
		Labels: map[string]string{},
	}

	app := FromApplicationV1(v1)

	if assert.NotNil(t, app) {
		assert.Nil(t, app.Env, "empty env map should be dropped")
		assert.Nil(t, app.Labels, "empty label map should be dropped")
	}
}

func Test_FromApplicationV1_keepsNonEmptyEnvAndLabels(t *testing.T) {
	v1 := &applicationv1.Application{
		Name:   "app",
		Image:  &applicationv1.Image{Name: "nginx", Tag: "latest"},
		Env:    map[string]string{"FOO": "bar"},
		Labels: map[string]string{"team": "platform"},
	}

	app := FromApplicationV1(v1)

	if assert.NotNil(t, app) {
		assert.Equal(t, map[string]string{"FOO": "bar"}, app.Env)
		assert.Equal(t, map[string]string{"team": "platform"}, app.Labels)
	}
}

/* Port conversions */

func Test_FromProtocolV1_defaultsUnspecifiedToTcp(t *testing.T) {
	assert.Equal(t, TcpProtocol, FromProtocolV1(applicationv1.Protocol_PROTOCOL_UNSPECIFIED))
}

func Test_FromProtocolV1_mapsKnownProtocols(t *testing.T) {
	assert.Equal(t, TcpProtocol, FromProtocolV1(applicationv1.Protocol_PROTOCOL_TCP))
	assert.Equal(t, UdpProtocol, FromProtocolV1(applicationv1.Protocol_PROTOCOL_UDP))
}

func Test_FromPortV1_defaultsUnspecifiedProtocolToTcp(t *testing.T) {
	v1 := &applicationv1.Port{ContainerPort: 80, HostPort: 8080}

	port := FromPortV1(v1)

	if assert.NotNil(t, port) {
		assert.Equal(t, TcpProtocol, port.Protocol)
	}
}

func Test_PortRoundtrip_preservesEveryField(t *testing.T) {
	original := Port{ContainerPort: 53, HostPort: 5353, Protocol: UdpProtocol}

	converted := FromPortV1(original.ToPortV1())

	if assert.NotNil(t, converted) {
		assert.Equal(t, original, *converted)
	}
}

/* Volume conversions */

func Test_VolumeRoundtrip_preservesEveryField(t *testing.T) {
	original := Volume{Source: "data", Destination: "/var/lib/data", ReadOnly: true}

	converted := FromVolumeV1(original.ToVolumeV1())

	if assert.NotNil(t, converted) {
		assert.Equal(t, original, *converted)
	}
}

func Test_FromVolumeV1_nilVolume(t *testing.T) {
	assert.Nil(t, FromVolumeV1(nil))
}

func Test_FromVolumesV1_emptySliceYieldsNil(t *testing.T) {
	assert.Nil(t, FromVolumesV1(nil))
	assert.Nil(t, FromVolumesV1([]*applicationv1.Volume{}))
}

/* HealthCheck conversions */

func Test_HealthCheckRoundtrip_preservesEveryField(t *testing.T) {
	original := &HealthCheck{
		Test:               []string{"CMD-SHELL", "curl -f http://localhost/"},
		IntervalSeconds:    15,
		TimeoutSeconds:     3,
		Retries:            5,
		StartPeriodSeconds: 60,
	}

	converted := FromHealthCheckV1(original.ToHealthCheckV1())

	assert.Equal(t, original, converted)
}

func Test_HealthCheckConversions_nilSafe(t *testing.T) {
	var healthCheck *HealthCheck

	assert.Nil(t, healthCheck.ToHealthCheckV1(), "nil receiver should yield nil")
	assert.Nil(t, FromHealthCheckV1(nil))
}

/* Resources conversions */

func Test_ResourcesRoundtrip_preservesEveryField(t *testing.T) {
	original := &Resources{Memory: "1g", Cpus: 2.5}

	converted := FromResourcesV1(original.ToResourcesV1())

	assert.Equal(t, original, converted)
}

func Test_ResourcesConversions_nilSafe(t *testing.T) {
	var resources *Resources

	assert.Nil(t, resources.ToResourcesV1(), "nil receiver should yield nil")
	assert.Nil(t, FromResourcesV1(nil))
}

/* Image conversions */

func Test_ImageRoundtrip_preservesEveryField(t *testing.T) {
	original := &Image{Name: "nginx", Tag: "1.27", PullPolicy: "always"}

	converted := FromImageV1(original.ToImageV1())

	assert.Equal(t, original, converted)
}

func Test_FromImageV1_nilImage(t *testing.T) {
	assert.Nil(t, FromImageV1(nil))
}
