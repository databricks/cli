package agents

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubLookPath replaces the package lookPath with one that resolves only the
// named binaries, restoring the original when the test finishes.
func stubLookPath(t *testing.T, onPath ...string) {
	t.Helper()
	set := make(map[string]bool, len(onPath))
	for _, name := range onPath {
		set[name] = true
	}
	orig := lookPath
	lookPath = func(name string) (string, error) {
		if set[name] {
			return filepath.Join("/usr/bin", name), nil
		}
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() { lookPath = orig })
}

// configDir returns a closure usable as Agent.ConfigDir. When create is true a
// directory is materialized so Detected reports it; otherwise it points at a
// path that does not exist.
func configDir(t *testing.T, create bool) func(context.Context) (string, error) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "config")
	if create {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	return func(_ context.Context) (string, error) { return dir, nil }
}

func TestHasBinary(t *testing.T) {
	ctx := t.Context()

	t.Run("empty binary is never on path", func(t *testing.T) {
		stubLookPath(t)
		a := &Agent{Binary: ""}
		assert.False(t, a.HasBinary(ctx))
	})

	t.Run("resolved binary is on path", func(t *testing.T) {
		stubLookPath(t, "claude")
		a := &Agent{Binary: "claude"}
		assert.True(t, a.HasBinary(ctx))
	})

	t.Run("missing binary is not on path", func(t *testing.T) {
		stubLookPath(t)
		a := &Agent{Binary: "claude"}
		assert.False(t, a.HasBinary(ctx))
	})
}

func TestDisplayState(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name     string
		binary   string
		onPath   []string
		plugin   *PluginSpec
		hasCfg   bool
		expected DisplayState
	}{
		{"plugin agent, binary on path", "claude", []string{"claude"}, &PluginSpec{}, false, StateAvailable},
		{"plugin agent, binary on path, config too", "claude", []string{"claude"}, &PluginSpec{}, true, StateAvailable},
		{"plugin agent, no binary, config exists", "claude", nil, &PluginSpec{}, true, StateInstalledCLIMissing},
		{"plugin agent, no binary, no config", "claude", nil, &PluginSpec{}, false, StateNotFound},
		{"manual-only agent always manual", "cursor-agent", nil, &PluginSpec{ManualOnly: true}, false, StateManualOnly},
		{"manual-only agent ignores presence", "cursor-agent", []string{"cursor-agent"}, &PluginSpec{ManualOnly: true}, true, StateManualOnly},
		{"files-only agent, binary on path", "opencode", []string{"opencode"}, nil, false, StateFilesOnly},
		{"files-only agent, config only", "opencode", nil, nil, true, StateFilesOnly},
		{"files-only agent, neither", "opencode", nil, nil, false, StateNotFound},
		{"files-only agent, no binary name, config", "", nil, nil, true, StateFilesOnly},
		{"files-only agent, no binary name, no config", "", nil, nil, false, StateNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stubLookPath(t, tc.onPath...)
			a := &Agent{Binary: tc.binary, Plugin: tc.plugin, ConfigDir: configDir(t, tc.hasCfg)}
			assert.Equal(t, tc.expected, a.DisplayState(ctx))
		})
	}
}

func TestPreselect(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name     string
		binary   string
		onPath   []string
		plugin   *PluginSpec
		hasCfg   bool
		expected bool
	}{
		{"available is pre-checked", "claude", []string{"claude"}, &PluginSpec{}, false, true},
		{"files-only with config is pre-checked", "opencode", nil, nil, true, true},
		{"files-only without config is not", "opencode", []string{"opencode"}, nil, false, false},
		{"manual-only is not pre-checked", "cursor-agent", []string{"cursor-agent"}, &PluginSpec{ManualOnly: true}, true, false},
		{"installed-cli-missing is not pre-checked", "claude", nil, &PluginSpec{}, true, false},
		{"not-found is not pre-checked", "claude", nil, &PluginSpec{}, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stubLookPath(t, tc.onPath...)
			a := &Agent{Binary: tc.binary, Plugin: tc.plugin, ConfigDir: configDir(t, tc.hasCfg)}
			assert.Equal(t, tc.expected, a.Preselect(ctx))
		})
	}
}

func TestProbePluginUnsupportedWithoutCapability(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name  string
		agent *Agent
	}{
		{"no binary", &Agent{Binary: "", Plugin: &PluginSpec{}}},
		{"no plugin", &Agent{Binary: "antigravity"}},
		{"manual only", &Agent{Binary: "cursor-agent", Plugin: &PluginSpec{ManualOnly: true}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.agent.ProbePlugin(ctx)
			assert.ErrorIs(t, err, ErrPluginUnsupported)
		})
	}
}

func TestProbePluginSupported(t *testing.T) {
	stubLookPath(t, "claude")
	ctx, stub := process.WithStub(t.Context())
	stub.WithCallback(func(*exec.Cmd) error { return nil })

	a := &Agent{Binary: "claude", Plugin: &PluginSpec{}}
	require.NoError(t, a.ProbePlugin(ctx))
	assert.Equal(t, []string{"claude plugin --help"}, stub.Commands())
}

func TestProbePluginCLIReportsUnsupported(t *testing.T) {
	stubLookPath(t, "codex")
	ctx, stub := process.WithStub(t.Context())
	stub.WithFailure(errors.New("unknown command: plugin"))

	a := &Agent{Binary: "codex", Plugin: &PluginSpec{}}
	err := a.ProbePlugin(ctx)
	assert.ErrorIs(t, err, ErrPluginUnsupported)
	assert.Equal(t, 1, stub.Len())
}

func TestProbePluginBinaryNotOnPath(t *testing.T) {
	stubLookPath(t)
	ctx, stub := process.WithStub(t.Context())

	a := &Agent{Binary: "claude", Plugin: &PluginSpec{}}
	err := a.ProbePlugin(ctx)
	assert.ErrorIs(t, err, exec.ErrNotFound)
	assert.Equal(t, 0, stub.Len(), "binary not on PATH must not be executed")
}

func TestProbePluginRefusesDotRelativeBinary(t *testing.T) {
	// LookPath returns ErrDot (with a path) when the binary resolves only
	// relative to cwd. ProbePlugin must refuse to run it.
	orig := lookPath
	lookPath = func(name string) (string, error) { return filepath.Join(".", name), exec.ErrDot }
	t.Cleanup(func() { lookPath = orig })

	ctx, stub := process.WithStub(t.Context())
	a := &Agent{Binary: "claude", Plugin: &PluginSpec{}}
	err := a.ProbePlugin(ctx)
	assert.ErrorIs(t, err, exec.ErrDot)
	assert.Equal(t, 0, stub.Len(), "a cwd-relative binary must never be executed")
}

func TestOpenCodeConfigDir(t *testing.T) {
	ctx := t.Context()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if runtime.GOOS == "windows" {
		appData := t.TempDir()
		t.Setenv("APPDATA", appData)
		dir, err := openCodeConfigDir(ctx)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(appData, "opencode"), dir)
		return
	}

	t.Run("honors XDG_CONFIG_HOME", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		dir, err := openCodeConfigDir(ctx)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(xdg, "opencode"), dir)
	})

	t.Run("defaults to ~/.config when XDG is unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir, err := openCodeConfigDir(ctx)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".config", "opencode"), dir)
	})
}
