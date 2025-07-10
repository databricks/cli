package root

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
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

func setupWithHost(t *testing.T, cmd *cobra.Command, host string) []diag.Diagnostic {
	setupDatabricksCfg(t)

	rootPath := t.TempDir()
	testutil.Chdir(t, rootPath)

	contents := fmt.Sprintf(`
workspace:
  host: %q
`, host)
	err := os.WriteFile(filepath.Join(rootPath, "databricks.yml"), []byte(contents), 0o644)
	require.NoError(t, err)

	ctx := logdiag.InitContext(cmd.Context())
	logdiag.SetCollect(ctx, true)
	cmd.SetContext(ctx)
	_ = MustConfigureBundle(cmd)
	return logdiag.FlushCollected(ctx)
}

func setupWithProfile(t *testing.T, cmd *cobra.Command, profile string) []diag.Diagnostic {
	setupDatabricksCfg(t)

	rootPath := t.TempDir()
	testutil.Chdir(t, rootPath)

	contents := fmt.Sprintf(`
workspace:
  profile: %q
`, profile)
	err := os.WriteFile(filepath.Join(rootPath, "databricks.yml"), []byte(contents), 0o644)
	require.NoError(t, err)

	ctx := logdiag.InitContext(cmd.Context())
	logdiag.SetCollect(ctx, true)
	cmd.SetContext(ctx)
	_ = MustConfigureBundle(cmd)
	return logdiag.FlushCollected(ctx)
}

func TestBundleConfigureDefault(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	diags := setupWithHost(t, cmd, "https://x.com")
	require.Empty(t, diags)

	assert.Equal(t, "https://x.com", cmdctx.ConfigUsed(cmd.Context()).Host)
}

func TestBundleConfigureWithMultipleMatches(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	diags := setupWithHost(t, cmd, "https://a.com")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "multiple profiles matched: PROFILE-1, PROFILE-2")
}

func TestBundleConfigureWithNonExistentProfileFlag(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("NOEXIST")
	require.NoError(t, err)

	diags := setupWithHost(t, cmd, "https://x.com")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "has no NOEXIST profile configured")
}

func TestBundleConfigureWithMismatchedProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-1")
	require.NoError(t, err)

	diags := setupWithHost(t, cmd, "https://x.com")
	assert.Equal(t, []diag.Diagnostic{{Summary: "cannot resolve bundle auth configuration: the host in the profile (https://a.com) doesn’t match the host configured in the bundle (https://x.com)"}}, diags)
}

func TestBundleConfigureWithCorrectProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-1")
	require.NoError(t, err)
	diags := setupWithHost(t, cmd, "https://a.com")

	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "PROFILE-1", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureWithMismatchedProfileEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	t.Setenv("DATABRICKS_CONFIG_PROFILE", "PROFILE-1")
	cmd := emptyCommand(t)

	diags := setupWithHost(t, cmd, "https://x.com")
	assert.Equal(t, []diag.Diagnostic{{Summary: "cannot resolve bundle auth configuration: the host in the profile (https://a.com) doesn’t match the host configured in the bundle (https://x.com)"}}, diags)
}

func TestBundleConfigureWithProfileFlagAndEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	t.Setenv("DATABRICKS_CONFIG_PROFILE", "NOEXIST")
	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-1")
	require.NoError(t, err)

	diags := setupWithHost(t, cmd, "https://a.com")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "PROFILE-1", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileDefault(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The profile in the databricks.yml file is used
	cmd := emptyCommand(t)

	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "a", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-1", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileFlag(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The --profile flag takes precedence over the profile in the databricks.yml file
	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-2")
	require.NoError(t, err)

	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The DATABRICKS_CONFIG_PROFILE environment variable takes precedence over the profile in the databricks.yml file
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "PROFILE-2")
	cmd := emptyCommand(t)

	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestBundleConfigureProfileFlagAndEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	// The --profile flag takes precedence over the DATABRICKS_CONFIG_PROFILE environment variable
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "NOEXIST")
	cmd := emptyCommand(t)
	err := cmd.Flag("profile").Value.Set("PROFILE-2")
	require.NoError(t, err)

	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestTargetFlagFull(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	cmd.SetArgs([]string{"version", "--target", "development"})

	ctx := context.Background()
	err := Execute(ctx, cmd)
	assert.NoError(t, err)

	assert.Equal(t, "development", getTarget(cmd))
}

func TestTargetFlagShort(t *testing.T) {
	cmd := emptyCommand(t)
	initTargetFlag(cmd)
	cmd.SetArgs([]string{"version", "-t", "production"})

	ctx := context.Background()
	err := Execute(ctx, cmd)
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
	err := Execute(ctx, cmd)
	assert.NoError(t, err)

	assert.Equal(t, "development", getTarget(cmd))
}
