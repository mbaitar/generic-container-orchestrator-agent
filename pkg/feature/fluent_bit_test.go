package feature

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFluentBit_CreateConfig(t *testing.T) {
	fb := &FluentBit{LogLevel: "debug"}
	config := fb.CreateConfig()

	expected := `[SERVICE]
	log_level debug

[INPUT]
	name forward
	listen 0.0.0.0
	port 24224
`

	assert.Equal(t, expected, config, "should have written the correct configuration")
}

func TestFluentBit_Name(t *testing.T) {
	fb := &FluentBit{LogLevel: "info"}
	assert.Equal(t, NameFluentBit, fb.Name())
}

func TestFluentBit_ConfigHash(t *testing.T) {
	fb1 := &FluentBit{LogLevel: "debug"}
	fb2 := &FluentBit{LogLevel: "info"}
	assert.NotEqual(t, fb1.ConfigHash(), fb2.ConfigHash(), "hash should not match")
}

func TestFluentBit_CreateConfig_withOutput(t *testing.T) {
	fb := &FluentBit{
		LogLevel: "debug",
		Labels:   "agent=fluent-bit",
		Output: map[string]string{
			"name":  "loki",
			"host":  "host.docker.internal",
			"match": "*",
		},
	}

	expected := `[SERVICE]
	log_level debug

[INPUT]
	name forward
	listen 0.0.0.0
	port 24224

[OUTPUT]
	host host.docker.internal
	match *
	name loki
	labels agent=fluent-bit
`

	assert.Equal(t, expected, fb.CreateConfig(), "should match expected configuration")
}
