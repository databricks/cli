package project

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectInitialize(t *testing.T) {
	ctx, err := Initialize(context.Background(), "./testdata", DefaultEnvironment)
	require.NoError(t, err)
	assert.Equal(t, Get(ctx).config.Name, "dev")
}
