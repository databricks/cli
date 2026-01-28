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
