package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// fingerprintToHash converts any struct to a deterministic string representation for use as a cache key.
func fingerprintToHash(fingerprint any) (string, error) {
	// Marshal map - json.Marshal sorts map keys alphabetically
	data, err := json.Marshal(fingerprint)
	if err != nil {
		return "", fmt.Errorf("failed to marshal normalized fingerprint: %w", err)
	}

	// Hash for consistent, reasonably-sized key.
	// hash[:] converts the [32]byte array returned by Sum256 to a []byte slice.
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
