package state

import "github.com/mabaitar/gco/agent/pkg/feature"

type Feature struct {
	// FluentBit specifies the fluent-bit feature configuration for the agent.
	FluentBit *feature.FluentBit `json:"fluentBit,omitempty"`
}
