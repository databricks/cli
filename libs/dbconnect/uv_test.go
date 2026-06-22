package dbconnect

import (
	"errors"
	"os"
	"path/filepath"
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
		pe := uvFailure(ErrProvisionFailed, underlying, "uv sync")
		assert.Equal(t, ErrProvisionFailed, pe.Code)
		assert.Contains(t, pe.Msg, "Connection refused")
		assert.NotEqual(t, '\n', pe.Msg[len(pe.Msg)-1], "Msg must not end with a newline")
	})

	t.Run("omits_stderr_suffix_when_empty", func(t *testing.T) {
		underlying := &process.ProcessError{
			Command: "uv sync",
			Err:     errors.New("exit status 2"),
			Stderr:  "",
		}
		pe := uvFailure(ErrProvisionFailed, underlying, "uv sync")
		assert.Equal(t, ErrProvisionFailed, pe.Code)
		assert.Equal(t, "uv sync failed", pe.Msg)
	})

	t.Run("non_process_error_uses_action_only", func(t *testing.T) {
		pe := uvFailure(ErrProvisionFailed, errors.New("some other error"), "uv sync")
		assert.Equal(t, ErrProvisionFailed, pe.Code)
		assert.Equal(t, "uv sync failed", pe.Msg)
	})
}
