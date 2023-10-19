package root

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipPrompt(t *testing.T) {
	ctx := context.Background()
	assert.False(t, shouldSkipPrompt(ctx))

	ctx = SkipPrompt(ctx)
	assert.True(t, shouldSkipPrompt(ctx))
}

func TestSkipLoadBundle(t *testing.T) {
	ctx := context.Background()
	assert.False(t, shouldSkipLoadBundle(ctx))

	ctx = SkipLoadBundle(ctx)
	assert.True(t, shouldSkipLoadBundle(ctx))
}
