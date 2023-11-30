package config

import (
	"revengy.io/gco/agent/internal/flag"
)

type Config struct {
	// General reflects the general agent configuration.
	General General
	// Grpc reflects the configuration for the gRPC server.
	Grpc Grpc
	// Http reflects the configuration for the HTTP server.
	Http Http
	// Docker reflects the configuration when the docker provider has been enabled.
	Docker DockerProvider
}

// DefaultConfig returns the default configuration for the agent.
func DefaultConfig() *Config {
	return &Config{
		General: General{
			ResetProviderOnStartup: false,
		},
		Grpc: Grpc{
			Enabled:          true,
			Port:             9000,
			Address:          "0.0.0.0",
			EnableReflection: true,
		},
		Http: Http{
			Enabled: true,
			Port:    8080,
			Address: "0.0.0.0",
		},
		Docker: DockerProvider{
			Enabled:                  true,
			UseDockerComposeGrouping: true,
		},
	}
}

func (c *Config) SetFlags() {
	// reset all flags before continuing
	flag.Reset()

	if c.Docker.Enabled {
		flag.Set(flag.IgnoreInstanceDiff)
	}

	if c.General.ResetProviderOnStartup {
		flag.Set(flag.RemoveAllOnStartup)
	}

	flag.Set(flag.ColoredLogs)
}
