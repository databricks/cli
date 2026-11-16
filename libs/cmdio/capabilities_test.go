package cmdio

import (
	"io"
	"testing"

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
			},
			expected: true,
		},
		{
			name: "MSYS2 pipe stdin blocks prompts",
			caps: Capabilities{
				stdinIsTTY:      true,
				stdoutIsTTY:     true,
				stderrIsTTY:     true,
				color:           true,
				stdinIsMSYSPipe: true,
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
			},
			expected: false,
		},
		{
			name: "stdout not TTY (prompts write to stderr, so this is OK)",
			caps: Capabilities{
				stdinIsTTY:  true,
				stdoutIsTTY: false,
				stderrIsTTY: true,
				color:       true,
			},
			expected: true,
		},
		{
			name: "stderr not TTY",
			caps: Capabilities{
				stdinIsTTY:  true,
				stdoutIsTTY: true,
				stderrIsTTY: false,
				color:       true,
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

func TestCapabilities_InteractiveMode(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		expected InteractiveMode
	}{
		{
			name: "full interactive - all TTYs, color, console stdin",
			caps: Capabilities{
				stdinIsTTY:  true,
				stderrIsTTY: true,
				color:       true,
			},
			expected: InteractiveModeFull,
		},
		{
			name: "output only - stdin not TTY",
			caps: Capabilities{
				stdinIsTTY:  false,
				stderrIsTTY: true,
				color:       true,
			},
			expected: InteractiveModeOutputOnly,
		},
		{
			name: "output only - MSYS2 pipe stdin",
			caps: Capabilities{
				stdinIsTTY:      true,
				stderrIsTTY:     true,
				color:           true,
				stdinIsMSYSPipe: true,
			},
			expected: InteractiveModeOutputOnly,
		},
		{
			name: "none - stderr not TTY",
			caps: Capabilities{
				stdinIsTTY:  true,
				stderrIsTTY: false,
				color:       true,
			},
			expected: InteractiveModeNone,
		},
		{
			name: "none - NO_COLOR set",
			caps: Capabilities{
				stdinIsTTY:  true,
				stderrIsTTY: true,
				color:       false,
			},
			expected: InteractiveModeNone,
		},
		{
			name: "none - no TTY support at all",
			caps: Capabilities{
				stdinIsTTY:  false,
				stderrIsTTY: false,
				color:       false,
			},
			expected: InteractiveModeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.caps.InteractiveMode())
		})
	}
}
