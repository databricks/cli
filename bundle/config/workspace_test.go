package config

import (
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
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
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
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
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

func TestWorkspaceNormalizeHostURL(t *testing.T) {
	t.Run("extracts workspace_id from query param", func(t *testing.T) {
		w := Workspace{
			Host: "https://spog.databricks.com/?o=12345",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://spog.databricks.com/", w.Host)
		assert.Equal(t, "12345", w.WorkspaceID)
	})

	t.Run("explicit workspace_id takes precedence", func(t *testing.T) {
		w := Workspace{
			Host:        "https://spog.databricks.com/?o=999",
			WorkspaceID: "explicit",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://spog.databricks.com/", w.Host)
		assert.Equal(t, "explicit", w.WorkspaceID)
	})

	t.Run("no-op for host without query params", func(t *testing.T) {
		w := Workspace{
			Host: "https://normal.databricks.com",
		}
		w.NormalizeHostURL()
		assert.Equal(t, "https://normal.databricks.com", w.Host)
		assert.Empty(t, w.WorkspaceID)
	})
}

func TestWorkspaceClientNormalizesHostBeforeProfileResolution(t *testing.T) {
	// Regression test: Client() must normalize the host URL (strip ?o= and
	// populate WorkspaceID) before building the SDK config and resolving
	// profiles. This ensures workspace_id is available for disambiguation.
	setupWorkspaceTest(t)

	err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile:     "ws1",
		Host:        "https://spog.databricks.com",
		Token:       "token1",
		WorkspaceID: "111",
	})
	require.NoError(t, err)

	err = databrickscfg.SaveToProfile(t.Context(), &config.Config{
		Profile:     "ws2",
		Host:        "https://spog.databricks.com",
		Token:       "token2",
		WorkspaceID: "222",
	})
	require.NoError(t, err)

	// Host with ?o= should be normalized and workspace_id used to disambiguate.
	w := Workspace{
		Host: "https://spog.databricks.com/?o=222",
	}
	client, err := w.Client()
	require.NoError(t, err)
	assert.Equal(t, "ws2", client.Config.Profile)
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
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
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
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
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
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
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
		err := databrickscfg.SaveToProfile(t.Context(), &config.Config{
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
