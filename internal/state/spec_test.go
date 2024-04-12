package state

import (
	"testing"

	"github.com/mabaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

func TestEmptySpec(t *testing.T) {
	spec := EmptySpec()
	if assert.NotNil(t, spec, "spec should not be nil") {
		assert.NotNil(t, spec.Applications, "resources should not be nil")
	}
}

func TestSpec_AddApplication(t *testing.T) {
	spec := EmptySpec()

	app := resource.Application{
		Name: "nginx",
		Image: resource.Image{
			Name: "nginx",
			Tag:  "latest",
		},
	}

	assert.Equal(t, 0, len(spec.Applications))
	err := spec.AddApplication(app)
	assert.Nil(t, err, "should not have thrown")
	if assert.Equal(t, 1, len(spec.Applications)) {
		appSpec := spec.Applications[0]
		assert.NotNil(t, appSpec, "should not be nil")
		assert.Equal(t, "nginx", appSpec.Name)
		assert.Equal(t, "nginx", appSpec.Image.Name)
		assert.Equal(t, "latest", appSpec.Image.Tag)
	}

	err = spec.AddApplication(app)
	assert.NotNil(t, err, "should have thrown as application already exists")
}

func TestSpec_UpdateApplication(t *testing.T) {
	spec := EmptySpec()

	app := resource.Application{
		Name: "nginx",
		Image: resource.Image{
			Name: "nginx",
			Tag:  "latest",
		},
	}

	assert.Equal(t, 0, len(spec.Applications))
	err := spec.AddApplication(app)
	assert.Nil(t, err, "should not have thrown")

	app.Image.Tag = "v1.0.0"
	err = spec.UpdateApplication(app)
	assert.Nil(t, err, "should not have thrown")

	if assert.Equal(t, 1, len(spec.Applications)) {
		res := spec.Applications[0]
		assert.Equal(t, "v1.0.0", res.Image.Tag)
	}

	app.Name = "blah"
	err = spec.UpdateApplication(app)
	assert.NotNil(t, err, "should not have found a matching application")
}

func TestSpec_RemoveApplication(t *testing.T) {
	spec := EmptySpec()

	app := resource.Application{
		Name: "nginx",
		Image: resource.Image{
			Name: "nginx",
			Tag:  "latest",
		},
	}

	assert.Equal(t, 0, len(spec.Applications))
	err := spec.AddApplication(app)
	assert.Nil(t, err, "should not have thrown")
	assert.Equal(t, 1, len(spec.Applications))

	err = spec.RemoveApplication(app.Name)
	assert.Nil(t, err, "should not have thrown")
	assert.Equal(t, 0, len(spec.Applications))

	err = spec.RemoveApplication(app.Name)
	assert.NotNil(t, "should not have found a matching application")
}
