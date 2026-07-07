package aitools

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUpdateMock(t *testing.T) *[]installer.UpdateOptions {
	t.Helper()
	orig := updateSkillsFn
	t.Cleanup(func() { updateSkillsFn = orig })

	var calls []installer.UpdateOptions
	updateSkillsFn = func(_ context.Context, _ installer.ManifestSource, _ []*agents.Agent, opts installer.UpdateOptions) (*installer.UpdateResult, error) {
		calls = append(calls, opts)
		return &installer.UpdateResult{}, nil
	}
	return &calls
}

func resetUpdatePluginSeams(t *testing.T) {
	t.Helper()
	origUpdatePlugins := updatePluginsFn
	origInstallPlugin := updateInstallPluginForAgentFn
	origRecordPlugins := updateRecordPluginInstallsFn
	origCleanupLegacy := updateCleanupLegacyFn
	origHasManagedRawSkills := hasManagedRawSkillsForAgentFn
	t.Cleanup(func() {
		updatePluginsFn = origUpdatePlugins
		updateInstallPluginForAgentFn = origInstallPlugin
		updateRecordPluginInstallsFn = origRecordPlugins
		updateCleanupLegacyFn = origCleanupLegacy
		hasManagedRawSkillsForAgentFn = origHasManagedRawSkills
	})
}

func TestUpdateCheckPrintsNoChanges(t *testing.T) {
	setupTestAgents(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	setupUpdateMock(t)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global", "--check"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, stderr.String(), "No changes.")
}

func TestUpdateCheckPrintsPluginState(t *testing.T) {
	setupTestAgents(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")

	origUpdateSkills := updateSkillsFn
	origUpdatePlugins := updatePluginsFn
	t.Cleanup(func() {
		updateSkillsFn = origUpdateSkills
		updatePluginsFn = origUpdatePlugins
	})
	updateSkillsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.UpdateOptions) (*installer.UpdateResult, error) {
		require.Fail(t, "pure plugin check should not update raw skills")
		return nil, nil
	}
	updatePluginsFn = func(context.Context, string, string) ([]installer.PluginUpdate, error) {
		require.Fail(t, "check mode must not run plugin update commands")
		return nil, nil
	}

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.6",
		Plugins: map[string]installer.PluginRecord{
			agents.NameClaudeCode: {Marketplace: "claude-plugins-official", Plugin: "databricks", Version: "0.2.6"},
		},
	}))

	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global", "--check"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, stderr.String(), "Claude Code  databricks plugin v0.2.6 (up to date)")
}

func TestUpdateMigratesManagedRawSkillsToPlugins(t *testing.T) {
	setupTestAgents(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	resetUpdatePluginSeams(t)

	var rawUpdateAgents []string
	origUpdateSkills := updateSkillsFn
	t.Cleanup(func() { updateSkillsFn = origUpdateSkills })
	updateSkillsFn = func(_ context.Context, _ installer.ManifestSource, targetAgents []*agents.Agent, _ installer.UpdateOptions) (*installer.UpdateResult, error) {
		for _, a := range targetAgents {
			rawUpdateAgents = append(rawUpdateAgents, a.Name)
		}
		return &installer.UpdateResult{}, nil
	}
	updatePluginsFn = func(context.Context, string, string) ([]installer.PluginUpdate, error) {
		return nil, nil
	}

	var installed []pluginCall
	updateInstallPluginForAgentFn = func(_ context.Context, a *agents.Agent, scope, ref string) (installer.PluginRecord, error) {
		installed = append(installed, pluginCall{agent: a.Name, scope: scope})
		return installer.PluginRecord{
			Marketplace: a.Plugin.Marketplace,
			Plugin:      a.Plugin.ID,
			Scope:       scope,
			Version:     installer.DisplaySkillsVersion(ref),
		}, nil
	}

	var cleaned []string
	updateCleanupLegacyFn = func(_ context.Context, a *agents.Agent, _ string) error {
		cleaned = append(cleaned, a.Name)
		return nil
	}
	updateRecordPluginInstallsFn = func(ctx context.Context, scope string, records map[string]installer.PluginRecord, ref string) error {
		return installer.RecordPluginInstalls(ctx, scope, records, ref)
	}
	hasManagedRawSkillsForAgentFn = func(_ context.Context, a *agents.Agent, _ string) (bool, error) {
		return a.Plugin != nil, nil
	}

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.5",
		Skills:        map[string]string{"databricks": "0.2.5"},
	}))

	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global"})

	require.NoError(t, cmd.Execute())

	assert.Equal(t, []pluginCall{
		{agent: agents.NameClaudeCode, scope: agentScopeUser},
		{agent: agents.NameCodex, scope: agentScopeUser},
		{agent: agents.NameCopilot, scope: agentScopeUser},
	}, installed)
	assert.Equal(t, []string{agents.NameClaudeCode, agents.NameCodex, agents.NameCopilot}, cleaned)
	assert.Equal(t, []string{agents.NameCursor}, rawUpdateAgents)
	assert.Contains(t, stderr.String(), "Installing databricks plugin for Claude Code...")
	assert.Contains(t, stderr.String(), "GitHub Copilot  databricks plugin v0.2.6")

	state, err := installer.LoadState(dir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Contains(t, state.Plugins, agents.NameClaudeCode)
	assert.Contains(t, state.Plugins, agents.NameCodex)
	assert.Contains(t, state.Plugins, agents.NameCopilot)
}

func TestUpdateKeepsRawSkillsWhenPluginMigrationStateWriteFails(t *testing.T) {
	setupTestAgents(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	resetUpdatePluginSeams(t)

	origUpdateSkills := updateSkillsFn
	t.Cleanup(func() { updateSkillsFn = origUpdateSkills })
	updateSkillsFn = func(context.Context, installer.ManifestSource, []*agents.Agent, installer.UpdateOptions) (*installer.UpdateResult, error) {
		require.Fail(t, "raw skill update should not continue after migration state write fails")
		return nil, nil
	}
	updatePluginsFn = func(context.Context, string, string) ([]installer.PluginUpdate, error) {
		return nil, nil
	}

	var installed []string
	updateInstallPluginForAgentFn = func(_ context.Context, a *agents.Agent, scope, ref string) (installer.PluginRecord, error) {
		installed = append(installed, a.Name)
		return installer.PluginRecord{
			Marketplace: a.Plugin.Marketplace,
			Plugin:      a.Plugin.ID,
			Scope:       scope,
			Version:     installer.DisplaySkillsVersion(ref),
		}, nil
	}

	var cleaned []string
	updateCleanupLegacyFn = func(_ context.Context, a *agents.Agent, _ string) error {
		cleaned = append(cleaned, a.Name)
		return nil
	}
	updateRecordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error {
		return errors.New("state write failed")
	}
	hasManagedRawSkillsForAgentFn = func(_ context.Context, a *agents.Agent, _ string) (bool, error) {
		return a.Plugin != nil, nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.5",
		Skills:        map[string]string{"databricks": "0.2.5"},
	}))

	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global"})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state write failed")
	assert.Equal(t, []string{agents.NameClaudeCode, agents.NameCodex, agents.NameCopilot}, installed)
	assert.Empty(t, cleaned)

	state, err := installer.LoadState(dir)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Empty(t, state.Plugins)
	assert.Equal(t, map[string]string{"databricks": "0.2.5"}, state.Skills)
}

func TestUpdateCheckPrintsPluginMigrationPlan(t *testing.T) {
	setupTestAgents(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	resetUpdatePluginSeams(t)
	setupUpdateMock(t)

	updatePluginsFn = func(context.Context, string, string) ([]installer.PluginUpdate, error) {
		require.Fail(t, "check mode must not run plugin update commands")
		return nil, nil
	}
	updateInstallPluginForAgentFn = func(context.Context, *agents.Agent, string, string) (installer.PluginRecord, error) {
		require.Fail(t, "check mode must not install plugins")
		return installer.PluginRecord{}, nil
	}
	updateRecordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error {
		require.Fail(t, "check mode must not record plugin installs")
		return nil
	}
	updateCleanupLegacyFn = func(context.Context, *agents.Agent, string) error {
		require.Fail(t, "check mode must not clean up legacy skills")
		return nil
	}
	hasManagedRawSkillsForAgentFn = func(_ context.Context, a *agents.Agent, _ string) (bool, error) {
		return a.Plugin != nil, nil
	}

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.5",
		Skills:        map[string]string{"databricks": "0.2.5"},
	}))

	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global", "--check"})

	require.NoError(t, cmd.Execute())
	assert.Contains(t, stderr.String(), "Claude Code  would migrate raw skills to databricks plugin")
	assert.Contains(t, stderr.String(), "Codex CLI  would migrate raw skills to databricks plugin")
	assert.Contains(t, stderr.String(), "GitHub Copilot  would migrate raw skills to databricks plugin")
	assert.NotContains(t, stderr.String(), "No changes.")
}

func TestUpdateDoesNotMigratePluginAgentWithoutManagedRawSkills(t *testing.T) {
	setupTestAgents(t)
	t.Setenv("DATABRICKS_SKILLS_REF", "v0.2.6")
	resetUpdatePluginSeams(t)

	var rawUpdateAgents []string
	origUpdateSkills := updateSkillsFn
	t.Cleanup(func() { updateSkillsFn = origUpdateSkills })
	updateSkillsFn = func(_ context.Context, _ installer.ManifestSource, targetAgents []*agents.Agent, _ installer.UpdateOptions) (*installer.UpdateResult, error) {
		for _, a := range targetAgents {
			rawUpdateAgents = append(rawUpdateAgents, a.Name)
		}
		return &installer.UpdateResult{}, nil
	}
	updatePluginsFn = func(context.Context, string, string) ([]installer.PluginUpdate, error) {
		return nil, nil
	}
	updateInstallPluginForAgentFn = func(context.Context, *agents.Agent, string, string) (installer.PluginRecord, error) {
		require.Fail(t, "must not migrate without managed raw skills")
		return installer.PluginRecord{}, nil
	}
	updateRecordPluginInstallsFn = func(context.Context, string, map[string]installer.PluginRecord, string) error {
		require.Fail(t, "must not record plugin installs without migration")
		return nil
	}
	updateCleanupLegacyFn = func(context.Context, *agents.Agent, string) error {
		require.Fail(t, "must not clean up without migration")
		return nil
	}
	hasManagedRawSkillsForAgentFn = func(context.Context, *agents.Agent, string) (bool, error) {
		return false, nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Release:       "v0.2.5",
		Skills:        map[string]string{"databricks": "0.2.5"},
	}))

	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, []string{agents.NameClaudeCode, agents.NameCursor}, rawUpdateAgents)
}

func TestUpdateNoPruneFlag(t *testing.T) {
	setupTestAgents(t)
	calls := setupUpdateMock(t)

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewUpdateCmd()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--scope", "global", "--no-prune"})

	require.NoError(t, cmd.Execute())
	require.Len(t, *calls, 1)
	assert.True(t, (*calls)[0].NoPrune)
}

func TestUpdateScopeFlag(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantScopes []string
		wantErr    string
	}{
		// scope=project requires installed project state and is covered via TestParseScopeFlag
		// (cmd/aitools/scope_test.go) and TestResolveScopeForUpdateProjectFlagWithState. Here
		// we cover the no-state paths and the failure modes specific to the Cobra wiring.
		{name: "scope global", args: []string{"--scope", "global"}, wantScopes: []string{installer.ScopeGlobal}},
		{name: "scope both with no installs falls through to global", args: []string{"--scope", "both"}, wantScopes: []string{installer.ScopeGlobal}},
		{name: "scope invalid value", args: []string{"--scope", "all"}, wantErr: `invalid --scope "all"`},
		{name: "scope conflicts with legacy project", args: []string{"--scope", "global", "--project"}, wantErr: "cannot use --scope with --project or --global"},
		// Legacy `--project --global` is the supported "update both scopes" path
		// (preserved until the deprecated flags are removed). Without state, it
		// falls through to global per TestResolveScopeForUpdateBothFlagsNeitherInstalled.
		{name: "legacy both flags fall through to global without state", args: []string{"--project", "--global"}, wantScopes: []string{installer.ScopeGlobal}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestAgents(t)
			calls := setupUpdateMock(t)

			ctx := cmdio.MockDiscard(t.Context())
			cmd := NewUpdateCmd()
			cmd.SetContext(ctx)
			cmd.SetArgs(tt.args)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			err := cmd.Execute()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, *calls, len(tt.wantScopes))
			for i, scope := range tt.wantScopes {
				assert.Equal(t, scope, (*calls)[i].Scope)
			}
		})
	}
}
