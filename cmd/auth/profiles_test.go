package auth

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfiles(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	// Create a config file with a profile
	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "profile1",
		Host:       "abc.cloud.databricks.com",
		Token:      "token1",
		AuthType:   "pat",
	})
	require.NoError(t, err)

	// Let the environment think we're using another profile
	t.Setenv("DATABRICKS_HOST", "https://def.cloud.databricks.com")
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	// Load the profile
	profile := &profileMetadata{Name: "profile1"}
	profile.Load(ctx, configFile, true)

	// Check the profile
	assert.Equal(t, "profile1", profile.Name)
	assert.Equal(t, "https://abc.cloud.databricks.com", profile.Host)
	assert.Equal(t, "aws", profile.Cloud)
	assert.Equal(t, "pat", profile.AuthType)
}

func TestProfilesDefaultMarker(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	// Create two profiles.
	for _, name := range []string{"profile-a", "profile-b"} {
		err := databrickscfg.SaveToProfile(ctx, &config.Config{
			ConfigFile: configFile,
			Profile:    name,
			Host:       "https://" + name + ".cloud.databricks.com",
			Token:      "token",
		})
		require.NoError(t, err)
	}

	// Set profile-a as the default.
	err := databrickscfg.SetDefaultProfile(ctx, "profile-a", configFile)
	require.NoError(t, err)

	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	// Read back the default profile and verify.
	defaultProfile, err := databrickscfg.GetDefaultProfile(ctx, configFile)
	require.NoError(t, err)
	assert.Equal(t, "profile-a", defaultProfile)
}
