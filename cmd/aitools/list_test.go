package aitools

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
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
			{
				Name:                 "claude-code",
				DisplayName:          "Claude Code",
				Managed:              true,
				Detected:             true,
				SupportsProjectScope: true,
				Installed:            map[string]installInfo{installer.ScopeGlobal: {Delivery: "plugin", Version: "0.2.6"}},
			},
			{
				Name:        "cursor",
				DisplayName: "Cursor",
				Managed:     false,
				Installed:   map[string]installInfo{},
			},
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
	assert.Equal(t, "Claude Code", first["display_name"])
	assert.Equal(t, true, first["managed"])
	assert.Equal(t, true, first["detected"])
	assert.Equal(t, true, first["supports_project_scope"])
	installed := first["installed"].(map[string]any)
	global := installed["global"].(map[string]any)
	assert.Equal(t, "0.2.6", global["version"])
	assert.Equal(t, "plugin", global["delivery"])

	// A not-installed agent emits "installed": {} rather than omitting the key,
	// matching the skills shape.
	second := agentsRaw[1].(map[string]any)
	assert.Contains(t, second, "installed")
	assert.Empty(t, second["installed"])
	assert.Equal(t, false, second["supports_project_scope"])
}

func TestBuildAgentEntries(t *testing.T) {
	// Isolate HOME so the skills-only on-disk detection doesn't pick up the
	// developer's real agent skills dirs.
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	globalState := &installer.InstallState{
		Plugins: map[string]installer.PluginRecord{
			"claude-code": {Plugin: "databricks", Version: "0.2.6"},
			"codex":       {Plugin: "databricks", Version: "0.2.5"},
		},
	}

	entries := buildAgentEntries(ctx, map[string]*installer.InstallState{
		installer.ScopeGlobal: globalState,
	})
	byName := map[string]agentEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	require.Contains(t, byName, "claude-code")
	assert.True(t, byName["claude-code"].Managed)
	assert.Equal(t, "Claude Code", byName["claude-code"].DisplayName)
	assert.Equal(t, "0.2.6", byName["claude-code"].Installed[installer.ScopeGlobal].Version)
	assert.Equal(t, "databricks plugin · v0.2.6 · up to date", agentStatusLabel(byName["claude-code"], "0.2.6"))
	// SupportsProjectScope is wired straight from the registry: claude-code allows
	// project scope, whereas the managed/skills-only global-only agents below do not.
	assert.True(t, byName["claude-code"].SupportsProjectScope)

	require.Contains(t, byName, "codex")
	assert.True(t, byName["codex"].Managed)
	assert.Equal(t, "0.2.5", byName["codex"].Installed[installer.ScopeGlobal].Version)
	assert.Equal(t, "databricks plugin · v0.2.5 · update available", agentStatusLabel(byName["codex"], "0.2.6"))
	assert.False(t, byName["codex"].SupportsProjectScope)

	// The JSON output carries an entry for every registry agent, including
	// skills-only agents like Cursor and managed agents with no recorded install.
	require.Contains(t, byName, "cursor")
	assert.False(t, byName["cursor"].Managed)
	assert.Empty(t, byName["cursor"].Installed)
	assert.False(t, byName["cursor"].SupportsProjectScope)

	require.Contains(t, byName, "copilot")
	assert.True(t, byName["copilot"].Managed)
	assert.Empty(t, byName["copilot"].Installed)
}

func TestBuildAgentEntriesReportsSkillsOnlyAgentFromDisk(t *testing.T) {
	// Skills-only agents (Plugin == nil) never get a plugin record; their skills
	// are symlinked into the agent's own skills dir. The install must still be
	// reported from disk, versioned by the scope's recorded release.
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)

	// On Windows, openCodeConfigDir checks APPDATA first; on other platforms,
	// it checks XDG_CONFIG_HOME. Override the platform-specific setting to route
	// config discovery to a temp directory so the test controls where the function
	// looks for skills, regardless of system defaults.
	configDir := filepath.Join(home, ".config")
	if runtime.GOOS == "windows" {
		ctx = env.Set(ctx, "APPDATA", configDir)
	} else {
		ctx = env.Set(ctx, "XDG_CONFIG_HOME", configDir)
	}

	// OpenCode is skills-only; its global skills dir is configDir/opencode/skills.
	// The CLI installs skills there as symlinks to the canonical store, so build
	// the symlink to exercise the real on-disk shape.
	canonical := filepath.Join(home, ".databricks", "aitools", "skills", "databricks-jobs")
	require.NoError(t, os.MkdirAll(canonical, 0o755))
	skillsDir := filepath.Join(configDir, "opencode", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	require.NoError(t, os.Symlink(canonical, filepath.Join(skillsDir, "databricks-jobs")))

	globalState := &installer.InstallState{Release: "0.2.6"}
	entries := buildAgentEntries(ctx, map[string]*installer.InstallState{
		installer.ScopeGlobal: globalState,
	})
	byName := map[string]agentEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	require.Contains(t, byName, "opencode")
	opencode := byName["opencode"]
	assert.False(t, opencode.Managed)
	assert.Equal(t, "0.2.6", opencode.Installed[installer.ScopeGlobal].Version)
	assert.Equal(t, "skills", opencode.Installed[installer.ScopeGlobal].Delivery)

	// A skills-only agent with nothing on disk stays empty.
	require.Contains(t, byName, "cursor")
	assert.Empty(t, byName["cursor"].Installed)
}

func TestBuildAgentEntriesReportsSkillsOnlyInstallForPluginAgent(t *testing.T) {
	// `install --skills-only` gives a plugin-capable agent skills on disk and no
	// plugin record, so the on-disk fallback must fire for it too rather than
	// reporting it as not installed.
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)

	canonical := filepath.Join(home, ".databricks", "aitools", "skills", "databricks-jobs")
	require.NoError(t, os.MkdirAll(canonical, 0o755))
	skillsDir := filepath.Join(home, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	require.NoError(t, os.Symlink(canonical, filepath.Join(skillsDir, "databricks-jobs")))

	entries := buildAgentEntries(ctx, map[string]*installer.InstallState{
		installer.ScopeGlobal: {Release: "0.2.6"},
	})
	byName := map[string]agentEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	require.Contains(t, byName, "claude-code")
	cc := byName["claude-code"]
	assert.True(t, cc.Managed)
	assert.Equal(t, "skills", cc.Installed[installer.ScopeGlobal].Delivery)
	assert.Equal(t, "0.2.6", cc.Installed[installer.ScopeGlobal].Version)

	// A skills install is not a plugin install, so the text view's "Plugin
	// installs:" section must not claim one.
	assert.False(t, cc.hasPluginInstall())
}

func TestBuildAgentEntriesPrefersPluginRecordOverSkillsOnDisk(t *testing.T) {
	// An agent can have both a plugin record and leftover skills on disk (e.g. a
	// --skills-only install later replaced by the plugin). The recorded plugin is
	// authoritative for the scope.
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)

	canonical := filepath.Join(home, ".databricks", "aitools", "skills", "databricks-jobs")
	require.NoError(t, os.MkdirAll(canonical, 0o755))
	skillsDir := filepath.Join(home, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	require.NoError(t, os.Symlink(canonical, filepath.Join(skillsDir, "databricks-jobs")))

	entries := buildAgentEntries(ctx, map[string]*installer.InstallState{
		installer.ScopeGlobal: {
			Release: "0.2.6",
			Plugins: map[string]installer.PluginRecord{
				"claude-code": {Plugin: "databricks", Version: "0.2.5", Scope: "user"},
			},
		},
	})
	byName := map[string]agentEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	cc := byName["claude-code"]
	assert.Equal(t, "plugin", cc.Installed[installer.ScopeGlobal].Delivery)
	assert.Equal(t, "0.2.5", cc.Installed[installer.ScopeGlobal].Version)
	assert.Equal(t, "user", cc.Installed[installer.ScopeGlobal].NativeScope)
}

func TestBuildAgentEntriesRecordsPerScopeVersions(t *testing.T) {
	// Same agent recorded in both scopes: global current, project stale. Both
	// versions are recorded; no scope is merged away.
	globalState := &installer.InstallState{Plugins: map[string]installer.PluginRecord{
		"claude-code": {Plugin: "databricks", Version: "0.2.6"},
	}}
	projectState := &installer.InstallState{Plugins: map[string]installer.PluginRecord{
		"claude-code": {Plugin: "databricks", Version: "0.2.5"},
	}}

	entries := buildAgentEntries(t.Context(), map[string]*installer.InstallState{
		installer.ScopeGlobal:  globalState,
		installer.ScopeProject: projectState,
	})
	byName := map[string]agentEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	require.Contains(t, byName, "claude-code")
	cc := byName["claude-code"]
	assert.True(t, cc.Managed)
	assert.Equal(t, "0.2.6", cc.Installed[installer.ScopeGlobal].Version)
	assert.Equal(t, "0.2.5", cc.Installed[installer.ScopeProject].Version)

	// The renderer collapses the scopes and surfaces the stale one, rather than
	// hiding it behind the up-to-date scope.
	assert.Equal(t, "databricks plugin · v0.2.5 · update available", agentStatusLabel(cc, "0.2.6"))
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
			want: "0/1 raw skill directories installed (global), 0/1 (project)",
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
			want: "0/1 raw skill directories installed (project)",
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
			want: "0/1 raw skill directories installed (global)",
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
	assert.Contains(t, got, "1/1 raw skill directories installed (global), 0/1 (project)")
}

func TestRenderListTextGroupsExperimental(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	out := listOutput{
		Release: "latest",
		Skills: []skillEntry{
			{Name: "databricks-jobs", LatestVersion: "1.0.0", Installed: map[string]string{}},
			{Name: "experimental-thing", LatestVersion: "0.1.0", Experimental: true, Installed: map[string]string{}},
		},
		Summary: map[string]scopeSummary{
			installer.ScopeGlobal: {Installed: 0, Total: 2, loaded: true},
		},
	}

	renderListText(ctx, out, installer.ScopeGlobal)

	got := stderr.String()
	availIdx := strings.Index(got, "Available raw skill directories")
	expIdx := strings.Index(got, "Experimental skills:")
	require.GreaterOrEqual(t, availIdx, 0, "available group header present")
	require.GreaterOrEqual(t, expIdx, 0, "experimental group header present")
	assert.Less(t, availIdx, expIdx, "available group comes before experimental group")
	// Stable skill sits in the first group; experimental skill sits under its own header.
	assert.Less(t, strings.Index(got, "databricks-jobs"), expIdx)
	assert.Less(t, expIdx, strings.Index(got, "experimental-thing"))
	// No inline tag now that they are grouped.
	assert.NotContains(t, got, "[experimental]")
}

func TestRenderListTextShowsPluginInstallsBeforeRawSkills(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	out := listOutput{
		Release: "0.2.6",
		Skills: []skillEntry{
			{Name: "databricks-jobs", LatestVersion: "1.0.0", Installed: map[string]string{}},
		},
		Summary: map[string]scopeSummary{
			installer.ScopeGlobal: {Installed: 0, Total: 1, loaded: true},
		},
		Agents: []agentEntry{
			{
				Name:        "claude-code",
				DisplayName: "Claude Code",
				Managed:     true,
				Detected:    true,
				Installed:   map[string]installInfo{installer.ScopeGlobal: {Delivery: "plugin", Version: "0.2.6", NativeScope: "user"}},
			},
			{
				Name:        "cursor",
				DisplayName: "Cursor",
				Managed:     false,
			},
		},
	}

	renderListText(ctx, out, installer.ScopeGlobal)

	got := stderr.String()
	pluginIdx := strings.Index(got, "Plugin installs:")
	rawIdx := strings.Index(got, "Available raw skill directories")
	require.GreaterOrEqual(t, pluginIdx, 0)
	require.GreaterOrEqual(t, rawIdx, 0)
	assert.Less(t, pluginIdx, rawIdx)
	assert.Contains(t, got, "Claude Code")
	assert.Contains(t, got, "databricks plugin · v0.2.6 · up to date")
	// Skills-only agents (and any without a recorded install) are JSON-only and
	// must not appear in the text "Plugin installs:" section.
	assert.NotContains(t, got, "Cursor")
	assert.Contains(t, got, "0/1 raw skill directories installed (global)")
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
