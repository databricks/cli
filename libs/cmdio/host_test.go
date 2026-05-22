package cmdio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// envKeysToIsolate lists every environment variable read by DetectHost.
// Tests clear all of them at the start so process env from the developer's
// shell (e.g. TERM_PROGRAM=iTerm.app on a macOS dev machine) cannot leak in.
var envKeysToIsolate = []string{
	envTermProgram,
	envTerminalEmulator,
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
			name: "vscode and forks all classify as vscode",
			envs: map[string]string{"TERM_PROGRAM": "vscode"},
			want: HostVSCode,
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
			name: "wezterm",
			envs: map[string]string{"TERM_PROGRAM": "WezTerm"},
			want: HostWezTerm,
		},
		{
			name: "ghostty",
			envs: map[string]string{"TERM_PROGRAM": "ghostty"},
			want: HostGhostty,
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
