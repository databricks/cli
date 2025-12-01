package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFingerprintStability tests that the fingerprintToHash function returns the same hash for the same input.
func TestFingerprintStability(t *testing.T) {
	fingerprint1 := struct {
		Key string `json:"key"`
	}{
		Key: "test-key",
	}

	fingerprint2 := struct {
		Key string `json:"key"`
	}{
		Key: "test-key2",
	}

	hash1, err := fingerprintToHash(fingerprint1)
	require.NoError(t, err)
	hash2, err := fingerprintToHash(fingerprint2)
	require.NoError(t, err)
	hash1ToCompare, err := fingerprintToHash(fingerprint1)
	require.NoError(t, err)

	assert.Equal(t, hash1ToCompare, hash1)
	assert.NotEqual(t, hash1, hash2)
}
