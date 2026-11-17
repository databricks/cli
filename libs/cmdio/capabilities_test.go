package cmdio

import (
	"io"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestNewCapabilities_Env(t *testing.T) {
	// Pin both variables so the developer's shell can't leak in.
	ctx := env.Set(env.Set(t.Context(), "NO_COLOR", ""), "TERM", "")
	c := newCapabilities(ctx, nil, nil, nil)
	assert.False(t, c.noColor)
	assert.False(t, c.dumb)

	c = newCapabilities(env.Set(ctx, "NO_COLOR", "1"), nil, nil, nil)
	assert.True(t, c.noColor)
	assert.False(t, c.dumb)

	c = newCapabilities(env.Set(ctx, "TERM", "dumb"), nil, nil, nil)
	assert.False(t, c.noColor)
	assert.True(t, c.dumb)
}

func TestCapabilities_SupportsInteractive(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		expected bool
	}{
		{"normal terminal", Capabilities{stderrIsTTY: true}, true},
		{"NO_COLOR only removes color", Capabilities{stderrIsTTY: true, noColor: true}, true},
		{"TERM=dumb can't redraw", Capabilities{stderrIsTTY: true, dumb: true}, false},
		{"stderr not TTY", Capabilities{stderrIsTTY: false}, false},
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
		{"full TTY support", Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true}, true},
		{"MSYS2 pipe stdin blocks prompts", Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true, stdinIsMSYSPipe: true}, false},
		{"stdin not TTY", Capabilities{stdoutIsTTY: true, stderrIsTTY: true}, false},
		{"stdout not TTY (prompts write to stderr, so this is OK)", Capabilities{stdinIsTTY: true, stderrIsTTY: true}, true},
		{"stderr not TTY", Capabilities{stdinIsTTY: true, stdoutIsTTY: true}, false},
		{"NO_COLOR still prompts", Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true, noColor: true}, true},
		{"TERM=dumb", Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true, dumb: true}, false},
		{"no TTY support at all", Capabilities{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.SupportsPrompt())
		})
	}
}

func TestCapabilities_SupportsPager(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		expected bool
	}{
		{"full TTY support", Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true}, true},
		{"NO_COLOR keeps the pager (uncolored)", Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true, noColor: true}, true},
		{"TERM=dumb can't redraw", Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true, dumb: true}, false},
		{"stdout redirected", Capabilities{stdinIsTTY: true, stderrIsTTY: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.SupportsPager())
		})
	}
}

func TestCapabilities_SupportsColor(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		writer   io.Writer
		expected bool
	}{
		{"TTY with no env restrictions", Capabilities{}, FakeTTY(io.Discard), true},
		{"TTY with NO_COLOR set", Capabilities{noColor: true}, FakeTTY(io.Discard), false},
		{"TTY with TERM=dumb", Capabilities{dumb: true}, FakeTTY(io.Discard), false},
		{"non-TTY writer", Capabilities{}, io.Discard, false},
		{"non-TTY with NO_COLOR", Capabilities{noColor: true}, io.Discard, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.SupportsColor(tt.writer))
		})
	}
}

func TestCapabilities_SupportsStreamColor(t *testing.T) {
	tests := []struct {
		name           string
		caps           Capabilities
		stdout, stderr bool
	}{
		{"both TTYs", Capabilities{stdoutIsTTY: true, stderrIsTTY: true}, true, true},
		{"NO_COLOR", Capabilities{stdoutIsTTY: true, stderrIsTTY: true, noColor: true}, false, false},
		{"only stdout TTY", Capabilities{stdoutIsTTY: true}, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.stdout, tt.caps.SupportsStdoutColor())
			assert.Equal(t, tt.stderr, tt.caps.SupportsStderrColor())
		})
	}
}

func TestCapabilities_InteractiveMode(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		expected InteractiveMode
	}{
		{"full interactive", Capabilities{stdinIsTTY: true, stderrIsTTY: true}, InteractiveModeFull},
		{"full interactive with NO_COLOR", Capabilities{stdinIsTTY: true, stderrIsTTY: true, noColor: true}, InteractiveModeFull},
		{"output only - stdin not TTY", Capabilities{stdinIsTTY: false, stderrIsTTY: true}, InteractiveModeOutputOnly},
		{"output only - MSYS2 pipe stdin", Capabilities{stdinIsTTY: true, stderrIsTTY: true, stdinIsMSYSPipe: true}, InteractiveModeOutputOnly},
		{"none - stderr not TTY", Capabilities{stdinIsTTY: true, stderrIsTTY: false}, InteractiveModeNone},
		{"none - TERM=dumb", Capabilities{stdinIsTTY: true, stderrIsTTY: true, dumb: true}, InteractiveModeNone},
		{"none - no TTY support at all", Capabilities{}, InteractiveModeNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.InteractiveMode())
		})
	}
}
