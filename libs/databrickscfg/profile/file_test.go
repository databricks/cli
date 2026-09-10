package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileCloud(t *testing.T) {
	assert.Equal(t, "AWS", Profile{Host: "https://dbc-XXXXXXXX-YYYY.cloud.databricks.com"}.Cloud())
	assert.Equal(t, "Azure", Profile{Host: "https://adb-xxx.y.azuredatabricks.net/"}.Cloud())
	assert.Equal(t, "GCP", Profile{Host: "https://workspace.gcp.databricks.com/"}.Cloud())
	assert.Equal(t, "AWS", Profile{Host: "https://some.invalid.host.com/"}.Cloud())
}

func TestProfilesSearchCaseInsensitive(t *testing.T) {
	profiles := Profiles{
		Profile{Name: "foo", Host: "bar"},
	}
	assert.True(t, profiles.SearchCaseInsensitive("f", 0))
	assert.True(t, profiles.SearchCaseInsensitive("OO", 0))
	assert.True(t, profiles.SearchCaseInsensitive("b", 0))
	assert.True(t, profiles.SearchCaseInsensitive("AR", 0))
	assert.False(t, profiles.SearchCaseInsensitive("qu", 0))
}

func TestLoadProfilesReturnsHomedirAsTilde(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "testdata")
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./testdata/databrickscfg")
	profiler := FileProfilerImpl{}
	file, err := profiler.GetPath(ctx)
	require.NoError(t, err)
	require.Equal(t, filepath.Clean("~/databrickscfg"), file)
}

func TestLoadProfilesReturnsHomedirAsTildeExoticFile(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "testdata")
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "~/databrickscfg")
	profiler := FileProfilerImpl{}
	file, err := profiler.GetPath(ctx)
	require.NoError(t, err)
	require.Equal(t, filepath.Clean("~/databrickscfg"), file)
}

func TestLoadProfilesReturnsHomedirAsTildeDefaultFile(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "testdata/sample-home")
	profiler := FileProfilerImpl{}
	file, err := profiler.GetPath(ctx)
	require.NoError(t, err)
	require.Equal(t, filepath.Clean("~/.databrickscfg"), file)
}

func TestLoadProfilesNoConfiguration(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "testdata")
	profiler := FileProfilerImpl{}
	_, err := profiler.LoadProfiles(ctx, MatchAllProfiles)
	require.ErrorIs(t, err, ErrNoConfiguration)
}

func TestLoadProfilesClientID(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".databrickscfg")
	err := os.WriteFile(configPath, []byte(`[u2m]
host = https://workspace.test
auth_type = databricks-cli
client_id = custom-client-id
`), 0o600)
	require.NoError(t, err)

	ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_FILE", configPath)
	profiles, err := (FileProfilerImpl{}).LoadProfiles(ctx, MatchAllProfiles)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "custom-client-id", profiles[0].ClientID)
	assert.False(t, profiles[0].HasClientCredentials)
}

func TestLoadProfilesMatchWorkspace(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./testdata/databrickscfg")
	profiler := FileProfilerImpl{}
	profiles, err := profiler.LoadProfiles(ctx, MatchWorkspaceProfiles)
	require.NoError(t, err)
	assert.Equal(t, []string{"DEFAULT", "query", "foo1", "foo2", "spog-ws1", "spog-ws2", "spog-dup1", "spog-dup2"}, profiles.Names())
}

func TestLoadProfilesMatchAccount(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./testdata/databrickscfg")
	profiler := FileProfilerImpl{}
	profiles, err := profiler.LoadProfiles(ctx, MatchAccountProfiles)
	require.NoError(t, err)
	assert.Equal(t, []string{"acc"}, profiles.Names())
}
