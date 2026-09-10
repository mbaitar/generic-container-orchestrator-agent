package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/* Validate: instances */

func Test_Validate_rejectsNegativeInstances(t *testing.T) {
	app := validApp()
	app.Instances = -1
	assert.ErrorContains(t, app.Validate(), "instances cannot be negative")
}

func Test_Validate_acceptsZeroInstances(t *testing.T) {
	// specifications without the field default to a single instance
	app := validApp()
	app.Instances = 0
	assert.NoError(t, app.Validate())
}

func Test_Validate_multipleInstancesRejectFixedHostPort(t *testing.T) {
	// validApp publishes host port 8080, which two instances cannot share
	app := validApp()
	app.Instances = 2
	assert.ErrorContains(t, app.Validate(), "fixed host port 8080")
}

func Test_Validate_multipleInstancesAcceptUnpublishedPorts(t *testing.T) {
	app := validApp()
	app.Instances = 2
	app.Ports = []Port{
		{ContainerPort: 80, HostPort: 0, Protocol: TcpProtocol},
	}
	assert.NoError(t, app.Validate(), "an unpublished port should not conflict between instances")
}

func Test_Validate_multipleInstancesRejectHostNetwork(t *testing.T) {
	app := validApp()
	app.Instances = 2
	app.Ports = nil
	app.NetworkMode = "host"
	assert.ErrorContains(t, app.Validate(), "host networking cannot be combined with multiple instances")
}

func Test_Validate_singleInstanceAcceptsFixedHostPort(t *testing.T) {
	app := validApp()
	app.Instances = 1
	assert.NoError(t, app.Validate())
}
