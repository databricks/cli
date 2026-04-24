package phases_test

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// markerScript returns a Script that creates a marker file inside dir so
// phase tests can assert the hook fired without scraping stdout.
func markerScript(dir, name string) config.Script {
	return config.Script{Content: "touch " + filepath.ToSlash(filepath.Join(dir, name))}
}

func markerExists(t *testing.T, dir, name string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func TestInitializeRunsPreAndPostInitScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution test skipped on windows")
	}
	f := newFixture(t)
	markerDir := t.TempDir()
	f.u.Config.Scripts = map[string]config.Script{
		config.ScriptPreInit:  markerScript(markerDir, "pre_init"),
		config.ScriptPostInit: markerScript(markerDir, "post_init"),
	}

	ctx, _ := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))
	logdiag.SetCollect(ctx, true)

	setting := phases.Initialize(ctx, f.u, phases.Options{Backend: f.backend})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.Equal(t, engine.EngineTerraform, setting.Type)
	assert.True(t, markerExists(t, markerDir, "pre_init"))
	assert.True(t, markerExists(t, markerDir, "post_init"))
}

func TestDeployRunsPreAndPostDeployScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution test skipped on windows")
	}
	f := newFixture(t)
	markerDir := t.TempDir()
	f.u.Config.Scripts = map[string]config.Script{
		config.ScriptPreDeploy:  markerScript(markerDir, "pre_deploy"),
		config.ScriptPostDeploy: markerScript(markerDir, "post_deploy"),
	}

	ctx, _ := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))
	logdiag.SetCollect(ctx, true)

	phases.Deploy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.True(t, markerExists(t, markerDir, "pre_deploy"))
	assert.True(t, markerExists(t, markerDir, "post_deploy"))
}

func TestDestroyRunsPreAndPostDestroyScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution test skipped on windows")
	}
	f := newFixture(t)
	markerDir := t.TempDir()
	f.u.Config.Scripts = map[string]config.Script{
		config.ScriptPreDestroy:  markerScript(markerDir, "pre_destroy"),
		config.ScriptPostDestroy: markerScript(markerDir, "post_destroy"),
	}

	ctx, _ := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	assert.True(t, markerExists(t, markerDir, "pre_destroy"))
	assert.True(t, markerExists(t, markerDir, "post_destroy"))
}

// TestDestroyNoopWhenRootMissing asserts destroy scripts do NOT fire when the
// workspace root doesn't exist (no active deployment). Mirrors the early
// return semantics of the Destroy phase.
func TestDestroyNoopWhenRootMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution test skipped on windows")
	}
	f := newFixture(t)
	markerDir := t.TempDir()
	f.u.Config.Scripts = map[string]config.Script{
		config.ScriptPreDestroy:  markerScript(markerDir, "pre_destroy"),
		config.ScriptPostDestroy: markerScript(markerDir, "post_destroy"),
	}

	// Re-stub GetStatusByPath to 404 — assertRootPathExists must short-circuit
	// Destroy, and no scripts should fire.
	mockWS := f.mockWS
	mockWS.GetMockWorkspaceAPI().ExpectedCalls = nil
	mockWS.GetMockWorkspaceAPI().EXPECT().
		GetStatusByPath(mock.Anything, f.u.Config.Workspace.RootPath).
		Return((*workspace.ObjectInfo)(nil), &apierr.APIError{StatusCode: http.StatusNotFound})

	ctx, _ := cmdio.NewTestContextWithStderr(logdiag.InitContext(t.Context()))
	logdiag.SetCollect(ctx, true)

	phases.Destroy(ctx, f.u, phases.Options{
		Backend:          f.backend,
		TerraformFactory: fakeTfFactory(f.tf),
	})

	require.False(t, logdiag.HasError(ctx))
	assert.False(t, markerExists(t, markerDir, "pre_destroy"))
	assert.False(t, markerExists(t, markerDir, "post_destroy"))
}
