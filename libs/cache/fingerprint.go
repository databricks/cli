package cache

import "github.com/databricks/cli/libs/hash"

// fingerprintToHash converts any struct to a deterministic string representation for use as a cache key.
func fingerprintToHash(fingerprint any) (string, error) {
	return hash.JSON(fingerprint)
}
