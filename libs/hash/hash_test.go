package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOfJSONStable tests that the hash of the same value is the same across runs.
func TestOfJSONDeterministic(t *testing.T) {
	v := struct {
		Key string `json:"key"`
	}{
		Key: "test-key",
	}

	hash, err := OfJSON(v)
	require.NoError(t, err)

	hashAgain, err := OfJSON(v)
	require.NoError(t, err)

	assert.Equal(t, hash, hashAgain)
}

// TestOfJSONReordering tests that the hash of a map is the same regardless of the order of the keys.
func TestOfJSONReordering(t *testing.T) {
	v1 := map[string]int{"a": 1, "b": 2}
	v2 := map[string]int{"b": 2, "a": 1}

	hash1, err := OfJSON(v1)
	require.NoError(t, err)

	hash2, err := OfJSON(v2)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2)
}

// TestOfJSONDifferentValues tests that the hash of different values is different.
func TestOfJSONDifferentValues(t *testing.T) {
	v1 := struct {
		Key string `json:"key"`
	}{
		Key: "test-key",
	}

	v2 := struct {
		Key string `json:"key"`
	}{
		Key: "test-key2",
	}

	hash1, err := OfJSON(v1)
	require.NoError(t, err)

	hash2, err := OfJSON(v2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
}

// TestOfJSONKnownValues tests that the hash of a known value is the expected value.
func TestOfJSONKnownValues(t *testing.T) {
	v := struct {
		Key string `json:"key"`
	}{
		Key: "test-key",
	}

	expectedHash := "1b329dc07a9fa87da7480f6b10cc917a40a4f460ac82aea3d09df477764f3101"

	hash, err := OfJSON(v)
	require.NoError(t, err)
	assert.Equal(t, expectedHash, hash)
}

// TestOfJSONUnsupportedTypes tests that the hash of an unsupported type is an error.
func TestOfJSONUnsupportedTypes(t *testing.T) {
	v := func() any {
		return nil
	}

	hash, err := OfJSON(v)
	require.Equal(t, "", hash)
	assert.Error(t, err)
}
