package cmdio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// envKeysToIsolate lists every environment variable read by DetectHost. Tests
// clear all of them at the start so process env from the developer's shell
// (e.g. TERM_PROGRAM=iTerm.app when running locally) cannot leak in.
var envKeysToIsolate = []string{
	envTermProgram,
	envTerminalEmulator,
	envCursorTraceID,
	envCFBundleID,
	envCopilotAgent,
	envCopilotIntegrator,
}

func isolateHostEnv(t *testing.T, overrides map[string]string) {
	for _, k := range envKeysToIsolate {
		t.Setenv(k, "")
	}
	for k, v := range overrides {
		t.Setenv(k, v)
	}
}

func TestDetectHost(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
		want Host
	}{
		{
			name: "no env vars",
			envs: nil,
			want: HostUnknown,
		},
		{
			name: "vscode",
			envs: map[string]string{"TERM_PROGRAM": "vscode"},
			want: HostVSCode,
		},
		{
			name: "vscode with copilot agent",
			envs: map[string]string{
				"TERM_PROGRAM":                 "vscode",
				"GITHUB_COPILOT_AGENT_VERSION": "1.2.3",
			},
			want: HostVSCodeCopilot,
		},
		{
			name: "cursor wins over vscode TERM_PROGRAM",
			envs: map[string]string{
				"TERM_PROGRAM":    "vscode",
				"CURSOR_TRACE_ID": "abc123",
			},
			want: HostCursor,
		},
		{
			name: "windsurf wins over vscode TERM_PROGRAM",
			envs: map[string]string{
				"TERM_PROGRAM":         "vscode",
				"__CFBundleIdentifier": "com.exafunction.windsurf",
			},
			want: HostWindsurf,
		},
		{
			name: "jetbrains",
			envs: map[string]string{"TERMINAL_EMULATOR": "JetBrains-JediTerm"},
			want: HostJetBrains,
		},
		{
			name: "apple terminal",
			envs: map[string]string{"TERM_PROGRAM": "Apple_Terminal"},
			want: HostAppleTerminal,
		},
		{
			name: "iterm",
			envs: map[string]string{"TERM_PROGRAM": "iTerm.app"},
			want: HostITerm,
		},
		{
			name: "warp",
			envs: map[string]string{"TERM_PROGRAM": "WarpTerminal"},
			want: HostWarp,
		},
		{
			name: "ghostty",
			envs: map[string]string{"TERM_PROGRAM": "ghostty"},
			want: HostGhostty,
		},
		{
			name: "wezterm",
			envs: map[string]string{"TERM_PROGRAM": "WezTerm"},
			want: HostWezTerm,
		},
		{
			name: "zed",
			envs: map[string]string{"TERM_PROGRAM": "zed"},
			want: HostZed,
		},
		{
			name: "hyper",
			envs: map[string]string{"TERM_PROGRAM": "Hyper"},
			want: HostHyper,
		},
		{
			name: "tabby",
			envs: map[string]string{"TERM_PROGRAM": "Tabby"},
			want: HostTabby,
		},
		{
			name: "unknown TERM_PROGRAM falls through to unknown",
			envs: map[string]string{"TERM_PROGRAM": "somethingnew"},
			want: HostUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateHostEnv(t, tt.envs)
			assert.Equal(t, tt.want, DetectHost(t.Context()))
		})
	}
}
