package aitools

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScopeTest(t *testing.T) (homeDir, projectDir string) {
	t.Helper()
	homeDir = t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir = t.TempDir()
	t.Chdir(projectDir)

	return homeDir, projectDir
}

func installGlobalState(t *testing.T, homeDir string) {
	t.Helper()
	globalDir := filepath.Join(homeDir, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, installer.SaveState(globalDir, &installer.InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		Skills:        map[string]string{"test-skill": "1.0.0"},
	}))
}

func installProjectState(t *testing.T, projectDir string) {
	t.Helper()
	projSkillsDir := filepath.Join(projectDir, ".databricks", "aitools", "skills")
	require.NoError(t, os.MkdirAll(projSkillsDir, 0o755))
	require.NoError(t, installer.SaveState(projSkillsDir, &installer.InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		Skills:        map[string]string{"test-skill": "1.0.0"},
		Scope:         installer.ScopeProject,
	}))
}

func nonInteractiveCtx(t *testing.T) context.Context {
	t.Helper()
	return cmdio.MockDiscard(t.Context())
}

func interactiveCtx(t *testing.T) (context.Context, func()) {
	t.Helper()
	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})

	drain := func(r *bufio.Reader) {
		buf := make([]byte, 4096)
		for {
			_, err := r.Read(buf)
			if err != nil {
				return
			}
		}
	}
	go drain(test.Stdout)
	go drain(test.Stderr)

	return ctx, test.Done
}

// --- detectInstalledScopes tests ---

func TestDetectInstalledScopesBoth(t *testing.T) {
	homeDir, projectDir := setupScopeTest(t)
	installGlobalState(t, homeDir)
	installProjectState(t, projectDir)

	ctx := nonInteractiveCtx(t)
	hasGlobal, hasProject, err := detectInstalledScopes(ctx)
	require.NoError(t, err)
	assert.True(t, hasGlobal)
	assert.True(t, hasProject)
}

func TestDetectInstalledScopesGlobalOnly(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)

	ctx := nonInteractiveCtx(t)
	hasGlobal, hasProject, err := detectInstalledScopes(ctx)
	require.NoError(t, err)
	assert.True(t, hasGlobal)
	assert.False(t, hasProject)
}

func TestDetectInstalledScopesProjectOnly(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)

	ctx := nonInteractiveCtx(t)
	hasGlobal, hasProject, err := detectInstalledScopes(ctx)
	require.NoError(t, err)
	assert.False(t, hasGlobal)
	assert.True(t, hasProject)
}

func TestDetectInstalledScopesNeither(t *testing.T) {
	setupScopeTest(t)

	ctx := nonInteractiveCtx(t)
	hasGlobal, hasProject, err := detectInstalledScopes(ctx)
	require.NoError(t, err)
	assert.False(t, hasGlobal)
	assert.False(t, hasProject)
}

// --- resolveScopeForUpdate tests ---

func TestResolveScopeForUpdateBothFlagsBothInstalled(t *testing.T) {
	homeDir, projectDir := setupScopeTest(t)
	installGlobalState(t, homeDir)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, true, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal, installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateBothFlagsOnlyGlobalInstalled(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, true, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateBothFlagsOnlyProjectInstalled(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	// Global always passes through, project state found.
	scopes, err := resolveScopeForUpdate(ctx, true, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal, installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateBothFlagsNeitherInstalled(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Global always passes through, project check fails silently.
	scopes, err := resolveScopeForUpdate(ctx, true, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateProjectFlagWithState(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, true, false)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateGlobalFlagWithState(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateProjectFlagNoInstall(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUpdate(ctx, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project-scoped skills found")
	assert.Contains(t, err.Error(), "install --project")
	assert.Contains(t, err.Error(), "Expected location:")
	assert.Contains(t, err.Error(), ".databricks/aitools/skills/")
}

func TestResolveScopeForUpdateGlobalFlagNoInstall(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Global flag always passes through to installer layer (for legacy install detection).
	scopes, err := resolveScopeForUpdate(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateNoFlagsGlobalOnly(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, false, false)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateNoFlagsProjectOnly(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, false, false)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateNoFlagsBothNonInteractive(t *testing.T) {
	homeDir, projectDir := setupScopeTest(t)
	installGlobalState(t, homeDir)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUpdate(ctx, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skills are installed in both global and project scopes")
	assert.Contains(t, err.Error(), "--global")
	assert.Contains(t, err.Error(), "--project")
}

func TestResolveScopeForUpdateNoFlagsBothInteractive(t *testing.T) {
	homeDir, projectDir := setupScopeTest(t)
	installGlobalState(t, homeDir)
	installProjectState(t, projectDir)

	orig := promptUpdateScopeSelection
	t.Cleanup(func() { promptUpdateScopeSelection = orig })
	promptCalled := false
	promptUpdateScopeSelection = func(_ context.Context) ([]string, error) {
		promptCalled = true
		return []string{installer.ScopeGlobal}, nil
	}

	ctx, done := interactiveCtx(t)
	defer done()

	scopes, err := resolveScopeForUpdate(ctx, false, false)
	require.NoError(t, err)
	assert.True(t, promptCalled)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateNoFlagsNeitherInstalled(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Falls through to global so the installer layer can handle legacy installs.
	scopes, err := resolveScopeForUpdate(ctx, false, false)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

// --- resolveScopeForUninstall tests ---

func TestResolveScopeForUninstallBothFlagsError(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUninstall(ctx, true, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot uninstall both scopes at once")
}

func TestResolveScopeForUninstallProjectFlagWithState(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, true, false)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeProject, scope)
}

func TestResolveScopeForUninstallGlobalFlagWithState(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUninstallProjectFlagNoInstall(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUninstall(ctx, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project-scoped skills found")
	assert.Contains(t, err.Error(), "install --project")
	assert.Contains(t, err.Error(), "Expected location:")
}

func TestResolveScopeForUninstallGlobalFlagNoInstall(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Global flag always passes through to installer layer (for legacy install detection).
	scope, err := resolveScopeForUninstall(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUninstallNoFlagsGlobalOnly(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, false, false)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUninstallNoFlagsProjectOnly(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, false, false)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeProject, scope)
}

func TestResolveScopeForUninstallNoFlagsBothNonInteractive(t *testing.T) {
	homeDir, projectDir := setupScopeTest(t)
	installGlobalState(t, homeDir)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUninstall(ctx, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skills are installed in both global and project scopes")
	assert.Contains(t, err.Error(), "--global")
	assert.Contains(t, err.Error(), "--project")
}

func TestResolveScopeForUninstallNoFlagsBothInteractive(t *testing.T) {
	homeDir, projectDir := setupScopeTest(t)
	installGlobalState(t, homeDir)
	installProjectState(t, projectDir)

	orig := promptUninstallScopeSelection
	t.Cleanup(func() { promptUninstallScopeSelection = orig })
	promptCalled := false
	promptUninstallScopeSelection = func(_ context.Context) (string, error) {
		promptCalled = true
		return installer.ScopeProject, nil
	}

	ctx, done := interactiveCtx(t)
	defer done()

	scope, err := resolveScopeForUninstall(ctx, false, false)
	require.NoError(t, err)
	assert.True(t, promptCalled)
	assert.Equal(t, installer.ScopeProject, scope)
}

func TestResolveScopeForUninstallNoFlagsNeitherInstalled(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Falls through to global so the installer layer can handle legacy installs.
	scope, err := resolveScopeForUninstall(ctx, false, false)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

// --- withExplicitScopeCheck tests ---

func TestWithExplicitScopeCheckProjectPresent(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := withExplicitScopeCheck(ctx, installer.ScopeProject, "update")
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeProject}, scopes)
}

func TestWithExplicitScopeCheckGlobalPresent(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := withExplicitScopeCheck(ctx, installer.ScopeGlobal, "update")
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestWithExplicitScopeCheckProjectMissing(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeProject, "update")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project-scoped skills found")
	assert.Contains(t, err.Error(), "install --project")
	assert.Contains(t, err.Error(), "Expected location:")
}

func TestWithExplicitScopeCheckGlobalMissing(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeGlobal, "update")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no globally-scoped skills installed")
	assert.Contains(t, err.Error(), "install --global")
}

// --- cross-scope hint tests ---

func TestWithExplicitScopeCheckProjectMissingHintsGlobalUpdate(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeProject, "update")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Global skills are installed")
	assert.Contains(t, err.Error(), "without --project to update those")
}

func TestWithExplicitScopeCheckProjectMissingHintsGlobalUninstall(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeProject, "uninstall")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Global skills are installed")
	assert.Contains(t, err.Error(), "without --project to uninstall those")
}

func TestWithExplicitScopeCheckGlobalMissingHintsProject(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeGlobal, "update")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Project-scoped skills are installed")
	assert.Contains(t, err.Error(), "without --global to update those")
}

func TestWithExplicitScopeCheckNoHintWhenNeitherInstalled(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeProject, "update")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "Global skills are installed")
	assert.NotContains(t, err.Error(), "Project-scoped skills are installed")
}

// --- legacy global install passthrough tests ---

func TestResolveScopeForUpdateGlobalFlagLegacyInstall(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	// Create skill directories on disk but no .state.json (legacy install).
	globalDir := filepath.Join(homeDir, ".databricks", "aitools", "skills", "some-skill")
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	ctx := nonInteractiveCtx(t)

	// Global flag should pass through to installer layer, not block on missing state.
	scopes, err := resolveScopeForUpdate(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUninstallGlobalFlagLegacyInstall(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	// Create skill directories on disk but no .state.json (legacy install).
	globalDir := filepath.Join(homeDir, ".databricks", "aitools", "skills", "some-skill")
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	ctx := nonInteractiveCtx(t)

	// Global flag should pass through to installer layer, not block on missing state.
	scope, err := resolveScopeForUninstall(ctx, false, true)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUpdateBothFlagsLegacyGlobal(t *testing.T) {
	homeDir, projectDir := setupScopeTest(t)
	// Global has a legacy install (dirs on disk, no state), project has state.
	globalDir := filepath.Join(homeDir, ".databricks", "aitools", "skills", "some-skill")
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	// Global passes through (for legacy detection), project has state so it's included.
	scopes, err := resolveScopeForUpdate(ctx, true, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal, installer.ScopeProject}, scopes)
}

// --- uninstall cross-scope hint verb tests ---

func TestResolveScopeForUninstallProjectFlagHintsUninstall(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	// Project flag with no project state should hint about global using "uninstall" verb.
	_, err := resolveScopeForUninstall(ctx, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without --project to uninstall those")
}

func TestResolveScopeForUpdateProjectFlagHintsUpdate(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	// Project flag with no project state should hint about global using "update" verb.
	_, err := resolveScopeForUpdate(ctx, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without --project to update those")
}
