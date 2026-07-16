package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedEnvActivation(t *testing.T) {
	t.Run("creates bashrc when absent", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv(env.HomeEnvVar(), home)

		require.NoError(t, seedEnvActivation(t.Context()))

		content, err := os.ReadFile(filepath.Join(home, ".bashrc"))
		require.NoError(t, err)
		assert.Contains(t, string(content), envActivationMarker)
		assert.Contains(t, string(content), `export PATH="$(dirname "$DATABRICKS_VIRTUAL_ENV"):$PATH"`)
	})

	t.Run("seeds the runtime guard so the snippet is a no-op when the var is unset", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv(env.HomeEnvVar(), home)

		require.NoError(t, seedEnvActivation(t.Context()))

		content, err := os.ReadFile(filepath.Join(home, ".bashrc"))
		require.NoError(t, err)
		assert.Contains(t, string(content), `if [ -n "$DATABRICKS_VIRTUAL_ENV" ]; then`)
	})

	t.Run("appends after existing content with separating newline", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv(env.HomeEnvVar(), home)
		bashrc := filepath.Join(home, ".bashrc")
		// No trailing newline, to exercise the separator.
		require.NoError(t, os.WriteFile(bashrc, []byte("export FOO=bar"), 0o644))

		require.NoError(t, seedEnvActivation(t.Context()))

		content, err := os.ReadFile(bashrc)
		require.NoError(t, err)
		assert.Contains(t, string(content), "export FOO=bar\n"+envActivationMarker)
	})

	t.Run("does not insert a blank line when content already ends in a newline", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv(env.HomeEnvVar(), home)
		bashrc := filepath.Join(home, ".bashrc")
		require.NoError(t, os.WriteFile(bashrc, []byte("export FOO=bar\n"), 0o644))

		require.NoError(t, seedEnvActivation(t.Context()))

		content, err := os.ReadFile(bashrc)
		require.NoError(t, err)
		assert.Contains(t, string(content), "export FOO=bar\n"+envActivationMarker)
		assert.NotContains(t, string(content), "export FOO=bar\n\n"+envActivationMarker)
	})

	t.Run("idempotent across restarts and preserves existing content", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv(env.HomeEnvVar(), home)
		bashrc := filepath.Join(home, ".bashrc")
		require.NoError(t, os.WriteFile(bashrc, []byte("export FOO=bar\n"), 0o644))

		require.NoError(t, seedEnvActivation(t.Context()))
		after, err := os.ReadFile(bashrc)
		require.NoError(t, err)

		// The second call must find the marker and leave the file byte-for-byte unchanged.
		require.NoError(t, seedEnvActivation(t.Context()))
		again, err := os.ReadFile(bashrc)
		require.NoError(t, err)

		assert.Equal(t, string(after), string(again))
		assert.Equal(t, 1, strings.Count(string(again), envActivationMarker))
		assert.Contains(t, string(again), "export FOO=bar")
	})
}

func TestWriteSessionConfig(t *testing.T) {
	wsHome := t.TempDir()
	require.NoError(t, writeSessionConfig(t.Context(), wsHome))

	// The rcfile carries the PATH fixup and lives under .config, not at the home root.
	rcfilePath := filepath.Join(wsHome, ".config", "bashrc")
	rcfile, err := os.ReadFile(rcfilePath)
	require.NoError(t, err)
	assert.Contains(t, string(rcfile), `export PATH="$(dirname "$DATABRICKS_VIRTUAL_ENV"):$PATH"`)
	assert.NoFileExists(t, filepath.Join(wsHome, ".bashrc"))

	// The OS-home ~/.bashrc is sourced ahead of the PATH fixup, so the fixup wins over
	// anything it prepends.
	assert.Contains(t, string(rcfile), `. "$`+workspace.OSHomeEnvVar+`/.bashrc"`)
	assert.Less(t, strings.Index(string(rcfile), osHomeSourceMarker), strings.Index(string(rcfile), envActivationMarker))

	// The IPython startup script lives under .config/ipython, not the home root's .ipython.
	initScript, err := os.ReadFile(filepath.Join(wsHome, ".config", "ipython", "profile_default", "startup", "init_script.py"))
	require.NoError(t, err)
	assert.Equal(t, jupyterInitScript, string(initScript))
	assert.NoDirExists(t, filepath.Join(wsHome, ".ipython"))

	// The rcfile is the user's general-purpose bashrc: re-seeding on a server restart
	// must preserve their edits and not duplicate the snippet.
	edited := string(rcfile) + "\nexport MY_CUSTOM=1\n"
	require.NoError(t, os.WriteFile(rcfilePath, []byte(edited), 0o644))

	require.NoError(t, writeSessionConfig(t.Context(), wsHome))

	after, err := os.ReadFile(rcfilePath)
	require.NoError(t, err)
	assert.Equal(t, edited, string(after))
	assert.Equal(t, 1, strings.Count(string(after), envActivationMarker))
	assert.Equal(t, 1, strings.Count(string(after), osHomeSourceMarker))
}

func TestSaveJupyterInitScript(t *testing.T) {
	ipythonDir := filepath.Join(t.TempDir(), "ipython")
	require.NoError(t, saveJupyterInitScript(t.Context(), ipythonDir))

	initScript, err := os.ReadFile(filepath.Join(ipythonDir, "profile_default", "startup", "init_script.py"))
	require.NoError(t, err)
	assert.Equal(t, jupyterInitScript, string(initScript))
}
