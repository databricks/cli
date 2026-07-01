// Package hash provides deterministic content-hashing helpers shared across the CLI.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// OfJSON returns the hex-encoded SHA-256 of v's JSON encoding. json.Marshal sorts
// map keys, so equal content yields an equal hash across runs and platforms,
// which makes the result safe to use as a cache key or a state fingerprint.
func OfJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("hashing value: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
