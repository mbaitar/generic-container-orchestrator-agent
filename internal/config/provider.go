package config

type DockerProvider struct {
	// Enabled is used to enable or disable the docker provider.
	Enabled bool
	// UseDockerComposeGrouping will add the 'gco' label to the managed containers resulting in a grouped view with docker desktop.
	UseDockerComposeGrouping bool
}
