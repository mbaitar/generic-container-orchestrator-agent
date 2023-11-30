package feature

const (
	NameFluentBit = "fluent-bit"
)

type Feature interface {
	// ConfigHash calculates the hash for the feature configuration.
	ConfigHash() string

	// Name returns the name of the feature.
	Name() string
}
