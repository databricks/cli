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

func setupScopeTest(t *testing.T) (homeDir string, projectDir string) {
	t.Helper()
	homeDir = t.TempDir()
	t.Setenv("HOME", homeDir)

	projectDir = t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() { os.Chdir(origDir) })

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

func TestResolveScopeForUpdateBothFlags(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, true, true)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal, installer.ScopeProject}, scopes)
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
	assert.Contains(t, err.Error(), "no project-scoped skills installed")
	assert.Contains(t, err.Error(), "Make sure you're in the project root")
}

func TestResolveScopeForUpdateGlobalFlagNoInstall(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUpdate(ctx, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no globally-scoped skills installed")
	assert.Contains(t, err.Error(), "install --global")
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

	_, err := resolveScopeForUpdate(ctx, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no skills installed")
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
	assert.Contains(t, err.Error(), "no project-scoped skills installed")
	assert.Contains(t, err.Error(), "Make sure you're in the project root")
}

func TestResolveScopeForUninstallGlobalFlagNoInstall(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUninstall(ctx, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no globally-scoped skills installed")
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

	_, err := resolveScopeForUninstall(ctx, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no skills installed")
}

// --- withExplicitScopeCheck tests ---

func TestWithExplicitScopeCheckProjectPresent(t *testing.T) {
	_, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := withExplicitScopeCheck(ctx, installer.ScopeProject)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeProject}, scopes)
}

func TestWithExplicitScopeCheckGlobalPresent(t *testing.T) {
	homeDir, _ := setupScopeTest(t)
	installGlobalState(t, homeDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := withExplicitScopeCheck(ctx, installer.ScopeGlobal)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestWithExplicitScopeCheckProjectMissing(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeProject)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project-scoped skills installed")
	assert.Contains(t, err.Error(), "Make sure you're in the project root")
}

func TestWithExplicitScopeCheckGlobalMissing(t *testing.T) {
	setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := withExplicitScopeCheck(ctx, installer.ScopeGlobal)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no globally-scoped skills installed")
	assert.Contains(t, err.Error(), "install --global")
}
