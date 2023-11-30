package feature

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeAndDecodeFeature(t *testing.T) {
	fb := &FluentBit{
		LogLevel: "info",
	}

	encoded := EncodeFeature(fb)
	assert.NotEqual(t, "", encoded, "should have returned a valid encoded structure")

	decodedFb := &FluentBit{}
	DecodeFeature(encoded, decodedFb)
	assert.Equal(t, "info", decodedFb.LogLevel)
}

func TestEncodeNilFeature(t *testing.T) {
	encoded := EncodeFeature(nil)
	assert.Equal(t, "", encoded, "should have returned an empty encoded structure")
}

func TestDecodeFeature_invalidHexData(t *testing.T) {
	fb := &FluentBit{LogLevel: "info"}
	DecodeFeature("invalid", fb)

	assert.Equal(t, "info", fb.LogLevel, "should not have been modified")
}
