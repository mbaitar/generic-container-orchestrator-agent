package docker

import (
	"github.com/stretchr/testify/assert"
	"revengy.io/gco/agent/pkg/feature"
	"revengy.io/gco/agent/pkg/resource"
	"strings"
	"testing"
)

type UnsupportedFeature struct {
}

func (u *UnsupportedFeature) ConfigHash() string {
	return "empty"
}

func (u *UnsupportedFeature) Name() string {
	return "unsupported"
}

func TestProvider_createFluentBitContainer(t *testing.T) {
	SetupForTests(t)
	client := NewTestClient()
	provider := &Provider{client: client}

	fluentBit := &feature.FluentBit{
		LogLevel: "info",
		Labels:   "agent=fluent-bit",
		Version:  "2.0.0",
	}

	ic, err := provider.createFluentBitContainer(fluentBit)
	assert.Nil(t, err, "should not have thrown an error")
	if assert.NotNil(t, ic, "should not have returned a nil container") {

		assert.Equal(t, string(resource.FeatureKind), ic.getLabel(kindLabelTag), "should have 'feature' kind label")
		assert.Equal(t, fluentBit.Name(), ic.getLabel(featureLabelTag), "should have 'fluent-bit' as feature name")
		assert.Equal(t, "gco.io.fluent-bit", ic.name, "should have container name 'gco.io.fluent-bit'")
		assert.Equal(t, "cr.fluentbit.io/fluent/fluent-bit:2.0.0", ic.image)

		// should expose port '24224'
		if assert.Equal(t, 1, len(ic.ports), "should have exposed the forward protocol port") {
			port := ic.ports[0]
			assert.Equal(t, "tcp", port.protocol())
			assert.Equal(t, "24224", port.privatePort())
			assert.Equal(t, "24224", port.publicPort())
		}

		// should create volume mount
		if assert.Equal(t, 1, len(ic.volumes), "should have created a volume mount") {
			mount := ic.volumes[0]
			assert.Equal(t, "/fluent-bit/etc/fluent-bit.conf", mount.destination)
			assert.True(t, mount.readonly, "should be read-only")
			assert.True(t, strings.HasSuffix(mount.source, "fluent-bit.conf"), "should have created fluent-bit.conf")
		}

	}
}
