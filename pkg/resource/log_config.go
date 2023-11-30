package resource

const (
	FluentdLogDriver = "fluentd"
)

// LogConfig reflects the logging configuration for an application.
type LogConfig struct {
	Disabled bool              `json:"disabled"`
	Driver   string            `json:"driver"`
	Config   map[string]string `json:"config,omitempty"`
}
