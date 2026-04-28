package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	ucmpkg "github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessUcmTerraformEngineNoBackendError guards the Critical fix from
// commit `Fix #98: Replace phases.Initialize with pre-fork mutator chain`.
// The pre-fix ProcessUcm called phases.Initialize with a zero-value
// phases.Options{}, which made deploy.Pull reject the empty Backend and
// short-circuit every terraform-engine UCM verb to ErrAlreadyPrinted before
// the verb body ran. The new ProcessUcm runs only the workspace-context
// mutators and variable resolution, deferring state pull to the verb's own
// phase calls — so terraform-engine ProcessUcm should NOT log an error here.
func TestProcessUcmTerraformEngineNoBackendError(t *testing.T) {
	testutil.CleanupEnvironment(t)
	// Restrict PATH so the SDK's auth resolution cannot invoke az/gcloud.
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", `C:\Windows\System32`)
	} else {
		t.Setenv("PATH", "/usr/bin:/bin")
	}
	setupDatabricksCfg(t)

	rootPath := t.TempDir()
	t.Chdir(rootPath)

	y := "ucm:\n  name: t\n  engine: terraform\n\nworkspace:\n  host: https://unique.example.com\n  profile: PROFILE-UNIQUE\n"
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "ucm.yml"), []byte(y), 0o644))

	cmd := &cobra.Command{Use: "validate"}
	cmd.Flags().String("target", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flags().StringSlice("var", nil, "")
	cmd.SetContext(cmdio.MockDiscard(t.Context()))

	got, err := ProcessUcm(cmd, ProcessOptions{
		// Seed CurrentUser so PopulateCurrentUser short-circuits without
		// hitting the network. Mirrors the PreMutateHook seam used by other
		// ucm tests; InitFunc runs before the mutator chain in our fork.
		InitFunc: func(u *ucmpkg.Ucm) {
			u.CurrentUser = &config.User{
				ShortName: "test-user",
				User:      &iam.User{UserName: "test-user@example.com"},
			}
		},
	})

	require.NotNil(t, got)
	if err != nil {
		t.Logf("ProcessUcm error: %v; first diag: %s", err, logdiag.GetFirstErrorSummary(cmd.Context()))
	}
	require.NoError(t, err)
	assert.False(t, logdiag.HasError(cmd.Context()),
		"ProcessUcm must not log errors during the workspace-context-only Initialize stand-in: %s",
		logdiag.GetFirstErrorSummary(cmd.Context()))
}

// ResolveEngineSetting tests cover the priority chain: ucm.engine config >
// DATABRICKS_UCM_ENGINE env var > Default. Mirrors the bundle parallel in
// cmd/bundle/utils/resolve_engine_test.go but checks the ucm-flavoured Source
// labels ("config" / "env" / "default") rather than bundle's longer
// descriptions.

func TestResolveEngineSettingConfigTakesPriority(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "terraform")
	u := &config.Ucm{Engine: engine.EngineDirect}
	got, err := ResolveEngineSetting(ctx, u)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineDirect, got.Type)
	assert.Equal(t, engine.EngineDirect, got.ConfigType)
	assert.Equal(t, "config", got.Source)
}

func TestResolveEngineSettingConfigOverridesInvalidEnv(t *testing.T) {
	// An invalid env var is ignored when the config already selects an engine.
	ctx := env.Set(t.Context(), engine.EnvVar, "bogus")
	u := &config.Ucm{Engine: engine.EngineTerraform}
	got, err := ResolveEngineSetting(ctx, u)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineTerraform, got.Type)
	assert.Equal(t, "config", got.Source)
}

func TestResolveEngineSettingFallsBackToEnv(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "direct")
	got, err := ResolveEngineSetting(ctx, &config.Ucm{})
	require.NoError(t, err)
	assert.Equal(t, engine.EngineDirect, got.Type)
	assert.Equal(t, engine.EngineNotSet, got.ConfigType)
	assert.Equal(t, "env", got.Source)
}

func TestResolveEngineSettingDefault(t *testing.T) {
	got, err := ResolveEngineSetting(t.Context(), &config.Ucm{})
	require.NoError(t, err)
	assert.Equal(t, engine.EngineTerraform, got.Type)
	assert.Equal(t, engine.EngineNotSet, got.ConfigType)
	assert.Equal(t, "default", got.Source)
}

func TestResolveEngineSettingNilUcm(t *testing.T) {
	got, err := ResolveEngineSetting(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, engine.EngineTerraform, got.Type)
	assert.Equal(t, "default", got.Source)
}

func TestResolveEngineSettingInvalidEnv(t *testing.T) {
	ctx := env.Set(t.Context(), engine.EnvVar, "bogus")
	_, err := ResolveEngineSetting(ctx, &config.Ucm{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), engine.EnvVar)
}
