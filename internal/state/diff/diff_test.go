package diff

import (
	"testing"

	"github.com/mbaitar/gco/agent/internal/state"
	"github.com/mbaitar/gco/agent/pkg/feature"
	"github.com/mbaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

func SampleApp(name string) *resource.Application {
	return &resource.Application{
		Name: name,
		Ports: []resource.Port{
			{ContainerPort: 80, HostPort: 8080, Protocol: resource.TcpProtocol},
		},
		Image: resource.Image{
			Name: "nginx",
			Tag:  "latest",
		},
		Instances: 1,
	}
}

func Test_changes_onlyAddedResources(t *testing.T) {
	actual := state.EmptySpec()
	desired := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
			*SampleApp("app-2"),
			*SampleApp("app-3"),
		},
	}

	c := compare(desired, actual)
	if assert.NotNil(t, c, "should not return nil") {
		assert.Equal(t, 3, len(c.apps.added), "should have found three new resources")

		app1Found, app2Found, app3Found := false, false, false
		for _, app := range c.apps.added {
			if app.Name == "app-1" {
				app1Found = true
			} else if app.Name == "app-2" {
				app2Found = true
			} else if app.Name == "app-3" {
				app3Found = true
			}

		}

		assert.True(t, app1Found, "should have found 'app-1'")
		assert.True(t, app2Found, "should have found 'app-2'")
		assert.True(t, app3Found, "should have found 'app-3'")
	}
}

func Test_changes_unchangedApplication(t *testing.T) {
	actual := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}
	desired := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}

	c := compare(desired, actual)
	if assert.NotNil(t, c, "should not return nil") {
		assert.Equal(t, 1, len(c.apps.unchanged), "should have found one unchanged resource")

		app1Found := false
		for _, app := range c.apps.unchanged {
			if app.Name == "app-1" {
				app1Found = true
			}
		}

		assert.True(t, app1Found, "should have found 'app-1'")
	}
}

func Test_changes_changedApplication(t *testing.T) {
	actualApp := SampleApp("app-1")
	desiredApp := SampleApp("app-1")
	desiredApp.Image.Tag = "v2.0.0"

	actual := &state.Spec{
		Applications: []resource.Application{
			*actualApp,
		},
	}

	desired := &state.Spec{
		Applications: []resource.Application{
			*desiredApp,
		},
	}

	c := compare(desired, actual)
	if assert.NotNil(t, c, "should not return nil") {
		assert.Equal(t, 1, len(c.apps.changed), "should have found one changed resource")

		app1Found := false
		for _, app := range c.apps.changed {
			if app.Name == "app-1" {
				app1Found = true
			}
		}

		assert.True(t, app1Found, "should have found 'app-1'")
	}
}

func Test_changes_removedApplication(t *testing.T) {
	actual := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}
	desired := state.EmptySpec()

	c := compare(desired, actual)
	if assert.NotNil(t, c, "should not return nil") {
		assert.Equal(t, 1, len(c.apps.removed), "should have found one removed resource")

		app1Found := false
		for _, app := range c.apps.removed {
			if app.Name == "app-1" {
				app1Found = true
			}
		}

		assert.True(t, app1Found, "should have found 'app-1'")
	}
}

func Test_changes_nilDesiredState(t *testing.T) {
	desired := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}

	c := compare(desired, nil)
	if assert.NotNil(t, c, "should not return nil") {
		assert.Equal(t, 1, len(c.apps.added), "should have found one added resource")
	}
}

func Test_changes_nilActualState(t *testing.T) {
	actual := &state.Spec{
		Applications: []resource.Application{
			*SampleApp("app-1"),
		},
	}

	c := compare(nil, actual)
	if assert.NotNil(t, c, "should not return nil") {
		assert.Equal(t, 1, len(c.apps.removed), "should have found one removed resource")
	}
}

func Test_changes_newFeature(t *testing.T) {
	desired := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{
				Labels: "agent=fluent-bit",
			},
		},
	}

	c := compare(desired, nil)
	assert.Equal(t, 0, len(c.features.changed))
	assert.Equal(t, 0, len(c.features.unchanged))
	assert.Equal(t, 0, len(c.features.removed))
	if assert.Equal(t, 1, len(c.features.added), "should have added one feature") {

		if fb, ok := c.features.added[0].(*feature.FluentBit); ok {
			assert.Equal(t, "agent=fluent-bit", fb.Labels)
		}

	}
}

func Test_changes_removedFeature(t *testing.T) {
	actual := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{
				Labels: "agent=fluent-bit",
			},
		},
	}

	c := compare(nil, actual)
	assert.Equal(t, 0, len(c.features.changed))
	assert.Equal(t, 0, len(c.features.unchanged))
	assert.Equal(t, 0, len(c.features.added))
	if assert.Equal(t, 1, len(c.features.removed), "should have removed one feature") {

		if fb, ok := c.features.removed[0].(*feature.FluentBit); ok {
			assert.Equal(t, "agent=fluent-bit", fb.Labels)
		}

	}
}

func Test_changes_changedFeature(t *testing.T) {
	desired := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{
				Labels: "agent=fluent-bit",
			},
		},
	}

	actual := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{
				Labels: "another=label",
			},
		},
	}

	c := compare(desired, actual)
	assert.Equal(t, 0, len(c.features.added))
	assert.Equal(t, 0, len(c.features.unchanged))
	assert.Equal(t, 0, len(c.features.removed))
	if assert.Equal(t, 1, len(c.features.changed)) {

		if fb, ok := c.features.changed[0].(*feature.FluentBit); ok {
			assert.Equal(t, "agent=fluent-bit", fb.Labels)
		}
	}
}

func Test_changes_unchangedFeature(t *testing.T) {
	desired := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{
				Labels: "agent=fluent-bit",
			},
		},
	}

	actual := &state.Spec{
		Applications: make([]resource.Application, 0),
		Feature: state.Feature{
			FluentBit: &feature.FluentBit{
				Labels: "agent=fluent-bit",
			},
		},
	}

	c := compare(desired, actual)
	assert.Equal(t, 0, len(c.features.changed))
	assert.Equal(t, 0, len(c.features.added))
	assert.Equal(t, 0, len(c.features.removed))
	if assert.Equal(t, 1, len(c.features.unchanged)) {

		if fb, ok := c.features.unchanged[0].(*feature.FluentBit); ok {
			assert.Equal(t, "agent=fluent-bit", fb.Labels)
		}
	}
}
