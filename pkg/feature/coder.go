package feature

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
)

// EncodeFeature encodes the feature structure to bytes and returns the hex representation of that structure.
func EncodeFeature(f Feature) string {
	buffer := bytes.NewBuffer(make([]byte, 0))
	encoder := gob.NewEncoder(buffer)

	err := encoder.Encode(f)
	if err != nil {
		return ""
	} else {
		return hex.EncodeToString(buffer.Bytes())
	}
}

// DecodeFeature decodes the hex representation of that structure and returns the created structure.
func DecodeFeature(hexInput string, f Feature) Feature {
	content, err := hex.DecodeString(hexInput)
	if err != nil {
		return f
	}

	buffer := bytes.NewBuffer(content)
	decoder := gob.NewDecoder(buffer)
	decoder.Decode(f)
	return f
}
