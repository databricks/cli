package localenv

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUvArgs(t *testing.T) {
	m := &uvManager{bin: "uv"}
	assert.Equal(t, []string{"sync"}, m.syncArgs())
	assert.Equal(t, []string{"python", "install", "3.12"}, m.pythonInstallArgs("3.12"))
	assert.Equal(t, []string{"pip", "install", "pip", "--python", "/p/.venv/bin/python"}, m.pipSeedArgs("/p/.venv/bin/python"))
}

func TestDiscoverUvFindsBinOnPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "uv")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)
	got, err := discoverUv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, bin, got)
}

func TestDiscoverUvFindsBinInLocalBin(t *testing.T) {
	// The installer drops uv (uv.exe on Windows) under ~/.local/bin; discoverUv
	// must find it there when it is not on PATH, using the OS-appropriate name.
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	exe := "uv"
	if runtime.GOOS == "windows" {
		exe = "uv.exe"
	}
	bin := filepath.Join(binDir, exe)
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))

	t.Setenv("PATH", t.TempDir()) // no uv on PATH
	ctx := env.WithUserHomeDir(t.Context(), home)
	ctx = env.Set(ctx, "XDG_BIN_HOME", "")
	got, err := discoverUv(ctx)
	require.NoError(t, err)
	assert.Equal(t, bin, got)
}

func TestDiscoverUvSkipsRelativeCandidatesWhenHomeUnset(t *testing.T) {
	// Regression: when HOME and XDG_BIN_HOME are unset the candidate paths
	// collapse to relative "uv" / ".local/bin/uv". discoverUv must not os.Stat
	// them against the CWD and return a stray ./uv as if it were the real binary.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "uv"), []byte("#!/bin/sh\n"), 0o755))
	t.Chdir(dir)
	t.Setenv("PATH", t.TempDir()) // a PATH with no uv, so LookPath falls through

	ctx := env.WithUserHomeDir(t.Context(), "")
	ctx = env.Set(ctx, "XDG_BIN_HOME", "")
	got, err := discoverUv(ctx)
	if err == nil {
		assert.True(t, filepath.IsAbs(got), "discoverUv must not return a relative path; got %q", got)
	}
}

func TestPipConfIndexURLReadsOSSpecificPath(t *testing.T) {
	// The primary pip config path for the current OS must be honored, not just
	// the Linux/XDG ~/.config/pip/pip.conf location.
	tmp := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), tmp)
	if runtime.GOOS == "windows" {
		ctx = env.Set(ctx, "APPDATA", filepath.Join(tmp, "AppData", "Roaming"))
	}

	paths := pipConfPaths(ctx)
	require.NotEmpty(t, paths)
	primary := paths[0]
	require.NoError(t, os.MkdirAll(filepath.Dir(primary), 0o755))
	require.NoError(t, os.WriteFile(primary, []byte("[global]\nindex-url = https://proxy.example/simple\n"), 0o644))

	assert.Equal(t, "https://proxy.example/simple", pipConfIndexURL(ctx))
}

func TestRedactURLCredentials(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://user:pass@proxy.example/simple", "https://proxy.example/simple"},
		{"https://token@proxy.example/simple", "https://proxy.example/simple"},
		{"https://proxy.example/simple", "https://proxy.example/simple"},
		{"not a url", "not a url"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, redactURLCredentials(tc.in))
	}
}

func TestPipConfIndexURLReadsLegacyPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("legacy ~/.pip path is Unix/macOS only")
	}
	// The legacy ~/.pip/pip.conf location pip still reads must be honored when the
	// newer per-user locations are absent.
	tmp := t.TempDir()
	legacy := filepath.Join(tmp, ".pip")
	require.NoError(t, os.MkdirAll(legacy, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "pip.conf"), []byte("[global]\nindex-url = https://legacy.example/simple\n"), 0o644))

	ctx := env.WithUserHomeDir(t.Context(), tmp)
	assert.Equal(t, "https://legacy.example/simple", pipConfIndexURL(ctx))
}

func TestPipConfIndexURL(t *testing.T) {
	t.Run("returns_url_from_pip_conf", func(t *testing.T) {
		tmp := t.TempDir()
		confDir := filepath.Join(tmp, ".config", "pip")
		require.NoError(t, os.MkdirAll(confDir, 0o755))
		confContent := "[global]\nindex-url = https://proxy.example/simple\n"
		require.NoError(t, os.WriteFile(filepath.Join(confDir, "pip.conf"), []byte(confContent), 0o644))

		ctx := env.WithUserHomeDir(t.Context(), tmp)
		got := pipConfIndexURL(ctx)
		assert.Equal(t, "https://proxy.example/simple", got)
	})

	t.Run("returns_empty_when_no_pip_conf", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := env.WithUserHomeDir(t.Context(), tmp)
		got := pipConfIndexURL(ctx)
		assert.Empty(t, got)
	})

	t.Run("returns_empty_when_no_index_url_in_conf", func(t *testing.T) {
		tmp := t.TempDir()
		confDir := filepath.Join(tmp, ".config", "pip")
		require.NoError(t, os.MkdirAll(confDir, 0o755))
		confContent := "[global]\nextra-index-url = https://other.example/simple\n"
		require.NoError(t, os.WriteFile(filepath.Join(confDir, "pip.conf"), []byte(confContent), 0o644))

		ctx := env.WithUserHomeDir(t.Context(), tmp)
		got := pipConfIndexURL(ctx)
		assert.Empty(t, got)
	})
}

func TestResolveIndexURLRespectsExistingEnv(t *testing.T) {
	m := &uvManager{}

	t.Run("returns_empty_when_UV_INDEX_URL_already_set", func(t *testing.T) {
		// When UV_INDEX_URL is in ctx, resolveIndexURL must not override it.
		ctx := env.Set(t.Context(), "UV_INDEX_URL", "https://explicit.example/simple")

		// Set up a pip.conf that would otherwise be used.
		tmp := t.TempDir()
		confDir := filepath.Join(tmp, ".config", "pip")
		require.NoError(t, os.MkdirAll(confDir, 0o755))
		confContent := "[global]\nindex-url = https://proxy.example/simple\n"
		require.NoError(t, os.WriteFile(filepath.Join(confDir, "pip.conf"), []byte(confContent), 0o644))
		ctx = env.WithUserHomeDir(ctx, tmp)

		got := m.resolveIndexURL(ctx)
		assert.Empty(t, got)
	})

	t.Run("returns_pip_conf_url_when_UV_INDEX_URL_unset", func(t *testing.T) {
		tmp := t.TempDir()
		confDir := filepath.Join(tmp, ".config", "pip")
		require.NoError(t, os.MkdirAll(confDir, 0o755))
		confContent := "[global]\nindex-url = https://proxy.example/simple\n"
		require.NoError(t, os.WriteFile(filepath.Join(confDir, "pip.conf"), []byte(confContent), 0o644))

		ctx := env.WithUserHomeDir(t.Context(), tmp)
		got := m.resolveIndexURL(ctx)
		assert.Equal(t, "https://proxy.example/simple", got)
	})
}

func TestUvFailureIncludesStderr(t *testing.T) {
	t.Run("includes_stderr_when_present", func(t *testing.T) {
		underlying := &process.ProcessError{
			Command: "uv sync",
			Err:     errors.New("exit status 2"),
			Stderr:  "error: Connection refused\n",
		}
		pe := uvFailure(ErrProvision, underlying, "uv sync")
		assert.Equal(t, ErrProvision, pe.Code)
		assert.Contains(t, pe.Msg, "Connection refused")
		assert.NotEqual(t, '\n', pe.Msg[len(pe.Msg)-1], "Msg must not end with a newline")
	})

	t.Run("omits_stderr_suffix_when_empty", func(t *testing.T) {
		underlying := &process.ProcessError{
			Command: "uv sync",
			Err:     errors.New("exit status 2"),
			Stderr:  "",
		}
		pe := uvFailure(ErrProvision, underlying, "uv sync")
		assert.Equal(t, ErrProvision, pe.Code)
		assert.Equal(t, "uv sync failed", pe.Msg)
	})

	t.Run("non_process_error_uses_action_only", func(t *testing.T) {
		pe := uvFailure(ErrProvision, errors.New("some other error"), "uv sync")
		assert.Equal(t, ErrProvision, pe.Code)
		assert.Equal(t, "uv sync failed", pe.Msg)
	})
}
