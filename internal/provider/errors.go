package provider

import "errors"

var (
	ErrFeatureNotSupported = errors.New("feature is not supported by provider")
	ErrFeatureNotFound     = errors.New("feature not found")
	ErrAppNotFound         = errors.New("application not found")
)
