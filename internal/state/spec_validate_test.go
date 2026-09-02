package state

import (
	"testing"

	"github.com/mbaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

func validStateApp(name string) resource.Application {
	return resource.Application{
		Name:  name,
		Image: resource.Image{Name: "nginx", Tag: "latest"},
	}
}

func TestSpec_Validate_acceptsValidSpec(t *testing.T) {
	spec := EmptySpec()
	spec.Applications = append(spec.Applications, validStateApp("a"), validStateApp("b"))

	assert.Nil(t, spec.Validate(), "should have accepted a valid spec")
}

func TestSpec_Validate_rejectsDuplicateNames(t *testing.T) {
	spec := EmptySpec()
	spec.Applications = append(spec.Applications, validStateApp("a"), validStateApp("a"))

	assert.NotNil(t, spec.Validate(), "should have rejected duplicate application names")
}

func TestSpec_Validate_rejectsInvalidApplication(t *testing.T) {
	app := validStateApp("a")
	app.RestartPolicy = "forever"

	spec := EmptySpec()
	spec.Applications = append(spec.Applications, app)

	assert.NotNil(t, spec.Validate(), "should have rejected an invalid application")
}
