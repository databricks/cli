package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitchCommand_WithProfileFlag(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "my-workspace",
		Host:       "https://abc.cloud.databricks.com",
		Token:      "token1",
	})
	require.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	ctx = cmdio.MockDiscard(ctx)

	cmd := New()
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"switch", "--profile", "my-workspace"})

	err = cmd.Execute()
	require.NoError(t, err)

	got, err := databrickscfg.GetDefaultProfile(ctx, configFile)
	require.NoError(t, err)
	assert.Equal(t, "my-workspace", got)
}

func TestSwitchCommand_ProfileNotFound(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "my-workspace",
		Host:       "https://abc.cloud.databricks.com",
		Token:      "token1",
	})
	require.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	ctx = cmdio.MockDiscard(ctx)

	cmd := New()
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"switch", "--profile", "nonexistent"})

	err = cmd.Execute()
	assert.ErrorContains(t, err, `profile "nonexistent" not found`)
}

func TestSwitchCommand_NonInteractiveNoProfile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: configFile,
		Profile:    "my-workspace",
		Host:       "https://abc.cloud.databricks.com",
		Token:      "token1",
	})
	require.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	ctx = cmdio.MockDiscard(ctx)

	cmd := New()
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"switch"})

	err = cmd.Execute()
	assert.ErrorContains(t, err, "non-interactive environment")
}

func TestSwitchCommand_WritesSettingsSection(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")

	for _, name := range []string{"profile-a", "profile-b"} {
		err := databrickscfg.SaveToProfile(ctx, &config.Config{
			ConfigFile: configFile,
			Profile:    name,
			Host:       fmt.Sprintf("https://%s.cloud.databricks.com", name),
			Token:      "token",
		})
		require.NoError(t, err)
	}

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	ctx = cmdio.MockDiscard(ctx)

	cmd := New()
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"switch", "--profile", "profile-a"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify the [databricks-cli-settings] section was written.
	contents, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "[databricks-cli-settings]")
	assert.Contains(t, string(contents), "default_profile = profile-a")

	// Switch to another profile.
	cmd = New()
	cmd.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"switch", "--profile", "profile-b"})

	err = cmd.Execute()
	require.NoError(t, err)

	got, err := databrickscfg.GetDefaultProfile(ctx, configFile)
	require.NoError(t, err)
	assert.Equal(t, "profile-b", got)
}
