package bundle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPanics(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "The function did not panic")
		assert.Equal(t, r, "context not configured with bundle")
	}()

	Get(context.Background())
}

func TestGetSuccess(t *testing.T) {
	ctx := Context(context.Background(), &Bundle{})
	require.NotNil(t, Get(ctx))
}
