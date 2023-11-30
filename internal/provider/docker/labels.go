package docker

import (
	"fmt"
	"revengy.io/gco/agent/pkg/resource"
)

// labelPrefix is the prefix used for the labels associated with GCO
const labelPrefix = "gco.io"

type labelTag string

func (t labelTag) string() string {
	return string(t)
}

func platformLabelTag(key string) labelTag {
	return labelTag(fmt.Sprintf("%s/%s", labelPrefix, key))
}

type label struct {
	tag   labelTag
	value string
}

var (
	managedByLabelTag = platformLabelTag("managed-by")
	nameLabelTag      = platformLabelTag("name")
	kindLabelTag      = platformLabelTag("kind")
	featureLabelTag   = platformLabelTag("feature")
	configLabelTag    = platformLabelTag("config")

	composeProjectLabelTag labelTag = "com.docker.compose.project"
)

func (l label) string() string {
	return fmt.Sprintf("%s=%s", l.tag.string(), l.value)
}

func managedByLabel() label {
	return label{tag: managedByLabelTag, value: "gco"}
}

func nameLabel(name string) label {
	return label{tag: nameLabelTag, value: name}
}

func kindLabel(kind resource.Kind) label {
	return label{tag: kindLabelTag, value: string(kind)}
}

func featureLabel(name string) label {
	return label{tag: featureLabelTag, value: name}
}

func configLabel(value string) label {
	return label{tag: configLabelTag, value: value}
}

func composeProjectLabel() label {
	return label{tag: composeProjectLabelTag, value: "gco"}
}

func customLabel(key string, value string) label {
	return label{tag: labelTag(key), value: value}
}
