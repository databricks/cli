package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPanics(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "The function did not panic")
		assert.Equal(t, "context not configured with bundle", r)
	}()

	Get(t.Context())
}

func TestGetSuccess(t *testing.T) {
	ctx := Context(t.Context(), &Bundle{})
	require.NotNil(t, Get(ctx))
}
