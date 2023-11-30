package hash

// ShortHash returns the truncated version of the hash.
func ShortHash(hash string) string {
	return hash[0:10]
}
