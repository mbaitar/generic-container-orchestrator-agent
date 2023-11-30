package hash

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
)

// CalculateHash calculates a hash based on the used input map.
func CalculateHash(input interface{}) string {
	// write input as JSON
	bytes, err := json.Marshal(input)
	if err != nil {
		// return empty string hash
		return ""
	}

	return CalculateHashFromBytes(bytes)
}

// CalculateHashFromString calculates a hash based on the used input string.
func CalculateHashFromString(input string) string {
	return CalculateHashFromBytes([]byte(input))
}

// CalculateHashFromBytes calculates a hash based on the used input bytes.
func CalculateHashFromBytes(bytes []byte) string {
	// calculate hash using MD5
	h := md5.New()
	h.Write(bytes)

	// write hash hex encoded
	return hex.EncodeToString(h.Sum(nil))
}
