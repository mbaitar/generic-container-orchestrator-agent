package resource

// Kind defines the type of resource in relation to the system.
type Kind string

var (
	ApplicationKind Kind = "app"
	FeatureKind     Kind = "feature"
)
