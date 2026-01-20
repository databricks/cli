package auth

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfiles(t *testing.T) {
	ctx := context.Background()
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

func TestProfilesWithScopes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	// Create a config file with a profile that has scopes
	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "scoped-profile",
		Host:       "abc.cloud.databricks.com",
		AuthType:   "databricks-cli",
		Scopes:     []string{"jobs", "pipelines", "clusters"},
	})
	require.NoError(t, err)

	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	profile := &profileMetadata{Name: "scoped-profile"}
	profile.Load(ctx, configFile, true)

	assert.Equal(t, "scoped-profile", profile.Name)
	assert.Equal(t, "https://abc.cloud.databricks.com", profile.Host)
	assert.Equal(t, "databricks-cli", profile.AuthType)
	// Scopes are loaded from the resolved config via cfg.GetScopes()
	assert.Equal(t, "clusters,jobs,pipelines", profile.Scopes)
}

func TestProfilesWithDefaultScopes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	// Create a config file with a profile that has no scopes
	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "default-scopes",
		Host:       "abc.cloud.databricks.com",
		AuthType:   "databricks-cli",
	})
	require.NoError(t, err)

	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	profile := &profileMetadata{Name: "default-scopes"}
	profile.Load(ctx, configFile, true)

	assert.Equal(t, "default-scopes", profile.Name)
	// cfg.GetScopes() returns "all-apis" when no scopes are set
	assert.Equal(t, "all-apis", profile.Scopes)
}

func TestProfilesValidationSkippedForRestrictedScopes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "restricted",
		Host:       "abc.cloud.databricks.com",
		AuthType:   "databricks-cli",
		Scopes:     []string{"jobs", "pipelines"},
	})
	require.NoError(t, err)

	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}

	profile := &profileMetadata{Name: "restricted"}
	// skipValidate=false but validation should still be skipped due to restricted scopes
	profile.Load(ctx, configFile, false)

	assert.Equal(t, "restricted", profile.Name)
	assert.Equal(t, "jobs,pipelines", profile.Scopes)
	assert.True(t, profile.ValidationSkipped)
	assert.False(t, profile.Valid)
}
