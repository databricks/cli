package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	stateTestFileRel      = ".databricks/aitools/skills/.state.json"
	stateTestWorkspaceID  = "123"
	stateTestOtherProfile = "dogfood"
	stateTestOtherHost    = "https://adb-456.test"
)

// stateTestCtx returns a context whose home directory is a fresh temp dir, plus the path of
// the aitools state file inside it.
func stateTestCtx(t *testing.T) (context.Context, string) {
	t.Helper()
	home := t.TempDir()
	return env.WithUserHomeDir(t.Context(), home), filepath.Join(home, stateTestFileRel)
}

func stateTestEntry() mcp.Entry {
	return mcp.Entry{
		Name:           "github-logfood",
		Service:        bindingTestService,
		Profile:        bindingTestProfile,
		Host:           bindingTestHost,
		WorkspaceID:    stateTestWorkspaceID,
		SignInRequired: true,
		Command:        "/usr/local/bin/databricks",
		Agents: map[string]mcp.AgentRecord{
			"claude-code": {Path: "/home/me/.claude.json", Scope: "user"},
		},
		InstalledAt: time.Date(2026, 11, 12, 10, 12, 0, 0, time.UTC),
		CLIVersion:  "0.300.0",
	}
}

// stateTestEntryWith returns the test entry with its profile, host and workspace replaced.
func stateTestEntryWith(profileName, host, workspaceID string) mcp.Entry {
	entry := stateTestEntry()
	entry.Profile = profileName
	entry.Host = host
	entry.WorkspaceID = workspaceID
	return entry
}

// writeStateFile writes raw JSON to the aitools state file at path.
func writeStateFile(t *testing.T, path, raw string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))
}

func TestLoadStateMissingFileIsEmpty(t *testing.T) {
	ctx, _ := stateTestCtx(t)

	state, err := mcp.LoadState(ctx)
	require.NoError(t, err)
	assert.Empty(t, state.Servers)
}

func TestSaveStateThenLoadStateRoundTrips(t *testing.T) {
	ctx, _ := stateTestCtx(t)

	saved := &mcp.State{Servers: []mcp.Entry{stateTestEntry()}}
	require.NoError(t, mcp.SaveState(ctx, saved))

	loaded, err := mcp.LoadState(ctx)
	require.NoError(t, err)
	assert.Equal(t, saved, loaded)
}

// TestSaveStateKeepsInstallerRecords checks that an MCP save changes only the MCP section of the
// shared file.
func TestSaveStateKeepsInstallerRecords(t *testing.T) {
	ctx, _ := stateTestCtx(t)
	dir, err := installer.GlobalSkillsDir(ctx)
	require.NoError(t, err)
	original := &installer.InstallState{
		SchemaVersion: installer.CurrentSchemaVersion,
		Release:       "v0.2.0",
		LastUpdated:   time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC),
		Skills:        map[string]string{"databricks": "1.0.0"},
		RepoDirs:      map[string]string{"databricks": "skills"},
		Scope:         installer.ScopeGlobal,
		Plugins: map[string]installer.PluginRecord{
			"claude-code": {Marketplace: "databricks", Plugin: "databricks", Scope: "user", Version: "1.0.0"},
		},
		Files: map[string]installer.FileRecord{
			"databricks/SKILL.md": {SHA256: "abc123", Origin: "skills"},
		},
	}
	require.NoError(t, installer.SaveState(dir, original))

	require.NoError(t, mcp.SaveState(ctx, &mcp.State{Servers: []mcp.Entry{stateTestEntry()}}))

	loaded, err := installer.LoadState(dir)
	require.NoError(t, err)
	original.MCP = &installer.MCPState{Servers: []installer.MCPEntry{stateTestEntry()}}
	assert.Equal(t, original, loaded)
}

// TestSaveStateUpgradesV2FileToV3 checks that the first MCP save on a v2 file writes schema v3.
func TestSaveStateUpgradesV2FileToV3(t *testing.T) {
	ctx, path := stateTestCtx(t)
	writeStateFile(t, path, `{"schema_version": 2, "release": "v0.2.0", "skills": {"databricks": "1.0.0"}}`)

	require.NoError(t, mcp.SaveState(ctx, &mcp.State{Servers: []mcp.Entry{stateTestEntry()}}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.EqualValues(t, 3, raw["schema_version"])
	assert.Equal(t, map[string]any{"databricks": "1.0.0"}, raw["skills"])
	assert.Contains(t, raw, "mcp")
}

// stateTestProfiles is the profiles file used by the drift tests. The profiles named
// spog-query, spog-none, spog-both and spog-abc name their workspace in the host query instead of
// workspace_id.
func stateTestProfiles() profile.InMemoryProfiler {
	return profile.InMemoryProfiler{Profiles: profile.Profiles{
		{Name: bindingTestProfile, Host: bindingTestHost, WorkspaceID: stateTestWorkspaceID},
		{Name: stateTestOtherProfile, Host: stateTestOtherHost, WorkspaceID: "456"},
		{Name: "spog-query", Host: "https://spog.test/?o=222"},
		{Name: "spog-none", Host: "https://spog.test/?w=none"},
		{Name: "spog-both", Host: "https://spog.test/?o=111&w=222"},
		{Name: "spog-abc", Host: "https://spog.test/?o=abc&w=222"},
	}}
}

func TestReconcile(t *testing.T) {
	ctx, _ := stateTestCtx(t)
	tests := []struct {
		name            string
		entry           mcp.Entry
		wantKind        mcp.DriftKind
		wantHost        string
		wantWorkspaceID string
	}{
		{
			name:  "matching host and workspace",
			entry: stateTestEntryWith(bindingTestProfile, "https://adb-123.test/", stateTestWorkspaceID),
		},
		{
			name:  "host differs only in case",
			entry: stateTestEntryWith(bindingTestProfile, "https://ADB-123.test", stateTestWorkspaceID),
		},
		{
			name:  "workspace not compared when the entry names none",
			entry: stateTestEntryWith(bindingTestProfile, bindingTestHost, "none"),
		},
		{
			name:  "workspace from host query matches the entry",
			entry: stateTestEntryWith("spog-query", "https://spog.test", "222"),
		},
		{
			name:            "workspace from host query differs from the entry",
			entry:           stateTestEntryWith("spog-query", "https://spog.test", stateTestWorkspaceID),
			wantKind:        mcp.DriftHostChanged,
			wantHost:        "https://spog.test/?o=222",
			wantWorkspaceID: "222",
		},
		{
			name:  "host query naming none is not compared",
			entry: stateTestEntryWith("spog-none", "https://spog.test", stateTestWorkspaceID),
		},
		{
			name:  "w in the host query wins over o",
			entry: stateTestEntryWith("spog-both", "https://spog.test", "222"),
		},
		{
			name:            "w in the host query differs from the o the entry recorded",
			entry:           stateTestEntryWith("spog-both", "https://spog.test", "111"),
			wantKind:        mcp.DriftHostChanged,
			wantHost:        "https://spog.test/?o=111&w=222",
			wantWorkspaceID: "222",
		},
		{
			name:  "non-numeric o is ignored when w names the workspace",
			entry: stateTestEntryWith("spog-abc", "https://spog.test", "222"),
		},
		{
			name:     "profile missing",
			entry:    stateTestEntryWith("gone", bindingTestHost, stateTestWorkspaceID),
			wantKind: mcp.DriftProfileMissing,
		},
		{
			name:            "host changed",
			entry:           stateTestEntryWith(stateTestOtherProfile, bindingTestHost, stateTestWorkspaceID),
			wantKind:        mcp.DriftHostChanged,
			wantHost:        stateTestOtherHost,
			wantWorkspaceID: "456",
		},
		{
			name:            "workspace changed on the same host",
			entry:           stateTestEntryWith(bindingTestProfile, bindingTestHost, "999"),
			wantKind:        mcp.DriftHostChanged,
			wantHost:        bindingTestHost,
			wantWorkspaceID: stateTestWorkspaceID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &mcp.State{Servers: []mcp.Entry{tt.entry}}

			drifts, err := mcp.Reconcile(ctx, stateTestProfiles(), state)
			require.NoError(t, err)
			if tt.wantKind == "" {
				assert.Empty(t, drifts)
				return
			}
			assert.Equal(t, []mcp.Drift{{
				Kind:               tt.wantKind,
				Entry:              tt.entry,
				CurrentHost:        tt.wantHost,
				CurrentWorkspaceID: tt.wantWorkspaceID,
			}}, drifts)
		})
	}
}

func TestReconcileWithoutConfigFileReportsEveryProfileMissing(t *testing.T) {
	ctx, _ := stateTestCtx(t)
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "absent"))
	entry := stateTestEntry()
	state := &mcp.State{Servers: []mcp.Entry{entry}}

	drifts, err := mcp.Reconcile(ctx, profile.FileProfilerImpl{}, state)
	require.NoError(t, err)
	assert.Equal(t, []mcp.Drift{{Kind: mcp.DriftProfileMissing, Entry: entry}}, drifts)
}

// TestReconcileChecksEntryAgainstPinnedConfigFile checks that an entry pinned to a config file
// is checked against that file, not the one the environment names.
func TestReconcileChecksEntryAgainstPinnedConfigFile(t *testing.T) {
	pinned := bindingTestWriteConfig(t, bindingTestProfileContents(bindingTestToken))
	ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "absent"))
	entry := stateTestEntry()
	entry.ConfigFile = pinned
	state := &mcp.State{Servers: []mcp.Entry{entry}}

	drifts, err := mcp.Reconcile(ctx, profile.FileProfilerImpl{}, state)
	require.NoError(t, err)
	assert.Empty(t, drifts)
}

// TestReconcileUnpinnedEntryChecksDefaultConfigFile checks that an entry with no pinned config file
// is checked against ~/.databrickscfg, even when DATABRICKS_CONFIG_FILE names another file.
func TestReconcileUnpinnedEntryChecksDefaultConfigFile(t *testing.T) {
	ctx, _ := stateTestCtx(t)
	home, err := env.UserHomeDir(ctx)
	require.NoError(t, err)
	writeStateFile(t, filepath.Join(home, ".databrickscfg"), "[logfood]\nhost = https://adb-456.test\n")
	ambient := bindingTestWriteConfig(t, bindingTestProfileContents(bindingTestToken))
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", ambient)
	state := &mcp.State{Servers: []mcp.Entry{stateTestEntry()}}

	drifts, err := mcp.Reconcile(ctx, profile.FileProfilerImpl{}, state)
	require.NoError(t, err)
	assert.Equal(t, []mcp.Drift{{
		Kind:               mcp.DriftHostChanged,
		Entry:              stateTestEntry(),
		CurrentHost:        "https://adb-456.test",
		CurrentWorkspaceID: "",
	}}, drifts)
}

func TestNormalizeHostIgnoresSchemeCaseAndTrailingSlash(t *testing.T) {
	assert.Equal(t, mcp.NormalizeHost("https://ADB-123.test/"), mcp.NormalizeHost("adb-123.test"))
	assert.NotEqual(t, mcp.NormalizeHost("https://adb-123.test"), mcp.NormalizeHost("https://adb-456.test"))
}

// TestReconcileProfileWithoutHostIsMissing checks that a profile section with no host is reported as
// missing, since it names no host to compare. It reads a real file, because the in-memory profiler
// does not model sections without a host.
func TestReconcileProfileWithoutHostIsMissing(t *testing.T) {
	ctx, _ := stateTestCtx(t)
	path := bindingTestWriteConfig(t, "[logfood]\nauth_type = pat\ntoken = dapi-profile-token\n")
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", path)
	entry := stateTestEntry()
	state := &mcp.State{Servers: []mcp.Entry{entry}}

	drifts, err := mcp.Reconcile(ctx, profile.FileProfilerImpl{}, state)
	require.NoError(t, err)
	assert.Equal(t, []mcp.Drift{{Kind: mcp.DriftProfileMissing, Entry: entry}}, drifts)
}

// TestReconcileReportsEntryErrorAndKeepsOthers checks that an entry that cannot be checked, here a "~"
// path with no home directory, is reported as an error without stopping the check of the other entries.
func TestReconcileReportsEntryErrorAndKeepsOthers(t *testing.T) {
	ctx, _ := stateTestCtx(t)
	ctx = env.WithUserHomeDir(ctx, "")
	unresolvable := stateTestEntry()
	unresolvable.Name = "github-home"
	unresolvable.ConfigFile = "~/.databrickscfg"
	changed := stateTestEntryWith(bindingTestProfile, "https://adb-999.test", stateTestWorkspaceID)
	changed.ConfigFile = filepath.Join(t.TempDir(), ".databrickscfg")
	state := &mcp.State{Servers: []mcp.Entry{unresolvable, changed}}

	drifts, err := mcp.Reconcile(ctx, stateTestProfiles(), state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MCP server github-home: cannot find homedir: $HOME is not set")
	assert.Equal(t, []mcp.Drift{{
		Kind:               mcp.DriftHostChanged,
		Entry:              changed,
		CurrentHost:        bindingTestHost,
		CurrentWorkspaceID: stateTestWorkspaceID,
	}}, drifts)
}
