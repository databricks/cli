package aitools

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCommandExists(t *testing.T) {
	cmd := NewListCmd()
	assert.Equal(t, "list", cmd.Use)
}

func TestListCommandCallsListFn(t *testing.T) {
	orig := listSkillsFn
	t.Cleanup(func() { listSkillsFn = orig })

	called := false
	listSkillsFn = func(cmd *cobra.Command, scope string) error {
		called = true
		return nil
	}

	ctx := cmdio.MockDiscard(t.Context())
	cmd := NewListCmd()
	cmd.SetContext(ctx)

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestListCommandHasScopeFlags(t *testing.T) {
	cmd := NewListCmd()
	f := cmd.Flags().Lookup("project")
	require.NotNil(t, f, "--project flag should exist (deprecated alias)")
	assert.NotEmpty(t, f.Deprecated, "--project should be marked deprecated")
	f = cmd.Flags().Lookup("global")
	require.NotNil(t, f, "--global flag should exist (deprecated alias)")
	assert.NotEmpty(t, f.Deprecated, "--global should be marked deprecated")
	f = cmd.Flags().Lookup("scope")
	require.NotNil(t, f, "--scope flag should exist")
}

func TestRenderListJSON(t *testing.T) {
	out := listOutput{
		Release: "0.1.0",
		Skills: []skillEntry{
			{
				Name:          "databricks-jobs",
				LatestVersion: "1.0.0",
				Experimental:  false,
				Installed: map[string]string{
					installer.ScopeGlobal:  "1.0.0",
					installer.ScopeProject: "0.9.0",
				},
			},
			{
				Name:          "experimental-thing",
				LatestVersion: "0.1.0",
				Experimental:  true,
				Installed:     map[string]string{},
			},
		},
		Summary: map[string]scopeSummary{
			installer.ScopeGlobal:  {Installed: 1, Total: 2},
			installer.ScopeProject: {Installed: 1, Total: 2},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, renderListJSON(&buf, out))

	var got listOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, out, got)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &raw))
	assert.Contains(t, raw, "release")
	assert.Contains(t, raw, "skills")
	assert.Contains(t, raw, "summary")

	skills := raw["skills"].([]any)
	first := skills[0].(map[string]any)
	assert.Equal(t, "databricks-jobs", first["name"])
	assert.Equal(t, "1.0.0", first["latest_version"])
	assert.Equal(t, false, first["experimental"])

	installed := first["installed"].(map[string]any)
	assert.Equal(t, "1.0.0", installed["global"])
	assert.Equal(t, "0.9.0", installed["project"])

	second := skills[1].(map[string]any)
	assert.Equal(t, true, second["experimental"])
	assert.Empty(t, second["installed"])
}

func TestRenderListJSONWithAgents(t *testing.T) {
	out := listOutput{
		Release: "0.2.6",
		Skills:  []skillEntry{},
		Summary: map[string]scopeSummary{installer.ScopeGlobal: {Installed: 0, Total: 0}},
		Agents: []agentEntry{
			{Name: "claude-code", Plugin: &pluginInfo{Version: "0.2.6", Managed: true}, Status: statusUpToDate},
			{Name: "cursor", Plugin: &pluginInfo{Managed: false}, Status: statusManualAddPlugin},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, renderListJSON(&buf, out))

	var raw map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &raw))
	// Existing contract keys remain.
	assert.Contains(t, raw, "release")
	assert.Contains(t, raw, "skills")
	assert.Contains(t, raw, "summary")

	agentsRaw := raw["agents"].([]any)
	require.Len(t, agentsRaw, 2)
	first := agentsRaw[0].(map[string]any)
	assert.Equal(t, "claude-code", first["name"])
	assert.Equal(t, "up_to_date", first["status"])
	plugin := first["plugin"].(map[string]any)
	assert.Equal(t, "0.2.6", plugin["version"])
	assert.Equal(t, true, plugin["managed"])

	second := agentsRaw[1].(map[string]any)
	assert.Equal(t, "manual_add_plugin", second["status"])
	assert.Equal(t, false, second["plugin"].(map[string]any)["managed"])
}

func TestBuildAgentEntries(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755))
	ctx := cmdio.MockDiscard(t.Context())

	globalState := &installer.InstallState{
		Plugins: map[string]installer.PluginRecord{
			"claude-code": {Plugin: "databricks", Version: "0.2.6"},
			"codex":       {Plugin: "databricks", Version: "0.2.5"},
		},
	}

	entries := buildAgentEntries(ctx, "0.2.6", globalState, nil)
	byName := map[string]agentEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	require.Contains(t, byName, "claude-code")
	assert.Equal(t, statusUpToDate, byName["claude-code"].Status)
	assert.True(t, byName["claude-code"].Plugin.Managed)

	require.Contains(t, byName, "codex")
	assert.Equal(t, statusUpdateAvailable, byName["codex"].Status)

	// Cursor is detected (config dir) but never CLI-managed.
	require.Contains(t, byName, "cursor")
	assert.Equal(t, statusManualAddPlugin, byName["cursor"].Status)
	assert.False(t, byName["cursor"].Plugin.Managed)
}

func TestRenderListJSONScopeFiltersSummary(t *testing.T) {
	out := listOutput{
		Release: "0.1.0",
		Skills:  []skillEntry{},
		Summary: map[string]scopeSummary{
			installer.ScopeGlobal: {Installed: 0, Total: 5},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, renderListJSON(&buf, out))

	var raw map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &raw))
	summary := raw["summary"].(map[string]any)
	assert.Contains(t, summary, "global")
	assert.NotContains(t, summary, "project")
}

func TestInstalledStatusFromEntry(t *testing.T) {
	tests := []struct {
		name       string
		entry      skillEntry
		bothScopes bool
		want       string
	}{
		{
			name:  "not installed",
			entry: skillEntry{LatestVersion: "1.0.0", Installed: map[string]string{}},
			want:  "not installed",
		},
		{
			name: "global up to date",
			entry: skillEntry{
				LatestVersion: "1.0.0",
				Installed:     map[string]string{installer.ScopeGlobal: "1.0.0"},
			},
			want: "v1.0.0 (up to date)",
		},
		{
			name: "project update available",
			entry: skillEntry{
				LatestVersion: "1.0.0",
				Installed:     map[string]string{installer.ScopeProject: "0.9.0"},
			},
			want: "v0.9.0 (update available)",
		},
		{
			name: "both scopes installed",
			entry: skillEntry{
				LatestVersion: "1.0.0",
				Installed: map[string]string{
					installer.ScopeGlobal:  "1.0.0",
					installer.ScopeProject: "0.9.0",
				},
			},
			bothScopes: true,
			want:       "v0.9.0 (update available) (project, global)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, installedStatusFromEntry(tt.entry, tt.bothScopes))
		})
	}
}

func TestSummaryLinePreservesStatePresence(t *testing.T) {
	tests := []struct {
		name string
		out  listOutput
		want string
	}{
		{
			name: "both state files loaded even with no installs",
			out: listOutput{
				Skills: []skillEntry{
					{Name: "databricks-jobs", LatestVersion: "1.0.0", Installed: map[string]string{}},
				},
				Summary: map[string]scopeSummary{
					installer.ScopeGlobal:  {Installed: 0, Total: 1, loaded: true},
					installer.ScopeProject: {Installed: 0, Total: 1, loaded: true},
				},
			},
			want: "0/1 skills installed (global), 0/1 (project)",
		},
		{
			name: "only project state loaded",
			out: listOutput{
				Skills: []skillEntry{
					{Name: "databricks-jobs", LatestVersion: "1.0.0", Installed: map[string]string{}},
				},
				Summary: map[string]scopeSummary{
					installer.ScopeGlobal:  {Installed: 0, Total: 1},
					installer.ScopeProject: {Installed: 0, Total: 1, loaded: true},
				},
			},
			want: "0/1 skills installed (project)",
		},
		{
			name: "only global state loaded",
			out: listOutput{
				Skills: []skillEntry{
					{Name: "databricks-jobs", LatestVersion: "1.0.0", Installed: map[string]string{}},
				},
				Summary: map[string]scopeSummary{
					installer.ScopeGlobal:  {Installed: 0, Total: 1, loaded: true},
					installer.ScopeProject: {Installed: 0, Total: 1},
				},
			},
			want: "0/1 skills installed (global)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, summaryLine(tt.out, ""))
		})
	}
}

func TestRenderListTextUsesLoadedStateForScopeLabels(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	out := listOutput{
		Release: "0.1.0",
		Skills: []skillEntry{
			{
				Name:          "databricks-jobs",
				LatestVersion: "1.0.0",
				Installed: map[string]string{
					installer.ScopeGlobal: "1.0.0",
				},
			},
		},
		Summary: map[string]scopeSummary{
			installer.ScopeGlobal:  {Installed: 1, Total: 1, loaded: true},
			installer.ScopeProject: {Installed: 0, Total: 1, loaded: true},
		},
	}

	renderListText(ctx, out, "")

	got := stderr.String()
	assert.Contains(t, got, "v1.0.0 (up to date) (global)")
	assert.Contains(t, got, "1/1 skills installed (global), 0/1 (project)")
}

func TestListScopeFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantScope string
		wantErr   string
	}{
		{name: "scope project", args: []string{"--scope", "project"}, wantScope: installer.ScopeProject},
		{name: "scope global", args: []string{"--scope", "global"}, wantScope: installer.ScopeGlobal},
		{name: "scope both shows both", args: []string{"--scope", "both"}, wantScope: ""},
		{name: "scope invalid", args: []string{"--scope", "all"}, wantErr: `invalid --scope "all"`},
		{name: "legacy both flags together rejected", args: []string{"--project", "--global"}, wantErr: "cannot use --global and --project together"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := listSkillsFn
			t.Cleanup(func() { listSkillsFn = orig })

			var gotScope string
			called := false
			listSkillsFn = func(_ *cobra.Command, scope string) error {
				called = true
				gotScope = scope
				return nil
			}

			ctx := cmdio.MockDiscard(t.Context())
			cmd := NewListCmd()
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
			assert.True(t, called)
			assert.Equal(t, tt.wantScope, gotScope)
		})
	}
}
