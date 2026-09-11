package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
