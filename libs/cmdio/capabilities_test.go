package cmdio

import (
	"context"
	"io"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestCapabilities_SupportsInteractive(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		expected bool
	}{
		{
			name:     "normal terminal",
			caps:     Capabilities{stderrIsTTY: true, color: true},
			expected: true,
		},
		{
			name:     "NO_COLOR set",
			caps:     Capabilities{stderrIsTTY: true, color: false},
			expected: false,
		},
		{
			name:     "stderr not TTY",
			caps:     Capabilities{stderrIsTTY: false, color: true},
			expected: false,
		},
		{
			name:     "both NO_COLOR and no TTY",
			caps:     Capabilities{stderrIsTTY: false, color: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.SupportsInteractive())
		})
	}
}

func TestCapabilities_SupportsPrompt(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		expected bool
	}{
		{
			name: "full TTY support",
			caps: Capabilities{
				stdinIsTTY:  true,
				stdoutIsTTY: true,
				stderrIsTTY: true,
				color:       true,
				isGitBash:   false,
			},
			expected: true,
		},
		{
			name: "Git Bash blocks prompts",
			caps: Capabilities{
				stdinIsTTY:  true,
				stdoutIsTTY: true,
				stderrIsTTY: true,
				color:       true,
				isGitBash:   true,
			},
			expected: false,
		},
		{
			name: "stdin not TTY",
			caps: Capabilities{
				stdinIsTTY:  false,
				stdoutIsTTY: true,
				stderrIsTTY: true,
				color:       true,
				isGitBash:   false,
			},
			expected: false,
		},
		{
			name: "stdout not TTY",
			caps: Capabilities{
				stdinIsTTY:  true,
				stdoutIsTTY: false,
				stderrIsTTY: true,
				color:       true,
				isGitBash:   false,
			},
			expected: false,
		},
		{
			name: "stderr not TTY",
			caps: Capabilities{
				stdinIsTTY:  true,
				stdoutIsTTY: true,
				stderrIsTTY: false,
				color:       true,
				isGitBash:   false,
			},
			expected: false,
		},
		{
			name: "NO_COLOR set",
			caps: Capabilities{
				stdinIsTTY:  true,
				stdoutIsTTY: true,
				stderrIsTTY: true,
				color:       false,
				isGitBash:   false,
			},
			expected: false,
		},
		{
			name: "no TTY support at all",
			caps: Capabilities{
				stdinIsTTY:  false,
				stdoutIsTTY: false,
				stderrIsTTY: false,
				color:       true,
				isGitBash:   false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.SupportsPrompt())
		})
	}
}

func TestDetectGitBash(t *testing.T) {
	ctx := context.Background()
	assert.False(t, detectGitBash(ctx))

	ctx = env.Set(ctx, "MSYSTEM", "MINGW64")
	ctx = env.Set(ctx, "TERM", "xterm")
	ctx = env.Set(ctx, "PS1", "\\[\033]0;$TITLEPREFIX:$PWD\007\\]\n\\[\033[32m\\]\\u@\\h \\[\033[35m\\]$MSYSTEM \\[\033[33m\\]\\w\\[\033[36m\\]`__git_ps1`\\[\033[0m\\]\n$")
	assert.True(t, detectGitBash(ctx))
}

func TestCapabilities_SupportsColor(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		writer   io.Writer
		expected bool
	}{
		{
			name:     "TTY with no env restrictions",
			caps:     Capabilities{color: true},
			writer:   FakeTTY(io.Discard),
			expected: true,
		},
		{
			name:     "TTY with NO_COLOR set",
			caps:     Capabilities{color: false},
			writer:   FakeTTY(io.Discard),
			expected: false,
		},
		{
			name:     "non-TTY writer",
			caps:     Capabilities{color: true},
			writer:   io.Discard,
			expected: false,
		},
		{
			name:     "non-TTY with NO_COLOR",
			caps:     Capabilities{color: false},
			writer:   io.Discard,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.caps.SupportsColor(tt.writer)
			assert.Equal(t, tt.expected, result)
		})
	}
}
