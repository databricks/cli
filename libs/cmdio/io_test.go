package cmdio

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestIsPromptSupportedFalseForGitBash(t *testing.T) {
	ctx := context.Background()
	ctx, _ = SetupTest(ctx)

	assert.True(t, IsPromptSupported(ctx))

	ctx = env.Set(ctx, "MSYSTEM", "MINGW64")
	ctx = env.Set(ctx, "TERM", "xterm")
	ctx = env.Set(ctx, "PS1", "\\[\033]0;$TITLEPREFIX:$PWD\007\\]\n\\[\033[32m\\]\\u@\\h \\[\033[35m\\]$MSYSTEM \\[\033[33m\\]\\w\\[\033[36m\\]`__git_ps1`\\[\033[0m\\]\n$")
	assert.False(t, IsPromptSupported(ctx))
}
