package state

import (
	"testing"

	"github.com/mabaitar/gco/agent/pkg/feature"
	"github.com/mabaitar/gco/agent/pkg/resource"
	"github.com/stretchr/testify/assert"
)

func TestSpec_Evaluate_shouldAppendFluentBitConfig(t *testing.T) {

	spec := &Spec{
		Applications: []resource.Application{
			{Name: "app-1", Image: resource.Image{Name: "nginx", Tag: "latest"}},
		},
		Feature: Feature{
			FluentBit: &feature.FluentBit{
				LogLevel: "info",
			},
		},
	}

	spec.Evaluate()

	// should append the log config to the application
	app := spec.GetApplication("app-1")
	if assert.NotNil(t, app, "should find 'app-1'") {
		if assert.NotNil(t, app.LogConfig, "should have log config populated") {
			assert.Equal(t, "fluentd", app.LogConfig.Driver)
			assert.False(t, app.LogConfig.Disabled)
			assert.Equal(t, "127.0.0.1:24224", app.LogConfig.Config["address"])
		}
	}

}

func TestSpec_Evaluate_shouldNotOverwriteExistingConfig(t *testing.T) {
	spec := &Spec{
		Applications: []resource.Application{
			{
				Name:  "app-1",
				Image: resource.Image{Name: "nginx", Tag: "latest"},
				LogConfig: &resource.LogConfig{
					Driver: "custom",
				},
			},
		},
		Feature: Feature{
			FluentBit: &feature.FluentBit{
				LogLevel: "info",
			},
		},
	}

	spec.Evaluate()

	// should append the log config to the application
	app := spec.GetApplication("app-1")
	if assert.NotNil(t, app, "should find 'app-1'") {
		if assert.NotNil(t, app.LogConfig, "should have log config populated") {
			assert.Equal(t, "custom", app.LogConfig.Driver)
			assert.False(t, app.LogConfig.Disabled)
		}
	}
}
