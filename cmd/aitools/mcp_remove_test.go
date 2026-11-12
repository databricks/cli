package aitools

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMCPRemoveFlags(t *testing.T) {
	names := []string{"github-logfood"}
	cases := []struct {
		name    string
		all     bool
		names   []string
		profile string
		want    string
	}{
		{"names", false, names, "", ""},
		{"profile", false, nil, "logfood", ""},
		{"all", true, nil, "", ""},
		{"all with profile", true, nil, "logfood", ""},
		{"nothing", false, nil, "", removeFlagsHint},
		{"all with names", true, names, "", "--all cannot be combined with --names"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMCPRemoveFlags(tc.all, tc.names, tc.profile)
			if tc.want == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.want, err.Error())
		})
	}

	err := validateMCPRemoveFlags(false, nil, "")
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
}

func TestSelectInstalls(t *testing.T) {
	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood"},
		{Name: "github-dogfood", Service: "system.ai.github", Profile: "dogfood"},
		{Name: "genie_one_mcp-logfood", Service: "system.ai.genie_one_mcp", Profile: "logfood"},
	}}

	all, err := selectInstalls(state, nil, "")
	require.NoError(t, err)
	assert.Len(t, all, 3)

	byProfile, err := selectInstalls(state, nil, "logfood")
	require.NoError(t, err)
	require.Len(t, byProfile, 2)
	assert.Equal(t, "github-logfood", byProfile[0].Name)

	// A service name matches every profile of that service, unless narrowed.
	byService, err := selectInstalls(state, []string{"SYSTEM.AI.GITHUB"}, "")
	require.NoError(t, err)
	assert.Len(t, byService, 2)

	byName, err := selectInstalls(state, []string{"github-dogfood"}, "")
	require.NoError(t, err)
	require.Len(t, byName, 1)
	assert.Equal(t, "dogfood", byName[0].Profile)

	_, err = selectInstalls(state, []string{"github-dogfood"}, "logfood")
	require.Error(t, err)
	assert.Equal(t, "no installed MCP server named github-dogfood on profile logfood", err.Error())
	assert.Equal(t, categoryServiceNotFound, errorCategory(err))

	// A profile with no installed server is a typo or an empty profile, not an empty removal.
	_, err = selectInstalls(state, nil, "typo")
	require.Error(t, err)
	assert.Equal(t, "no installed MCP servers on profile typo", err.Error())
	assert.Equal(t, categoryServiceNotFound, errorCategory(err))

	none, err := selectInstalls(&mcp.State{}, nil, "")
	require.NoError(t, err, "--all on an empty state is a no-op")
	assert.Empty(t, none)
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	return doc
}

// readRemoveOutput decodes the JSON document of `mcp remove`.
func readRemoveOutput(t *testing.T, stdout string) removeOutput {
	t.Helper()
	var out removeOutput
	require.NoError(t, json.Unmarshal([]byte(stdout), &out), stdout)
	return out
}

func TestMcpRemoveDeletesOwnedEntriesAndPrunesMissing(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	require.NoError(t, os.WriteFile(claudePath, []byte(`{"numStartups": 3, "mcpServers": {"web_search": {"command": "ug", "args": ["mcp-proxy"]}}}`), 0o600))

	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	// Recorded but never written: remove prunes it without touching the file.
	absent := mcp.Entry{
		Name: "dbsql-logfood", Service: "system.ai.dbsql", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand,
		Agents: map[string]mcp.AgentRecord{"claude-code": {Path: claudePath, Scope: "user"}},
	}
	other := seedInstall(t, claude, "github-dogfood", "system.ai.github", "dogfood", "https://adb-2.test")
	saveState(t, genie, absent, other)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood,dbsql-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	assert.Equal(t, []removeRemoved{
		{Name: "genie_one_mcp-logfood", Service: "system.ai.genie_one_mcp", Profile: "logfood", Agents: []string{"claude-code"}},
		{Name: "dbsql-logfood", Service: "system.ai.dbsql", Profile: "logfood", Agents: []string{"claude-code"}},
	}, out.Removed)
	assert.Empty(t, out.Skipped)

	doc := readJSONFile(t, claudePath)
	assert.EqualValues(t, 3, doc["numStartups"], "keys outside mcpServers are kept")
	servers := doc["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "web_search", "entries we did not write are kept")
	assert.Contains(t, servers, "github-dogfood", "other servers are kept")
	assert.NotContains(t, servers, "genie_one_mcp-logfood")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, "github-dogfood", state.Servers[0].Name)
}

func TestMcpRemoveSkipsEntryChangedOutsideCLI(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	// Someone points the same name at another command. It is no longer ours.
	repointEntry(t, claudePath, "genie_one_mcp-logfood", "/usr/bin/other")
	before, err := os.ReadFile(claudePath)
	require.NoError(t, err)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "genie_one_mcp-logfood", out.Skipped[0].Name)
	assert.Equal(t, "claude-code", out.Skipped[0].Agent)

	after, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the agent config is not written")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Contains(t, state.Servers[0].Agents, "claude-code", "the record stays, so list keeps showing the entry")
}

func TestMcpRemoveKeepsEntryWhenAgentConfigMoved(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	// The agent now reads its config from another directory. The entry is still in the file it was written to.
	moved := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", moved)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed, "a file the agent no longer reads is not pruned")
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "claude-code", out.Skipped[0].Agent)
	assert.Contains(t, out.Skipped[0].Reason, filepath.ToSlash(claudePath), "the reason names the file the entry was written to")
	assert.Contains(t, out.Skipped[0].Reason, filepath.ToSlash(filepath.Join(moved, ".claude.json")), "and the file the agent reads now")

	assert.Contains(t, readJSONFile(t, claudePath)["mcpServers"], "genie_one_mcp-logfood", "the entry in the old file is kept")
	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Contains(t, state.Servers[0].Agents, "claude-code", "the record is kept, so list can show the path change")
}

func TestMcpRemoveExplicitAgentFailureIsAnError(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	codex := agents.ByName("codex")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	codexPath := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
	require.NoError(t, os.WriteFile(codexPath, []byte("[mcp_servers.broken\n"), 0o600))
	entry.Agents[codex.Name] = mcp.AgentRecord{Path: codexPath, Scope: "user"}
	saveState(t, entry)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--agents", "claude-code,codex", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "codex", out.Skipped[0].Agent)
	assert.Contains(t, out.Skipped[0].Reason, "could not update Codex CLI")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, []string{"codex"}, slices.Sorted(maps.Keys(state.Servers[0].Agents)))
}

func TestMcpRemoveFailsWhenNoAgentWasRemoved(t *testing.T) {
	home := withHome(t)
	codex := agents.ByName("codex")
	codexPath := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
	require.NoError(t, os.WriteFile(codexPath, []byte("[mcp_servers.broken\n"), 0o600))
	saveState(t, mcp.Entry{
		Name: "genie_one_mcp-logfood", Service: "system.ai.genie_one_mcp", Profile: "logfood", Host: "https://adb-1.test",
		Command: testDatabricksCommand, Agents: map[string]mcp.AgentRecord{codex.Name: {Path: codexPath, Scope: "user"}},
	})

	// No --agents: the failure is only a warning when another agent was removed. Here none was.
	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "codex", out.Skipped[0].Agent)
}

func TestMcpRemoveRefusesProfileAndAgentWithNothingInstalled(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	stdout, _, err := runMcpCmd(t, "remove", "--profile", "typo", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	out := readRemoveOutput(t, stdout.String())
	assert.Equal(t, "no installed MCP servers on profile typo", out.Error)
	assert.Equal(t, categoryServiceNotFound, out.ErrorCategory)

	stdout, _, err = runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--agents", "cursor", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	out = readRemoveOutput(t, stdout.String())
	assert.Equal(t, "no selected MCP server is installed for Cursor", out.Error)
	assert.Equal(t, categoryServiceNotFound, out.ErrorCategory)

	// Nothing was removed by either refusal.
	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Contains(t, state.Servers[0].Agents, "claude-code")
}

func TestMcpRemoveReportsResultsWhenStateCannotBeSaved(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	// The agent entry is deleted first; then the state directory cannot take the new state file.
	stateDir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, os.Chmod(stateDir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(stateDir, 0o700) })

	stdout, stderr, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood")
	require.Error(t, err)
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "Removed genie_one_mcp-logfood", "the results are printed before the error")
	assert.NotContains(t, readJSONFile(t, claudePath)["mcpServers"], "genie_one_mcp-logfood")
}

func TestMcpRemoveDropsRecordOfNamedEntry(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	dir := t.TempDir()
	first := filepath.Join(dir, "first.cfg")
	second := filepath.Join(dir, "second.cfg")
	for _, path := range []string{first, second} {
		require.NoError(t, os.WriteFile(path, []byte("[logfood]\nhost = https://adb-1.test\ntoken = t\n"), 0o600))
	}

	// Two servers of one service and profile, each pinned to its own config file.
	kept := seedInstallEntry(t, claude, seed{"genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test", testDatabricksCommand, first})
	removed := seedInstallEntry(t, claude, seed{"genie_one_mcp-logfood-2", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test", testDatabricksCommand, second})
	saveState(t, kept, removed)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood-2", "--output", "json")
	require.NoError(t, err)
	assert.Len(t, readRemoveOutput(t, stdout.String()).Removed, 1)

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, "genie_one_mcp-logfood", state.Servers[0].Name)
	assert.Contains(t, state.Servers[0].Agents, "claude-code", "the record of the other server is kept")
}

func TestMcpRemoveLeavesEntriesWithoutRecord(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	// Written by this CLI, but the state that recorded it is gone. Remove does not know it, so it stays.
	seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")

	stdout, _, err := runMcpCmd(t, "remove", "--all", "--output", "json")
	require.NoError(t, err)
	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed)
	assert.Empty(t, out.Skipped)
	assert.Contains(t, readJSONFile(t, claudePath)["mcpServers"], "genie_one_mcp-logfood")
}

func TestMcpRemoveFailsWithoutSelection(t *testing.T) {
	withHome(t)
	stdout, _, err := runMcpCmd(t, "remove", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	out := readRemoveOutput(t, stdout.String())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, out.ErrorCategory)
	assert.Equal(t, removeFlagsHint, out.Error)
	assert.Empty(t, out.Removed)
	assert.Empty(t, out.Skipped)
}
