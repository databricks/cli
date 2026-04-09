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

func setupScopeTest(t *testing.T) (globalDir, projectDir string) {
	t.Helper()
	homeDir := t.TempDir()
	projectRoot := t.TempDir()
	globalDir = filepath.Join(homeDir, ".databricks", "aitools", "skills")
	projectDir = filepath.Join(projectRoot, ".databricks", "aitools", "skills")
	return globalDir, projectDir
}

func installGlobalState(t *testing.T, globalDir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(globalDir, 0o755))
	require.NoError(t, installer.SaveState(globalDir, &installer.InstallState{
		SchemaVersion: 1,
		Release:       "v0.1.0",
		Skills:        map[string]string{"test-skill": "1.0.0"},
	}))
}

func installProjectState(t *testing.T, projectDir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, installer.SaveState(projectDir, &installer.InstallState{
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

// --- detectInstalledScopes tests (table-driven) ---

func TestDetectInstalledScopes(t *testing.T) {
	tests := []struct {
		name        string
		installGlob bool
		installProj bool
		wantGlobal  bool
		wantProject bool
	}{
		{"Both", true, true, true, true},
		{"GlobalOnly", true, false, true, false},
		{"ProjectOnly", false, true, false, true},
		{"Neither", false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalDir, projectDir := setupScopeTest(t)
			if tt.installGlob {
				installGlobalState(t, globalDir)
			}
			if tt.installProj {
				installProjectState(t, projectDir)
			}

			hasGlobal, hasProject, err := detectInstalledScopes(globalDir, projectDir)
			require.NoError(t, err)
			assert.Equal(t, tt.wantGlobal, hasGlobal)
			assert.Equal(t, tt.wantProject, hasProject)
		})
	}
}

// --- withExplicitScopeCheck tests (table-driven) ---

func TestWithExplicitScopeCheck(t *testing.T) {
	tests := []struct {
		name        string
		scope       string
		installGlob bool
		installProj bool
		wantScopes  []string
		wantErr     string
	}{
		{
			name:        "ProjectPresent",
			scope:       installer.ScopeProject,
			installProj: true,
			wantScopes:  []string{installer.ScopeProject},
		},
		{
			name:        "GlobalPresent",
			scope:       installer.ScopeGlobal,
			installGlob: true,
			wantScopes:  []string{installer.ScopeGlobal},
		},
		{
			name:    "ProjectMissing",
			scope:   installer.ScopeProject,
			wantErr: "no project-scoped skills found",
		},
		{
			name:    "GlobalMissing",
			scope:   installer.ScopeGlobal,
			wantErr: "no globally-scoped skills installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalDir, projectDir := setupScopeTest(t)
			if tt.installGlob {
				installGlobalState(t, globalDir)
			}
			if tt.installProj {
				installProjectState(t, projectDir)
			}

			hasGlobal, hasProject, err := detectInstalledScopes(globalDir, projectDir)
			require.NoError(t, err)

			var dir string
			if tt.scope == installer.ScopeProject {
				dir = projectDir
			} else {
				dir = globalDir
			}

			scopes, err := withExplicitScopeCheck(dir, tt.scope, "update", projectDir, hasGlobal, hasProject)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantScopes, scopes)
			}
		})
	}
}

// --- cross-scope hint tests ---

func TestCrossScopeHint(t *testing.T) {
	tests := []struct {
		name      string
		scope     string
		verb      string
		hasGlobal bool
		hasProj   bool
		wantHint  string
	}{
		{
			name:      "ProjectMissingHintsGlobalUpdate",
			scope:     installer.ScopeProject,
			verb:      "update",
			hasGlobal: true,
			wantHint:  "without --project to update those",
		},
		{
			name:      "ProjectMissingHintsGlobalUninstall",
			scope:     installer.ScopeProject,
			verb:      "uninstall",
			hasGlobal: true,
			wantHint:  "without --project to uninstall those",
		},
		{
			name:     "GlobalMissingHintsProject",
			scope:    installer.ScopeGlobal,
			verb:     "update",
			hasProj:  true,
			wantHint: "without --global to update those",
		},
		{
			name:  "NeitherInstalledNoHint",
			scope: installer.ScopeProject,
			verb:  "update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := crossScopeHint(tt.scope, tt.verb, tt.hasGlobal, tt.hasProj)
			if tt.wantHint != "" {
				assert.Contains(t, hint, tt.wantHint)
			} else {
				assert.Empty(t, hint)
			}
		})
	}
}

// --- scopeNotInstalledError tests ---

func TestScopeNotInstalledErrorProjectIncludesPath(t *testing.T) {
	projectDir := "/some/project"
	err := scopeNotInstalledError(installer.ScopeProject, "update", projectDir, false, false)
	assert.Contains(t, err.Error(), "no project-scoped skills found")
	assert.Contains(t, err.Error(), "install --project")
	assert.Contains(t, err.Error(), "Expected location:")
	assert.Contains(t, err.Error(), "/some/project/.databricks/aitools/skills/")
}

func TestScopeNotInstalledErrorGlobal(t *testing.T) {
	err := scopeNotInstalledError(installer.ScopeGlobal, "update", "/irrelevant", false, false)
	assert.Contains(t, err.Error(), "no globally-scoped skills installed")
	assert.Contains(t, err.Error(), "install --global")
}

// --- resolveScopeForUpdate tests ---

func TestResolveScopeForUpdateBothFlagsBothInstalled(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, true, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal, installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateBothFlagsOnlyGlobalInstalled(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, true, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateBothFlagsOnlyProjectInstalled(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	// Global always passes through, project state found.
	scopes, err := resolveScopeForUpdate(ctx, true, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal, installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateBothFlagsNeitherInstalled(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Global always passes through, project check fails silently.
	scopes, err := resolveScopeForUpdate(ctx, true, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateProjectFlagWithState(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, true, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateGlobalFlagWithState(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, false, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateProjectFlagNoInstall(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUpdate(ctx, true, false, globalDir, projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project-scoped skills found")
	assert.Contains(t, err.Error(), "install --project")
	assert.Contains(t, err.Error(), "Expected location:")
	assert.Contains(t, err.Error(), ".databricks/aitools/skills/")
}

func TestResolveScopeForUpdateGlobalFlagNoInstall(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Global flag always passes through to installer layer (for legacy install detection).
	scopes, err := resolveScopeForUpdate(ctx, false, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateNoFlagsGlobalOnly(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateNoFlagsProjectOnly(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scopes, err := resolveScopeForUpdate(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeProject}, scopes)
}

func TestResolveScopeForUpdateNoFlagsBothNonInteractive(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUpdate(ctx, false, false, globalDir, projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skills are installed in both global and project scopes")
	assert.Contains(t, err.Error(), "--global")
	assert.Contains(t, err.Error(), "--project")
}

func TestResolveScopeForUpdateNoFlagsBothInteractive(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
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

	scopes, err := resolveScopeForUpdate(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.True(t, promptCalled)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUpdateNoFlagsNeitherInstalled(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Falls through to global so the installer layer can handle legacy installs.
	scopes, err := resolveScopeForUpdate(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

// --- resolveScopeForUninstall tests ---

func TestResolveScopeForUninstallBothFlagsError(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUninstall(ctx, true, true, globalDir, projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot uninstall both scopes at once")
}

func TestResolveScopeForUninstallProjectFlagWithState(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, true, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeProject, scope)
}

func TestResolveScopeForUninstallGlobalFlagWithState(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, false, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUninstallProjectFlagNoInstall(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUninstall(ctx, true, false, globalDir, projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no project-scoped skills found")
	assert.Contains(t, err.Error(), "install --project")
	assert.Contains(t, err.Error(), "Expected location:")
}

func TestResolveScopeForUninstallGlobalFlagNoInstall(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Global flag always passes through to installer layer (for legacy install detection).
	scope, err := resolveScopeForUninstall(ctx, false, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUninstallNoFlagsGlobalOnly(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUninstallNoFlagsProjectOnly(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	scope, err := resolveScopeForUninstall(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeProject, scope)
}

func TestResolveScopeForUninstallNoFlagsBothNonInteractive(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	_, err := resolveScopeForUninstall(ctx, false, false, globalDir, projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skills are installed in both global and project scopes")
	assert.Contains(t, err.Error(), "--global")
	assert.Contains(t, err.Error(), "--project")
}

func TestResolveScopeForUninstallNoFlagsBothInteractive(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
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

	scope, err := resolveScopeForUninstall(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.True(t, promptCalled)
	assert.Equal(t, installer.ScopeProject, scope)
}

func TestResolveScopeForUninstallNoFlagsNeitherInstalled(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	ctx := nonInteractiveCtx(t)

	// Falls through to global so the installer layer can handle legacy installs.
	scope, err := resolveScopeForUninstall(ctx, false, false, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

// --- legacy global install passthrough tests ---

func TestResolveScopeForUpdateGlobalFlagLegacyInstall(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	// Create skill directories on disk but no .state.json (legacy install).
	legacyDir := filepath.Join(globalDir, "some-skill")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	ctx := nonInteractiveCtx(t)

	// Global flag should pass through to installer layer, not block on missing state.
	scopes, err := resolveScopeForUpdate(ctx, false, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal}, scopes)
}

func TestResolveScopeForUninstallGlobalFlagLegacyInstall(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	// Create skill directories on disk but no .state.json (legacy install).
	legacyDir := filepath.Join(globalDir, "some-skill")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	ctx := nonInteractiveCtx(t)

	// Global flag should pass through to installer layer, not block on missing state.
	scope, err := resolveScopeForUninstall(ctx, false, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, installer.ScopeGlobal, scope)
}

func TestResolveScopeForUpdateBothFlagsLegacyGlobal(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	// Global has a legacy install (dirs on disk, no state), project has state.
	legacyDir := filepath.Join(globalDir, "some-skill")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	installProjectState(t, projectDir)
	ctx := nonInteractiveCtx(t)

	// Global passes through (for legacy detection), project has state so it's included.
	scopes, err := resolveScopeForUpdate(ctx, true, true, globalDir, projectDir)
	require.NoError(t, err)
	assert.Equal(t, []string{installer.ScopeGlobal, installer.ScopeProject}, scopes)
}

// --- uninstall cross-scope hint verb tests ---

func TestResolveScopeForUninstallProjectFlagHintsUninstall(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	ctx := nonInteractiveCtx(t)

	// Project flag with no project state should hint about global using "uninstall" verb.
	_, err := resolveScopeForUninstall(ctx, true, false, globalDir, projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without --project to uninstall those")
}

func TestResolveScopeForUpdateProjectFlagHintsUpdate(t *testing.T) {
	globalDir, projectDir := setupScopeTest(t)
	installGlobalState(t, globalDir)
	ctx := nonInteractiveCtx(t)

	// Project flag with no project state should hint about global using "update" verb.
	_, err := resolveScopeForUpdate(ctx, true, false, globalDir, projectDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without --project to update those")
}
