package root

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipPrompt(t *testing.T) {
	ctx := t.Context()
	assert.False(t, shouldSkipPrompt(ctx))

	ctx = SkipPrompt(ctx)
	assert.True(t, shouldSkipPrompt(ctx))
}

func TestSkipLoadBundle(t *testing.T) {
	ctx := t.Context()
	assert.False(t, shouldSkipLoadBundle(ctx))

	ctx = SkipLoadBundle(ctx)
	assert.True(t, shouldSkipLoadBundle(ctx))
}
