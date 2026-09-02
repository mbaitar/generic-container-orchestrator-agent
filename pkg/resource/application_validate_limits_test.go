package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validLimitsApp() Application {
	return Application{
		Name:  "app",
		Image: Image{Name: "nginx", Tag: "latest"},
	}
}

func TestValidate_rejectsMemoryBelowDockerMinimum(t *testing.T) {
	app := validLimitsApp()

	for _, memory := range []string{"4m", "512k", "512"} {
		app.Resources = &Resources{Memory: memory}
		err := app.Validate()
		assert.NotNil(t, err, "should have rejected memory limit '%s' below 6m", memory)
	}

	app.Resources = &Resources{Memory: "6m"}
	assert.Nil(t, app.Validate(), "should have accepted the 6m minimum")
}

func TestValidate_rejectsUnknownHealthCheckType(t *testing.T) {
	app := validLimitsApp()

	app.HealthCheck = &HealthCheck{Test: []string{"curl", "-f", "http://localhost/"}}
	assert.NotNil(t, app.Validate(), "should have rejected a health check without a CMD prefix")

	for _, kind := range []string{"CMD", "CMD-SHELL", "NONE"} {
		app.HealthCheck = &HealthCheck{Test: []string{kind, "true"}}
		assert.Nil(t, app.Validate(), "should have accepted health check type '%s'", kind)
	}
}

func TestValidate_rejectsRelativeVolumeSource(t *testing.T) {
	app := validLimitsApp()

	app.Volumes = []Volume{{Source: "conf/app", Destination: "/data"}}
	assert.NotNil(t, app.Validate(), "should have rejected a relative volume source")

	app.Volumes = []Volume{{Source: "/etc/conf", Destination: "/data"}}
	assert.Nil(t, app.Validate(), "should have accepted an absolute volume source")

	app.Volumes = []Volume{{Source: "my-volume", Destination: "/data"}}
	assert.Nil(t, app.Validate(), "should have accepted a named volume source")
}

func TestValidate_rejectsRootVolumeDestination(t *testing.T) {
	app := validLimitsApp()

	app.Volumes = []Volume{{Source: "my-volume", Destination: "/"}}
	assert.NotNil(t, app.Validate(), "should have rejected '/' as volume destination")
}

func TestValidate_rejectsDuplicateVolumeDestinations(t *testing.T) {
	app := validLimitsApp()

	app.Volumes = []Volume{
		{Source: "vol-a", Destination: "/data"},
		{Source: "vol-b", Destination: "/data"},
	}
	assert.NotNil(t, app.Validate(), "should have rejected duplicate volume destinations")
}
