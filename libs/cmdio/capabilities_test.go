package cmdio

import (
	"io"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestNewCapabilities_Env(t *testing.T) {
	ctx := t.Context()
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
	tty := Capabilities{stdinIsTTY: true, stdoutIsTTY: true, stderrIsTTY: true}
	with := func(f func(*Capabilities)) Capabilities {
		c := tty
		f(&c)
		return c
	}
	tests := []struct {
		name     string
		caps     Capabilities
		expected bool
	}{
		{"full TTY support", tty, true},
		{"MSYS2 pipe stdin blocks prompts", with(func(c *Capabilities) { c.stdinIsMSYSPipe = true }), false},
		{"stdin not TTY", with(func(c *Capabilities) { c.stdinIsTTY = false }), false},
		{"stdout not TTY (prompts write to stderr, so this is OK)", with(func(c *Capabilities) { c.stdoutIsTTY = false }), true},
		{"stderr not TTY", with(func(c *Capabilities) { c.stderrIsTTY = false }), false},
		{"NO_COLOR still prompts", with(func(c *Capabilities) { c.noColor = true }), true},
		{"TERM=dumb", with(func(c *Capabilities) { c.dumb = true }), false},
		{"no TTY support at all", Capabilities{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.SupportsPrompt())
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
	c := Capabilities{stdoutIsTTY: true, stderrIsTTY: true}
	assert.True(t, c.SupportsStdoutColor())
	assert.True(t, c.SupportsStderrColor())

	c.noColor = true
	assert.False(t, c.SupportsStdoutColor())
	assert.False(t, c.SupportsStderrColor())

	c = Capabilities{stdoutIsTTY: true}
	assert.True(t, c.SupportsStdoutColor())
	assert.False(t, c.SupportsStderrColor())
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
