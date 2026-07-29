// Package hash provides deterministic content hashing helpers.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// OfJSON returns a deterministic sha256 hex digest of v's JSON encoding.
// json.Marshal sorts map keys, so the digest is stable across runs for equal values.
func OfJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal value for hashing: %w", err)
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
