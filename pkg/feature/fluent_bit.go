package feature

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mabaitar/gco/agent/internal/hash"
	"github.com/mabaitar/gco/agent/pkg/resource"
)

type FluentBit struct {
	hash string

	// LogLevel sets the log level.
	LogLevel string `json:"logLevel"`

	// Labels specifies a key-value comma seperated list for the default labels communicated to the output.
	Labels string `json:"labels"`

	// Version specifies the version of fluent-bit to use.
	Version string `json:"version"`

	// Output specifies the output configuration.
	Output map[string]string `json:"output"`
}

func (fb *FluentBit) ConfigHash() string {
	if fb.hash == "" {
		fb.hash = hash.CalculateHash(fb)
	}

	return fb.hash
}

func (fb *FluentBit) Name() string {
	return NameFluentBit
}

// DefaultLogConfig creates the default logging configuration based on the current FluentBit config.
func (fb *FluentBit) DefaultLogConfig() *resource.LogConfig {
	return &resource.LogConfig{
		Driver:   resource.FluentdLogDriver,
		Disabled: false,
		Config: map[string]string{
			"address": "127.0.0.1:24224",
		},
	}
}

// CreateConfig creates a string version of the configuration required for fluent-bit.
// See configuration syntax at: https://docs.fluentbit.io/manual
func (fb *FluentBit) CreateConfig() string {
	var builder strings.Builder

	// [SERVICE] section
	builder.WriteString(fb.writeConfigHeader("SERVICE"))
	builder.WriteString(fb.writeConfigPropWithDefault("log_level", fb.LogLevel, "info"))
	builder.WriteString(fb.writeEndConfigHeader())

	// [INPUT] section
	builder.WriteString(fb.writeConfigHeader("INPUT"))
	builder.WriteString(fb.writeConfigProp("name", "forward"))
	builder.WriteString(fb.writeConfigProp("listen", "0.0.0.0"))
	builder.WriteString(fb.writeConfigProp("port", "24224"))
	builder.WriteString(fb.writeEndConfigHeader())

	// [OUTPUT] section
	if len(fb.Output) > 0 {
		builder.WriteString(fb.writeConfigHeader("OUTPUT"))
		builder.WriteString(fb.writeMapAsConfigProp(fb.Output))
		builder.WriteString(fb.writeConfigPropWithDefault("labels", fb.Labels, "agent=fluent-bit"))
		builder.WriteString(fb.writeEndConfigHeader())
	}

	return strings.Trim(builder.String(), "\n") + "\n"
}

// writeConfigHeader writes a configuration header according to the fluent-bit syntax.
func (fb *FluentBit) writeConfigHeader(key string) string {
	header := fmt.Sprintf("[%s]", strings.ToUpper(key))
	return header + "\n"
}

// writeConfigPropWithDefault writes a configuration property and falls back to the defaultValue if value is empty.
func (fb *FluentBit) writeConfigPropWithDefault(key string, value string, defaultValue string) string {
	finalValue := value
	if finalValue == "" {
		finalValue = defaultValue
	}

	return fb.writeConfigProp(key, finalValue)
}

// writeConfigProp writes a configuration property according to the fluent-bit syntax.
func (fb *FluentBit) writeConfigProp(key string, value string) string {
	entry := fmt.Sprintf("%s %s", strings.ToLower(key), value)
	return "\t" + entry + "\n"
}

// writeEndConfigHeader writes the end of a configuration header.
func (fb *FluentBit) writeEndConfigHeader() string {
	return "\n"
}

// writeMapAsConfigProp writes the map as configuration properties but sorts the keys for a stable result.
func (fb *FluentBit) writeMapAsConfigProp(entries map[string]string) string {
	keys := make([]string, 0)

	for key := range entries {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	output := ""
	for _, key := range keys {
		value := entries[key]
		output += fb.writeConfigProp(key, value)
	}

	return output
}
