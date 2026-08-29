package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	s := NewSession()

	_, ok := s.Get("missing")
	assert.False(t, ok)
}

func TestSessionGetAndSet(t *testing.T) {
	s := NewSession()
	s.Set("warehouse_endpoint", "abc")

	value, ok := s.Get("warehouse_endpoint")
	require.True(t, ok)
	assert.Equal(t, "abc", value)
}

func TestGetSession(t *testing.T) {
	s := NewSession()
	ctx := WithSession(t.Context(), s)

	got, err := GetSession(ctx)
	require.NoError(t, err)
	assert.Same(t, s, got)
}

func TestGetSessionMissing(t *testing.T) {
	_, err := GetSession(t.Context())
	assert.EqualError(t, err, "session not found in context")
}
