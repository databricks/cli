package cmdio

import (
	"context"
	"testing"

	"github.com/databricks/bricks/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestTryResolveDefaultToInplaceWithInplaceSupported(t *testing.T) {
	ctx := NewContext(context.Background(), NewLogger(flags.ModeDefault, true))

	logger, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, flags.ModeDefault, logger.Mode)

	TryResolveDefaultToInplace(ctx)
	logger, ok = FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, flags.ModeInplace, logger.Mode)
}

func TestTryResolveDefaultToInplaceWithInplaceNotSupported(t *testing.T) {
	ctx := NewContext(context.Background(), NewLogger(flags.ModeDefault, false))

	logger, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, flags.ModeDefault, logger.Mode)

	TryResolveDefaultToInplace(ctx)
	logger, ok = FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, flags.ModeAppend, logger.Mode)
}

func TestResolveDefaultToAppend(t *testing.T) {
	ctx := NewContext(context.Background(), NewLogger(flags.ModeDefault, true))

	logger, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, flags.ModeDefault, logger.Mode)

	ResolveDefaultToAppend(ctx)
	logger, ok = FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, flags.ModeAppend, logger.Mode)
}
