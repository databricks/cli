package config

import (
	"context"
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWorkspaceTest(t *testing.T) string {
	testutil.CleanupEnvironment(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	return home
}

func TestWorkspaceResolveProfileFromHost(t *testing.T) {
	// If only a workspace host is specified, try to find a profile that uses
	// the same workspace host (unambiguously).
	w := Workspace{
		Host: "https://abc.cloud.databricks.com",
	}

	t.Run("no config file", func(t *testing.T) {
		setupWorkspaceTest(t)
		_, err := w.Client()
		assert.NoError(t, err)
	})

	t.Run("default config file", func(t *testing.T) {
		setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(context.Background(), &config.Config{
			Profile: "default",
			Host:    "https://abc.cloud.databricks.com",
			Token:   "123",
		})
		require.NoError(t, err)

		client, err := w.Client()
		assert.NoError(t, err)
		assert.Equal(t, "default", client.Config.Profile)
	})

	t.Run("custom config file", func(t *testing.T) {
		home := setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(context.Background(), &config.Config{
			ConfigFile: filepath.Join(home, "customcfg"),
			Profile:    "custom",
			Host:       "https://abc.cloud.databricks.com",
			Token:      "123",
		})
		require.NoError(t, err)

		t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(home, "customcfg"))
		client, err := w.Client()
		assert.NoError(t, err)
		assert.Equal(t, "custom", client.Config.Profile)
	})
}

func TestWorkspaceVerifyProfileForHost(t *testing.T) {
	// If both a workspace host and a profile are specified,
	// verify that the host configured in the profile matches
	// the host configured in the bundle configuration.
	w := Workspace{
		Host:    "https://abc.cloud.databricks.com",
		Profile: "abc",
	}

	t.Run("no config file", func(t *testing.T) {
		setupWorkspaceTest(t)
		_, err := w.Client()
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("default config file with match", func(t *testing.T) {
		setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(context.Background(), &config.Config{
			Profile: "abc",
			Host:    "https://abc.cloud.databricks.com",
		})
		require.NoError(t, err)

		_, err = w.Client()
		assert.NoError(t, err)
	})

	t.Run("default config file with mismatch", func(t *testing.T) {
		setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(context.Background(), &config.Config{
			Profile: "abc",
			Host:    "https://def.cloud.databricks.com",
		})
		require.NoError(t, err)

		_, err = w.Client()
		assert.ErrorContains(t, err, "doesn’t match the host configured in the bundle")
	})

	t.Run("custom config file with match", func(t *testing.T) {
		home := setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(context.Background(), &config.Config{
			ConfigFile: filepath.Join(home, "customcfg"),
			Profile:    "abc",
			Host:       "https://abc.cloud.databricks.com",
		})
		require.NoError(t, err)

		t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(home, "customcfg"))
		_, err = w.Client()
		assert.NoError(t, err)
	})

	t.Run("custom config file with mismatch", func(t *testing.T) {
		home := setupWorkspaceTest(t)

		// This works if there is a config file with a matching profile.
		err := databrickscfg.SaveToProfile(context.Background(), &config.Config{
			ConfigFile: filepath.Join(home, "customcfg"),
			Profile:    "abc",
			Host:       "https://def.cloud.databricks.com",
		})
		require.NoError(t, err)

		t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(home, "customcfg"))
		_, err = w.Client()
		assert.ErrorContains(t, err, "doesn’t match the host configured in the bundle")
	})
}
