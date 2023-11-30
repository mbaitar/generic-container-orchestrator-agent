package state

import "revengy.io/gco/agent/pkg/feature"

type Feature struct {
	// FluentBit specifies the fluent-bit feature configuration for the agent.
	FluentBit *feature.FluentBit `json:"fluentBit,omitempty"`
}
