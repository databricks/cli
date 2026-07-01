package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfJSONStable(t *testing.T) {
	// Equal content (including maps built in a different key order) must hash equal.
	a, err := OfJSON(map[string]int{"a": 1, "b": 2})
	require.NoError(t, err)
	b, err := OfJSON(map[string]int{"b": 2, "a": 1})
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestOfJSONDistinct(t *testing.T) {
	a, err := OfJSON("hello")
	require.NoError(t, err)
	b, err := OfJSON("world")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestOfJSONKnownVector(t *testing.T) {
	// sha256 of the JSON encoding of the string "hello", i.e. of the 7 bytes `"hello"`.
	h, err := OfJSON("hello")
	require.NoError(t, err)
	assert.Equal(t, "5aa762ae383fbb727af3c7a36d4940a5b8c40a989452d2304fc958ff3f354e7a", h)
}

func TestOfJSONStructKnownVector(t *testing.T) {
	// A struct hashes over its JSON encoding (honoring json tags), locking the wire format.
	h, err := OfJSON(struct {
		Key string `json:"key"`
	}{Key: "test-key"})
	require.NoError(t, err)
	assert.Equal(t, "1b329dc07a9fa87da7480f6b10cc917a40a4f460ac82aea3d09df477764f3101", h)
}

func TestOfJSONUnmarshalableErrors(t *testing.T) {
	// Channels cannot be JSON-encoded, so OfJSON must surface the marshal error.
	_, err := OfJSON(make(chan int))
	assert.Error(t, err)
}
