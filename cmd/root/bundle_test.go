package root

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDatabricksCfg(t *testing.T) {
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}

	cfg := []byte("[PROFILE-1]\nhost = https://a.com\ntoken = a\n[PROFILE-2]\nhost = https://a.com\ntoken = b\n")
	err := os.WriteFile(filepath.Join(tempHomeDir, ".databrickscfg"), cfg, 0o644)
	assert.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	t.Setenv(homeEnvVar, tempHomeDir)
}

func emptyCommand(t *testing.T) *cobra.Command {
	ctx := context.Background()
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	initProfileFlag(cmd)
	return cmd
}

func setupWithHost(t *testing.T, cmd *cobra.Command, host string) error {
	setupDatabricksCfg(t)

	rootPath := t.TempDir()
	testutil.Chdir(t, rootPath)

	contents := fmt.Sprintf(`
workspace:
  host: %q
`, host)
	err := os.WriteFile(filepath.Join(rootPath, "databricks.yml"), []byte(contents), 0o644)
	require.NoError(t, err)

	_, diags := MustConfigureBundle(cmd)
	return diags.Error()
}

func setupWithProfile(t *testing.T, cmd *cobra.Command, profile string) error {
	setupDatabricksCfg(t)

	rootPath := t.TempDir()
	testutil.Chdir(t, rootPath)

	contents := fmt.Sprintf(`
workspace:
  profile: %q
`, profile)
	err := os.WriteFile(filepath.Join(rootPath, "databricks.yml"), []byte(contents), 0o644)
	require.NoError(t, err)

	_, diags := MustConfigureBundle(cmd)
	return diags.Error()
}

func TestBundleConfigureDefault(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := setupWithHost(t, cmd, "https://x.com")
	require.NoError(t, err)

	assert.Equal(t, "https://x.com", ConfigUsed(cmd.Context()).Host)
}

func TestBundleConfigureWithMultipleMatches(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := setupWithHost(t, cmd, "https://a.com")
	assert.ErrorContains(t, err, "multiple profiles matched: PROFILE-1, PROFILE-2")
}

func TestBundleConfigureWithNonExistentProfileFlag(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("NOEXIST")
	require.NoError(t, err)

	err = setupWithHost(t, cmd, "https://x.com")
	assert.ErrorContains(t, err, "has no NOEXIST profile configured")
}

func TestBundleConfigureWithMismatchedProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-1")
	require.NoError(t, err)

	err = setupWithHost(t, cmd, "https://x.com")
	assert.ErrorContains(t, err, "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com")
}

func TestBundleConfigureWithCorrectProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-1")
	require.NoError(t, err)
	err = setupWithHost(t, cmd, "https://a.com")

	require.NoError(t, err)
	assert.Equal(t, "https://a.com", ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "PROFILE-1", ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureWithMismatchedProfileEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	t.Setenv("DATABRICKS_CONFIG_PROFILE", "PROFILE-1")
	cmd := emptyCommand(t)

	err := setupWithHost(t, cmd, "https://x.com")
	assert.ErrorContains(t, err, "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com")
}

func TestBundleConfigureWithProfileFlagAndEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	t.Setenv("DATABRICKS_CONFIG_PROFILE", "NOEXIST")
	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-1")
	require.NoError(t, err)

	err = setupWithHost(t, cmd, "https://a.com")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "PROFILE-1", ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileDefault(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The profile in the databricks.yml file is used
	cmd := emptyCommand(t)

	err := setupWithProfile(t, cmd, "PROFILE-1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "a", ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-1", ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileFlag(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The --profile flag takes precedence over the profile in the databricks.yml file
	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-2")
	require.NoError(t, err)

	err = setupWithProfile(t, cmd, "PROFILE-1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The DATABRICKS_CONFIG_PROFILE environment variable takes precedence over the profile in the databricks.yml file
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "PROFILE-2")
	cmd := emptyCommand(t)

	err := setupWithProfile(t, cmd, "PROFILE-1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileFlagAndEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The --profile flag takes precedence over the DATABRICKS_CONFIG_PROFILE environment variable
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "NOEXIST")
	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-2")
	require.NoError(t, err)

	err = setupWithProfile(t, cmd, "PROFILE-1")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", ConfigUsed(cmd.Context()).Profile)
}

func TestTargetFlagFull(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	cmd.SetArgs([]string{"version", "--target", "development"})

	ctx := context.Background()
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "development", getTarget(cmd))
}

func TestTargetFlagShort(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	cmd.SetArgs([]string{"version", "-t", "production"})

	ctx := context.Background()
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "production", getTarget(cmd))
}

// TODO: remove when environment flag is fully deprecated
func TestTargetEnvironmentFlag(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	initEnvironmentFlag(cmd)
	cmd.SetArgs([]string{"version", "--environment", "development"})

	ctx := context.Background()
	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "development", getTarget(cmd))
}
