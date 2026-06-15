package dresources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashStateValue(t *testing.T) {
	stringHash, err := hashStateValue("hello")
	require.NoError(t, err)
	require.IsType(t, "", stringHash)
	assert.True(t, strings.HasPrefix(stringHash.(string), stateHashPrefix))

	// Same content always hashes to the same value.
	again, err := hashStateValue("hello")
	require.NoError(t, err)
	assert.Equal(t, stringHash, again)

	// Different content hashes differently.
	other, err := hashStateValue("world")
	require.NoError(t, err)
	assert.NotEqual(t, stringHash, other)

	// A map hashes deterministically regardless of literal key order.
	mapHash, err := hashStateValue(map[string]any{"a": 1, "b": 2})
	require.NoError(t, err)
	mapHash2, err := hashStateValue(map[string]any{"b": 2, "a": 1})
	require.NoError(t, err)
	assert.Equal(t, mapHash, mapHash2)
}

func TestHashStateValueIdempotent(t *testing.T) {
	hashed, err := hashStateValue("some content")
	require.NoError(t, err)

	// Re-hashing a placeholder returns it unchanged.
	again, err := hashStateValue(hashed)
	require.NoError(t, err)
	assert.Equal(t, hashed, again)
}

func TestHashStateValueEmptyAndNil(t *testing.T) {
	empty, err := hashStateValue("")
	require.NoError(t, err)
	assert.Empty(t, empty)

	null, err := hashStateValue(nil)
	require.NoError(t, err)
	assert.Nil(t, null)
}
