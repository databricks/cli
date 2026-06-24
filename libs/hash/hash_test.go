package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONStable(t *testing.T) {
	// Equal content (including maps built in a different key order) must hash equal.
	a, err := JSON(map[string]int{"a": 1, "b": 2})
	require.NoError(t, err)
	b, err := JSON(map[string]int{"b": 2, "a": 1})
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestJSONDistinct(t *testing.T) {
	a, err := JSON("hello")
	require.NoError(t, err)
	b, err := JSON("world")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestJSONKnownVector(t *testing.T) {
	// sha256 of the JSON encoding of the string "hello", i.e. of the 7 bytes `"hello"`.
	h, err := JSON("hello")
	require.NoError(t, err)
	assert.Equal(t, "5aa762ae383fbb727af3c7a36d4940a5b8c40a989452d2304fc958ff3f354e7a", h)
}

func TestJSONUnmarshalableErrors(t *testing.T) {
	// Channels cannot be JSON-encoded, so JSON must surface the marshal error.
	_, err := JSON(make(chan int))
	assert.Error(t, err)
}
