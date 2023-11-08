package databrickscfg

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileCloud(t *testing.T) {
	assert.Equal(t, Profile{Host: "https://dbc-XXXXXXXX-YYYY.cloud.databricks.com"}.Cloud(), "AWS")
	assert.Equal(t, Profile{Host: "https://adb-xxx.y.azuredatabricks.net/"}.Cloud(), "Azure")
	assert.Equal(t, Profile{Host: "https://workspace.gcp.databricks.com/"}.Cloud(), "GCP")
	assert.Equal(t, Profile{Host: "https://some.invalid.host.com/"}.Cloud(), "AWS")
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
	ctx := context.Background()
	ctx = env.WithUserHomeDir(ctx, "testdata")
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./testdata/databrickscfg")
	file, _, err := LoadProfiles(ctx, func(p Profile) bool { return true })
	require.NoError(t, err)
	require.Equal(t, filepath.Clean("~/databrickscfg"), file)
}

func TestLoadProfilesReturnsHomedirAsTildeExoticFile(t *testing.T) {
	ctx := context.Background()
	ctx = env.WithUserHomeDir(ctx, "testdata")
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "~/databrickscfg")
	file, _, err := LoadProfiles(ctx, func(p Profile) bool { return true })
	require.NoError(t, err)
	require.Equal(t, filepath.Clean("~/databrickscfg"), file)
}

func TestLoadProfilesReturnsHomedirAsTildeDefaultFile(t *testing.T) {
	ctx := context.Background()
	ctx = env.WithUserHomeDir(ctx, "testdata/sample-home")
	file, _, err := LoadProfiles(ctx, func(p Profile) bool { return true })
	require.NoError(t, err)
	require.Equal(t, filepath.Clean("~/.databrickscfg"), file)
}

func TestLoadProfilesNoConfiguration(t *testing.T) {
	ctx := context.Background()
	ctx = env.WithUserHomeDir(ctx, "testdata")
	_, _, err := LoadProfiles(ctx, func(p Profile) bool { return true })
	require.ErrorIs(t, err, ErrNoConfiguration)
}

func TestLoadProfilesMatchWorkspace(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./testdata/databrickscfg")
	_, profiles, err := LoadProfiles(ctx, MatchWorkspaceProfiles)
	require.NoError(t, err)
	assert.Equal(t, []string{"DEFAULT", "query", "foo1", "foo2"}, profiles.Names())
}

func TestLoadProfilesMatchAccount(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./testdata/databrickscfg")
	_, profiles, err := LoadProfiles(ctx, MatchAccountProfiles)
	require.NoError(t, err)
	assert.Equal(t, []string{"acc"}, profiles.Names())
}
