package databrickscfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileCloud(t *testing.T) {
	assert.Equal(t, Profile{Host: "https://dbc-XXXXXXXX-YYYY.cloud.databricks.com"}.Cloud(), "AWS")
	assert.Equal(t, Profile{Host: "https://adb-xxx.y.azuredatabricks.net/"}.Cloud(), "Azure")
	assert.Equal(t, Profile{Host: "https://workspace.gcp.databricks.com/"}.Cloud(), "GCP")
	assert.Equal(t, Profile{Host: "https://some.invalid.host.com/"}.Cloud(), "AWS")
}

func TestLoadProfilesReturnsHomedirAsTilde(t *testing.T) {
	t.Setenv("HOME", "./testdata")
	file, _, err := LoadProfiles("./testdata/databrickscfg", func(p Profile) bool { return true })
	require.NoError(t, err)
	assert.Equal(t, "~/databrickscfg", file)
}

func TestLoadProfilesMatchWorkspace(t *testing.T) {
	_, profiles, err := LoadProfiles("./testdata/databrickscfg", MatchWorkspaceProfiles)
	require.NoError(t, err)
	assert.Equal(t, []string{"DEFAULT", "query", "foo1", "foo2"}, profiles.Names())
}

func TestLoadProfilesMatchAccount(t *testing.T) {
	_, profiles, err := LoadProfiles("./testdata/databrickscfg", MatchAccountProfiles)
	require.NoError(t, err)
	assert.Equal(t, []string{"acc"}, profiles.Names())
}
