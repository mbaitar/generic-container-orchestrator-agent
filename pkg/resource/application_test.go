package resource

import (
	"testing"

	"github.com/mbaitar/gco/agent/internal/hash"
	"github.com/stretchr/testify/assert"
)

// validApp returns a minimal valid application without any of the
// extended fields (env, volumes, labels, network mode, restart policy,
// health check, resources).
func validApp() *Application {
	return &Application{
		Name: "app",
		Image: Image{
			Name: "nginx",
			Tag:  "latest",
		},
		Ports: []Port{
			{ContainerPort: 80, HostPort: 8080, Protocol: TcpProtocol},
		},
		Instances: 1,
	}
}

// fullApp returns a valid application with every extended field set.
func fullApp() *Application {
	app := validApp()
	app.Env = map[string]string{"FOO": "bar", "BAZ": "qux"}
	app.Volumes = []Volume{
		{Source: "data", Destination: "/var/lib/data", ReadOnly: true},
	}
	app.Labels = map[string]string{"team": "platform"}
	app.NetworkMode = "bridge"
	app.RestartPolicy = RestartPolicyAlways
	app.HealthCheck = &HealthCheck{
		Test:               []string{"CMD", "curl", "-f", "http://localhost/"},
		IntervalSeconds:    30,
		TimeoutSeconds:     5,
		Retries:            3,
		StartPeriodSeconds: 10,
	}
	app.Resources = &Resources{Memory: "512m", Cpus: 0.5}
	return app
}

/* CalculateHash */

func Test_CalculateHash_deterministicAcrossCalls(t *testing.T) {
	app := fullApp()

	first := app.CalculateHash()
	second := app.CalculateHash()

	assert.NotEmpty(t, first, "hash should not be empty")
	assert.Equal(t, first, second, "repeated calls should return the same hash")
}

func Test_CalculateHash_deterministicAcrossInstances(t *testing.T) {
	first := fullApp().CalculateHash()
	second := fullApp().CalculateHash()

	assert.Equal(t, first, second, "identical applications should hash identically")
}

func Test_CalculateHash_deterministicAcrossMapInsertionOrders(t *testing.T) {
	first := validApp()
	first.Env = map[string]string{}
	first.Env["A"] = "1"
	first.Env["B"] = "2"
	first.Env["C"] = "3"
	first.Labels = map[string]string{}
	first.Labels["x"] = "1"
	first.Labels["y"] = "2"

	second := validApp()
	second.Env = map[string]string{}
	second.Env["C"] = "3"
	second.Env["A"] = "1"
	second.Env["B"] = "2"
	second.Labels = map[string]string{}
	second.Labels["y"] = "2"
	second.Labels["x"] = "1"

	assert.Equal(t, first.CalculateHash(), second.CalculateHash(),
		"map insertion order should not influence the hash")
}

func Test_CalculateHash_envChangesHash(t *testing.T) {
	app := validApp()
	app.Env = map[string]string{"FOO": "bar"}

	assert.NotEqual(t, validApp().CalculateHash(), app.CalculateHash(),
		"setting env should change the hash")
}

func Test_CalculateHash_volumesChangeHash(t *testing.T) {
	app := validApp()
	app.Volumes = []Volume{{Source: "data", Destination: "/data"}}

	assert.NotEqual(t, validApp().CalculateHash(), app.CalculateHash(),
		"setting volumes should change the hash")
}

func Test_CalculateHash_labelsChangeHash(t *testing.T) {
	app := validApp()
	app.Labels = map[string]string{"team": "platform"}

	assert.NotEqual(t, validApp().CalculateHash(), app.CalculateHash(),
		"setting labels should change the hash")
}

func Test_CalculateHash_networkModeChangesHash(t *testing.T) {
	app := validApp()
	app.NetworkMode = "host"

	assert.NotEqual(t, validApp().CalculateHash(), app.CalculateHash(),
		"setting the network mode should change the hash")
}

func Test_CalculateHash_restartPolicyChangesHash(t *testing.T) {
	app := validApp()
	app.RestartPolicy = RestartPolicyAlways

	assert.NotEqual(t, validApp().CalculateHash(), app.CalculateHash(),
		"setting the restart policy should change the hash")
}

func Test_CalculateHash_healthCheckChangesHash(t *testing.T) {
	app := validApp()
	app.HealthCheck = &HealthCheck{Test: []string{"CMD", "true"}}

	assert.NotEqual(t, validApp().CalculateHash(), app.CalculateHash(),
		"setting a health check should change the hash")
}

func Test_CalculateHash_resourcesChangeHash(t *testing.T) {
	app := validApp()
	app.Resources = &Resources{Memory: "512m"}

	assert.NotEqual(t, validApp().CalculateHash(), app.CalculateHash(),
		"setting resources should change the hash")
}

func Test_CalculateHash_backwardCompatibleWithPorts(t *testing.T) {
	app := validApp()

	// pre-change formula: name, image_name, image_tag and ports (only when
	// non-empty), hashed via the internal hash package
	expected := hash.CalculateHash(map[string]interface{}{
		"name":       app.Name,
		"image_name": app.Image.Name,
		"image_tag":  app.Image.Tag,
		"ports":      app.Ports,
	})

	assert.Equal(t, expected, app.CalculateHash(),
		"application without new fields should keep the pre-change hash")
}

func Test_CalculateHash_backwardCompatibleWithoutPorts(t *testing.T) {
	app := validApp()
	app.Ports = nil

	expected := hash.CalculateHash(map[string]interface{}{
		"name":       app.Name,
		"image_name": app.Image.Name,
		"image_tag":  app.Image.Tag,
	})

	assert.Equal(t, expected, app.CalculateHash(),
		"application without ports should keep the pre-change hash")
}

func Test_CalculateHash_nilAndEmptyEnvHashTheSame(t *testing.T) {
	withNil := validApp()
	withNil.Env = nil

	withEmpty := validApp()
	withEmpty.Env = map[string]string{}

	assert.Equal(t, withNil.CalculateHash(), withEmpty.CalculateHash(),
		"nil and empty env maps should hash the same")
}

func Test_CalculateHash_nilAndEmptyLabelsHashTheSame(t *testing.T) {
	withNil := validApp()
	withNil.Labels = nil

	withEmpty := validApp()
	withEmpty.Labels = map[string]string{}

	assert.Equal(t, withNil.CalculateHash(), withEmpty.CalculateHash(),
		"nil and empty label maps should hash the same")
}

/* SetHash */

func Test_SetHash_overridesCalculatedHash(t *testing.T) {
	app := validApp()
	app.SetHash("stored-hash")

	assert.Equal(t, "stored-hash", app.CalculateHash(),
		"CalculateHash should return the overridden hash")
}

func Test_SetHash_overridesCachedHash(t *testing.T) {
	app := validApp()
	calculated := app.CalculateHash()

	app.SetHash("stored-hash")

	assert.NotEqual(t, calculated, app.CalculateHash())
	assert.Equal(t, "stored-hash", app.CalculateHash(),
		"SetHash should replace an already cached hash")
}

/* Validate */

func Test_Validate_acceptsValidApplication(t *testing.T) {
	assert.NoError(t, validApp().Validate())
}

func Test_Validate_acceptsFullApplication(t *testing.T) {
	assert.NoError(t, fullApp().Validate())
}

func Test_Validate_rejectsMissingName(t *testing.T) {
	app := validApp()
	app.Name = ""

	assert.ErrorContains(t, app.Validate(), "application name is required")
}

func Test_Validate_rejectsMissingImageName(t *testing.T) {
	app := validApp()
	app.Image.Name = ""

	assert.ErrorContains(t, app.Validate(), "image name is required")
}

func Test_Validate_rejectsMissingImageTag(t *testing.T) {
	app := validApp()
	app.Image.Tag = ""

	assert.ErrorContains(t, app.Validate(), "image tag is required")
}

func Test_Validate_rejectsEmptyEnvKey(t *testing.T) {
	app := validApp()
	app.Env = map[string]string{"": "value"}

	assert.ErrorContains(t, app.Validate(), "environment variable names cannot be empty")
}

func Test_Validate_rejectsEnvKeyWithEquals(t *testing.T) {
	app := validApp()
	app.Env = map[string]string{"FOO=BAR": "value"}

	assert.ErrorContains(t, app.Validate(), "cannot contain '='")
}

func Test_Validate_acceptsEnvValueWithEquals(t *testing.T) {
	app := validApp()
	app.Env = map[string]string{"FOO": "bar=baz"}

	assert.NoError(t, app.Validate(), "'=' is only forbidden in the key")
}

func Test_Validate_rejectsVolumeWithoutSource(t *testing.T) {
	app := validApp()
	app.Volumes = []Volume{{Destination: "/data"}}

	assert.ErrorContains(t, app.Validate(), "volumes require both a source and a destination")
}

func Test_Validate_rejectsVolumeWithoutDestination(t *testing.T) {
	app := validApp()
	app.Volumes = []Volume{{Source: "data"}}

	assert.ErrorContains(t, app.Validate(), "volumes require both a source and a destination")
}

func Test_Validate_rejectsVolumeSourceWithColon(t *testing.T) {
	app := validApp()
	app.Volumes = []Volume{{Source: "da:ta", Destination: "/data"}}

	assert.ErrorContains(t, app.Validate(), "volume paths cannot contain ':'")
}

func Test_Validate_rejectsVolumeDestinationWithColon(t *testing.T) {
	app := validApp()
	app.Volumes = []Volume{{Source: "data", Destination: "/da:ta"}}

	assert.ErrorContains(t, app.Validate(), "volume paths cannot contain ':'")
}

func Test_Validate_rejectsRelativeVolumeDestination(t *testing.T) {
	app := validApp()
	app.Volumes = []Volume{{Source: "data", Destination: "data"}}

	assert.ErrorContains(t, app.Validate(), "must be an absolute path")
}

func Test_Validate_acceptsNamedVolume(t *testing.T) {
	app := validApp()
	app.Volumes = []Volume{{Source: "data", Destination: "/var/lib/data", ReadOnly: true}}

	assert.NoError(t, app.Validate())
}

func Test_Validate_rejectsEmptyLabelKey(t *testing.T) {
	app := validApp()
	app.Labels = map[string]string{"": "value"}

	assert.ErrorContains(t, app.Validate(), "label names cannot be empty")
}

func Test_Validate_rejectsReservedLabelExact(t *testing.T) {
	app := validApp()
	app.Labels = map[string]string{"gco.io": "value"}

	assert.ErrorContains(t, app.Validate(), "reserved 'gco.io' namespace")
}

func Test_Validate_rejectsReservedLabelPrefix(t *testing.T) {
	app := validApp()
	app.Labels = map[string]string{"gco.io/hash": "value"}

	assert.ErrorContains(t, app.Validate(), "reserved 'gco.io' namespace")
}

func Test_Validate_acceptsSimilarButUnreservedLabel(t *testing.T) {
	app := validApp()
	app.Labels = map[string]string{"gco.io-custom": "value", "gco.iox": "value"}

	assert.NoError(t, app.Validate(),
		"only the exact 'gco.io' key and the 'gco.io/' prefix are reserved")
}

func Test_Validate_acceptsKnownRestartPolicies(t *testing.T) {
	policies := []string{"", RestartPolicyNo, RestartPolicyAlways, RestartPolicyOnFailure, RestartPolicyUnlessStopped}
	for _, policy := range policies {
		app := validApp()
		app.RestartPolicy = policy

		assert.NoError(t, app.Validate(), "restart policy '%s' should be valid", policy)
	}
}

func Test_Validate_rejectsUnknownRestartPolicy(t *testing.T) {
	app := validApp()
	app.RestartPolicy = "sometimes"

	assert.ErrorContains(t, app.Validate(), "invalid restart policy 'sometimes'")
}

func Test_Validate_rejectsHealthCheckWithoutTest(t *testing.T) {
	app := validApp()
	app.HealthCheck = &HealthCheck{IntervalSeconds: 30}

	assert.ErrorContains(t, app.Validate(), "health checks require a test command")
}

func Test_Validate_acceptsHealthCheckWithTest(t *testing.T) {
	app := validApp()
	app.HealthCheck = &HealthCheck{Test: []string{"CMD", "true"}}

	assert.NoError(t, app.Validate())
}

func Test_Validate_acceptsValidMemoryLimits(t *testing.T) {
	for _, memory := range []string{"512m", "1g"} {
		app := validApp()
		app.Resources = &Resources{Memory: memory}

		assert.NoError(t, app.Validate(), "memory limit '%s' should be valid", memory)
	}
}

func Test_Validate_rejectsInvalidMemoryLimit(t *testing.T) {
	app := validApp()
	app.Resources = &Resources{Memory: "512x"}

	assert.ErrorContains(t, app.Validate(), "invalid memory limit '512x'")
}

func Test_Validate_rejectsNegativeCpus(t *testing.T) {
	app := validApp()
	app.Resources = &Resources{Cpus: -0.5}

	assert.ErrorContains(t, app.Validate(), "cpu limit cannot be negative")
}

func Test_Validate_acceptsZeroAndPositiveCpus(t *testing.T) {
	for _, cpus := range []float64{0, 0.5, 2} {
		app := validApp()
		app.Resources = &Resources{Cpus: cpus}

		assert.NoError(t, app.Validate(), "cpu limit '%f' should be valid", cpus)
	}
}
