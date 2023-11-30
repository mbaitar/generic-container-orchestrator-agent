package docker

// imagePullPolicy describes which policy to use when an image is requested.
type imagePullPolicy string

var (
	alwaysPullPolicy     imagePullPolicy = "always"
	whenNotPresentPolicy imagePullPolicy = "when_not_present"
)
