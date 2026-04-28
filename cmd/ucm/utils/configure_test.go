package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupConfigureCfg writes a databrickscfg with a single host (a.com) and two
// profiles. Mirrors cmd/root/bundle_test.go.setupDatabricksCfg so the
// MustConfigureUcm tests below cover the same matrix as MustConfigureBundle.
// Kept distinct from cmd/ucm/utils/auth_test.go.setupDatabricksCfg, which
// targets MustWorkspaceClient ambiguity and uses a different fixture.
func setupConfigureCfg(t *testing.T) {
	t.Helper()
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}

	cfg := []byte("[PROFILE-1]\nhost = https://a.com\ntoken = a\n[PROFILE-2]\nhost = https://a.com\ntoken = b\n")
	require.NoError(t, os.WriteFile(filepath.Join(tempHomeDir, ".databrickscfg"), cfg, 0o644))

	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	t.Setenv(homeEnvVar, tempHomeDir)
}

// emptyCommand builds a cobra command primed with a discard cmdio context and
// the persistent --profile / --target flags MustConfigureUcm reads off
// cmd.Flags(). Mirrors cmd/root/bundle_test.go.emptyCommand.
func emptyCommand(t *testing.T) *cobra.Command {
	t.Helper()
	ctx := t.Context()
	ctx = cmdio.MockDiscard(ctx)
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.PersistentFlags().String("profile", "", "")
	cmd.PersistentFlags().String("target", "", "")
	return cmd
}

// setupWithHost writes a ucm.yml with the given workspace.host and runs
// MustConfigureUcm. Returns the diagnostics MustConfigureUcm collected so
// individual tests can assert on the outcome.
func setupWithHost(t *testing.T, cmd *cobra.Command, host string) []diag.Diagnostic {
	t.Helper()
	setupConfigureCfg(t)

	rootPath := t.TempDir()
	t.Chdir(rootPath)

	contents := fmt.Sprintf("ucm:\n  name: t\nworkspace:\n  host: %q\n", host)
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "ucm.yml"), []byte(contents), 0o644))

	ctx := logdiag.InitContext(cmd.Context())
	logdiag.SetCollect(ctx, true)
	cmd.SetContext(ctx)
	_ = MustConfigureUcm(cmd)
	return logdiag.FlushCollected(ctx)
}

// setupWithProfile is the profile-anchored counterpart of setupWithHost.
func setupWithProfile(t *testing.T, cmd *cobra.Command, profile string) []diag.Diagnostic {
	t.Helper()
	setupConfigureCfg(t)

	rootPath := t.TempDir()
	t.Chdir(rootPath)

	contents := fmt.Sprintf("ucm:\n  name: t\nworkspace:\n  profile: %q\n", profile)
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "ucm.yml"), []byte(contents), 0o644))

	ctx := logdiag.InitContext(cmd.Context())
	logdiag.SetCollect(ctx, true)
	cmd.SetContext(ctx)
	_ = MustConfigureUcm(cmd)
	return logdiag.FlushCollected(ctx)
}

func TestUcmConfigureDefault(t *testing.T) {
	testutil.CleanupEnvironment(t)
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", `C:\Windows\System32`)
	} else {
		t.Setenv("PATH", "/usr/bin:/bin")
	}

	cmd := emptyCommand(t)
	diags := setupWithHost(t, cmd, "https://x.com")
	require.Empty(t, diags)

	assert.Equal(t, "https://x.com", cmdctx.ConfigUsed(cmd.Context()).Host)
}

func TestUcmConfigureWithMultipleMatches(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	diags := setupWithHost(t, cmd, "https://a.com")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "multiple profiles matched: PROFILE-1, PROFILE-2")
	assert.Contains(t, diags[0].Summary, "Matching workspace profiles: PROFILE-1, PROFILE-2")
	assert.Contains(t, diags[0].Summary, "DATABRICKS_CONFIG_PROFILE=PROFILE-1")
}

func TestUcmConfigureWithNonExistentProfileFlag(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	require.NoError(t, cmd.Flag("profile").Value.Set("NOEXIST"))

	diags := setupWithHost(t, cmd, "https://x.com")
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "has no NOEXIST profile configured")
}

func TestUcmConfigureWithCorrectProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	require.NoError(t, cmd.Flag("profile").Value.Set("PROFILE-1"))

	diags := setupWithHost(t, cmd, "https://a.com")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "PROFILE-1", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestUcmConfigureProfileDefault(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "a", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-1", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestUcmConfigureProfileFlag(t *testing.T) {
	testutil.CleanupEnvironment(t)

	cmd := emptyCommand(t)
	require.NoError(t, cmd.Flag("profile").Value.Set("PROFILE-2"))

	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestUcmConfigureProfileEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	t.Setenv("DATABRICKS_CONFIG_PROFILE", "PROFILE-2")
	cmd := emptyCommand(t)

	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

func TestUcmConfigureProfileFlagAndEnvVariable(t *testing.T) {
	testutil.CleanupEnvironment(t)

	t.Setenv("DATABRICKS_CONFIG_PROFILE", "NOEXIST")
	cmd := emptyCommand(t)
	require.NoError(t, cmd.Flag("profile").Value.Set("PROFILE-2"))

	diags := setupWithProfile(t, cmd, "PROFILE-1")
	require.Empty(t, diags)
	assert.Equal(t, "https://a.com", cmdctx.ConfigUsed(cmd.Context()).Host)
	assert.Equal(t, "b", cmdctx.ConfigUsed(cmd.Context()).Token)
	assert.Equal(t, "PROFILE-2", cmdctx.ConfigUsed(cmd.Context()).Profile)
}

// TestUcmConfigureMultiMatchInteractivePromptFires mirrors the bundle parallel:
// when multiple profiles match a host and the prompt is supported, we surface
// the search prompt rather than emitting a guidance error.
func TestUcmConfigureMultiMatchInteractivePromptFires(t *testing.T) {
	testutil.CleanupEnvironment(t)

	setupConfigureCfg(t)

	rootPath := t.TempDir()
	t.Chdir(rootPath)

	contents := "ucm:\n  name: t\nworkspace:\n  host: \"https://a.com\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "ucm.yml"), []byte(contents), 0o644))

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	ctx, io := cmdio.SetupTest(ctx, cmdio.TestOptions{PromptSupported: true})
	defer io.Done()

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.PersistentFlags().String("profile", "", "")
	cmd.PersistentFlags().String("target", "", "")

	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx := logdiag.InitContext(cmd.Context())
		logdiag.SetCollect(ctx, true)
		cmd.SetContext(ctx)
		_ = MustConfigureUcm(cmd)
	}()

	// Verify the prompt fires by reading output from stderr.
	// promptui with StartInSearchMode writes a search cursor first.
	line, _, readErr := io.Stderr.ReadLine()
	if assert.NoError(t, readErr, "expected prompt output on stderr") {
		assert.Contains(t, string(line), "Search:")
	}

	// Cancel to unblock the prompt.
	cancel()
	<-done
}

// TestTryConfigureUcm_ReturnsNilWhenNoUcmYml verifies the TryConfigureUcm
// contract: missing ucm.yml is not an error, the function quietly returns nil.
// Mirrors bundle.TryLoad's behaviour and is the seam shareable verbs (auth,
// configure) call when ucm context is optional.
func TestTryConfigureUcm_ReturnsNilWhenNoUcmYml(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Chdir(t.TempDir())

	cmd := emptyCommand(t)
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)
	got := TryConfigureUcm(cmd)
	assert.Nil(t, got)
	assert.False(t, logdiag.HasError(ctx))
}
