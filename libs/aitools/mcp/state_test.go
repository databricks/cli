package mcp_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const stateTestFileRel = ".databricks/aitools/mcp/state.json"

// stateTestCtx returns a context whose home directory is a fresh temp dir, plus the
// path of the state file inside it.
func stateTestCtx(t *testing.T) (context.Context, string) {
	t.Helper()
	home := t.TempDir()
	return env.WithUserHomeDir(t.Context(), home), filepath.Join(home, stateTestFileRel)
}

func stateTestEntry() mcp.Entry {
	return mcp.Entry{
		Name:           "github-logfood",
		Service:        "system.ai.github",
		Profile:        "logfood",
		Host:           "https://adb-123.test",
		WorkspaceID:    "123",
		SignInRequired: true,
		Command:        "/usr/local/bin/databricks",
		Agents: map[string]mcp.AgentRecord{
			"claude-code": {Path: "/home/me/.claude.json", Scope: "user"},
		},
		InstalledAt: time.Date(2026, 11, 12, 10, 12, 0, 0, time.UTC),
		CLIVersion:  "0.300.0",
	}
}

func TestLoadStateMissingFileIsEmpty(t *testing.T) {
	ctx, _ := stateTestCtx(t)

	state, err := mcp.LoadState(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, state.SchemaVersion)
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

func TestSaveStateWritesSection10Shape(t *testing.T) {
	ctx, path := stateTestCtx(t)

	require.NoError(t, mcp.SaveState(ctx, &mcp.State{Servers: []mcp.Entry{stateTestEntry()}}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	want := `{
  "schema_version": 1,
  "servers": [
    {
      "name": "github-logfood",
      "service": "system.ai.github",
      "profile": "logfood",
      "host": "https://adb-123.test",
      "workspace_id": "123",
      "sign_in_required": true,
      "command": "/usr/local/bin/databricks",
      "config_file": "",
      "agents": {
        "claude-code": {
          "path": "/home/me/.claude.json",
          "scope": "user"
        }
      },
      "installed_at": "2026-11-12T10:12:00Z",
      "cli_version": "0.300.0"
    }
  ]
}
`
	assert.Equal(t, want, string(data))
}

func TestLoadStateMigratesMissingSchemaVersion(t *testing.T) {
	ctx, path := stateTestCtx(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"servers": []}`), 0o600))

	state, err := mcp.LoadState(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, state.SchemaVersion)
}

func TestLoadStateRejectsNewerSchemaVersion(t *testing.T) {
	ctx, path := stateTestCtx(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version": 2, "servers": []}`), 0o600))

	_, err := mcp.LoadState(ctx)
	require.ErrorContains(t, err, "schema_version 2")
}

func TestLoadStateRejectsInvalidJSON(t *testing.T) {
	ctx, path := stateTestCtx(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{not json`), 0o600))

	_, err := mcp.LoadState(ctx)
	require.ErrorContains(t, err, "failed to parse MCP state file")
}

// stateTestProfiles is the profiles file used by the drift tests.
func stateTestProfiles() profile.InMemoryProfiler {
	return profile.InMemoryProfiler{Profiles: profile.Profiles{
		{Name: "logfood", Host: "https://adb-123.test"},
		{Name: "dogfood", Host: "https://adb-456.test"},
	}}
}

func TestReconcileReportsNoDriftForMatchingHost(t *testing.T) {
	entry := stateTestEntry()
	entry.Host = "https://adb-123.test/"
	state := &mcp.State{Servers: []mcp.Entry{entry}}

	drifts, err := mcp.Reconcile(t.Context(), stateTestProfiles(), state)
	require.NoError(t, err)
	assert.Empty(t, drifts)
}

func TestReconcileReportsProfileMissing(t *testing.T) {
	entry := stateTestEntry()
	entry.Profile = "gone"
	state := &mcp.State{Servers: []mcp.Entry{entry}}

	drifts, err := mcp.Reconcile(t.Context(), stateTestProfiles(), state)
	require.NoError(t, err)
	assert.Equal(t, []mcp.Drift{{Kind: mcp.DriftProfileMissing, Entry: entry}}, drifts)
}

func TestReconcileReportsHostChanged(t *testing.T) {
	entry := stateTestEntry()
	entry.Profile = "dogfood"
	entry.Host = "https://adb-123.test"
	state := &mcp.State{Servers: []mcp.Entry{entry}}

	drifts, err := mcp.Reconcile(t.Context(), stateTestProfiles(), state)
	require.NoError(t, err)
	assert.Equal(t, []mcp.Drift{{Kind: mcp.DriftHostChanged, Entry: entry, CurrentHost: "https://adb-456.test"}}, drifts)
}

func TestReconcileWithoutConfigFileReportsEveryProfileMissing(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "absent"))
	entry := stateTestEntry()
	state := &mcp.State{Servers: []mcp.Entry{entry}}

	drifts, err := mcp.Reconcile(ctx, profile.FileProfilerImpl{}, state)
	require.NoError(t, err)
	assert.Equal(t, []mcp.Drift{{Kind: mcp.DriftProfileMissing, Entry: entry}}, drifts)
}

// TestNormalizeHostMatchesWithHost checks that NormalizeHost agrees with profile.WithHost.
func TestNormalizeHostMatchesWithHost(t *testing.T) {
	cases := [][2]string{
		{"https://adb-123.test", "adb-123.test"},
		{"https://adb-123.test/", "https://adb-123.test"},
		{"adb-123.test", "https://adb-123.test"},
		{"https://adb-123.test", "https://adb-456.test"},
		{"https://adb-123.test:443", "https://adb-123.test"},
	}
	for _, c := range cases {
		want := profile.WithHost(c[0])(profile.Profile{Host: c[1]})
		got := mcp.NormalizeHost(c[0]) == mcp.NormalizeHost(c[1])
		assert.Equal(t, want, got, "%q vs %q", c[0], c[1])
	}
}
