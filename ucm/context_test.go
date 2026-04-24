package ucm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextGetOrNilReturnsNilWhenUnset(t *testing.T) {
	assert.Nil(t, GetOrNil(t.Context()))
}

func TestContextReturnsAttachedUcm(t *testing.T) {
	u := &Ucm{}
	ctx := Context(t.Context(), u)
	require.Same(t, u, Get(ctx))
}

func TestContextGetPanicsWhenUnset(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "The function did not panic")
		assert.Equal(t, "context not configured with ucm", r)
	}()

	Get(t.Context())
}
