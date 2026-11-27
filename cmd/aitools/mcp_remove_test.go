package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
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
			err := validateMCPRemoveFlags(tc.all, tc.names, tc.profile, false)
			if tc.want == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.want, err.Error())
		})
	}

	// In a terminal, no choice opens the picker instead of failing.
	assert.NoError(t, validateMCPRemoveFlags(false, nil, "", true))

	err := validateMCPRemoveFlags(false, nil, "", false)
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
	assert.Equal(t, protos.AitoolsErrorCategoryMCPServiceNotFound, errorCategory(err))

	// A profile with nothing installed has nothing to remove, the same as --all on an empty state.
	none, err := selectInstalls(state, nil, "typo")
	require.NoError(t, err)
	assert.Empty(t, none)

	empty, err := selectInstalls(&mcp.State{}, nil, "")
	require.NoError(t, err, "--all on an empty state is a no-op")
	assert.Empty(t, empty)
}

func TestSelectInstallsLeafAndCaseRules(t *testing.T) {
	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood"},
		{Name: "github-dogfood", Service: "system.ai.github", Profile: "dogfood"},
		{Name: "github-team", Service: "team.ai.github", Profile: "logfood"},
	}}

	// Server names compare without case.
	byName, err := selectInstalls(state, []string{"GITHUB-LOGFOOD"}, "")
	require.NoError(t, err)
	require.Len(t, byName, 1)

	// A leaf is the last part of the service name. Here it names services in two schemas, so it is refused.
	_, err = selectInstalls(state, []string{"github"}, "")
	require.Error(t, err)
	assert.Equal(t, "github matches more than one MCP service (system.ai.github, team.ai.github); pass the full service name or a server name", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))

	// Narrowed to a profile where only one schema is installed, the leaf is unambiguous.
	byLeaf, err := selectInstalls(state, []string{"github"}, "dogfood")
	require.NoError(t, err)
	require.Len(t, byLeaf, 1)
	assert.Equal(t, "dogfood", byLeaf[0].Profile)

	// A name with a dot is a full service name, not a leaf.
	full, err := selectInstalls(state, []string{"team.ai.github"}, "")
	require.NoError(t, err)
	require.Len(t, full, 1)
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

// TestMcpRemoveMakesNoRequestWhenTelemetryIsOff checks that remove edits its agent config without a request to the
// profile's host. Nothing is uploaded when telemetry is off, so no auth is resolved for an upload.
func TestMcpRemoveMakesNoRequestWhenTelemetryIsOff(t *testing.T) {
	home := withHome(t)
	t.Setenv("DATABRICKS_CLI_DISABLE_TELEMETRY", "1")
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = oauth-m2m\nclient_id = id\nclient_secret = secret\n")
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", server.URL)
	saveState(t, genie)

	parent := newMCPTestRoot()
	ctx, _ := cmdio.NewTestContextWithStderr(telemetry.WithNewLogger(t.Context()))
	parent.SetOut(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "remove", "--names", "genie_one_mcp-logfood", "--output", "json"})
	require.NoError(t, parent.ExecuteContext(ctx))
	require.Empty(t, telemetry.BufferedEvents(ctx))
	_, uploads := telemetry.UploadConfig(ctx)
	require.False(t, uploads)
	assert.NotContains(t, jsonServers(t, claudePath), "genie_one_mcp-logfood")
}

// wantOtherEntryReason is the skipped reason for an entry named name, in path, that this CLI did not write and that is not a
// Databricks entry: it is left in place, and the record for the agent is dropped.
func wantOtherEntryReason(name, path string) string {
	return "Claude Code has an entry named " + name + " in " + filepath.ToSlash(path) +
		" that this CLI did not write for this server; left in place and not changed. The install record for Claude Code was dropped."
}

// TestMcpRemoveDropsRecordOfEntryThisCLIDidNotWrite checks that an entry this CLI did not write, which is not a Databricks
// entry, is left in place and untouched, and the record for its agent is dropped. Nothing of this CLI's is left in the
// agent config, so the run is not refused, whether it names the server or takes every server.
func TestMcpRemoveDropsRecordOfEntryThisCLIDidNotWrite(t *testing.T) {
	cases := []struct {
		name     string
		selector []string
	}{
		{name: "names", selector: []string{"--names", "genie_one_mcp-logfood"}},
		{name: "all", selector: []string{"--all"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := withHome(t)
			claude := agents.ByName("claude-code")
			claudePath := filepath.Join(home, ".claude.json")
			entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
			saveState(t, entry)
			// Someone points the same name at another command. It is no longer ours, and it is not a Databricks entry.
			repointEntry(t, claudePath, "genie_one_mcp-logfood", "/usr/bin/other")
			before, err := os.ReadFile(claudePath)
			require.NoError(t, err)

			stdout, _, err := runMcpCmd(t, append([]string{"remove"}, append(tc.selector, "--output", "json")...)...)
			require.NoError(t, err, "nothing of this CLI's is left in the agent config, so the run is not refused")

			out := readRemoveOutput(t, stdout.String())
			require.Len(t, out.Removed, 1)
			assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
			require.Len(t, out.Skipped, 1)
			assert.Equal(t, "claude-code", out.Skipped[0].Agent)
			assert.Equal(t, wantOtherEntryReason("genie_one_mcp-logfood", claudePath), out.Skipped[0].Reason)
			assert.NotContains(t, out.Skipped[0].Reason, "--force", "an entry this CLI did not write is not offered for replacement")
			assert.NotContains(t, out.Skipped[0].Reason, "delete", "an entry this CLI did not write is not offered for deletion")

			after, err := os.ReadFile(claudePath)
			require.NoError(t, err)
			assert.Equal(t, before, after, "the agent config is not written")

			state, err := mcp.LoadState(t.Context())
			require.NoError(t, err)
			assert.Empty(t, state.Servers, "the record went")
		})
	}
}

// TestMcpRemoveDropsRecordWhenMovedConfigHasOtherEntry checks that when the agent's config moved and the recorded file no
// longer holds the entry, a same-name entry this CLI did not write in the file the agent reads now is left alone, and the
// record for the agent is dropped without refusing the run.
func TestMcpRemoveDropsRecordWhenMovedConfigHasOtherEntry(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	// The recorded file no longer holds the entry.
	require.NoError(t, os.WriteFile(claudePath, []byte("{}"), 0o600))

	moved := t.TempDir()
	movedPath := filepath.Join(moved, ".claude.json")
	require.NoError(t, os.WriteFile(movedPath, []byte(`{"mcpServers":{"genie_one_mcp-logfood":{"command":"npx","args":["my-own-genie-mcp"]}}}`), 0o600))
	before, err := os.ReadFile(movedPath)
	require.NoError(t, err)
	t.Setenv("CLAUDE_CONFIG_DIR", moved)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err, "the other entry is not ours, so the run is not refused")

	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, wantOtherEntryReason("genie_one_mcp-logfood", movedPath), out.Skipped[0].Reason)

	after, err := os.ReadFile(movedPath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the user's entry in the file the agent reads now is not changed")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Empty(t, state.Servers, "the record went")
}

// TestMcpRemoveKeepsServerWhenMovedConfigHasChangedDatabricksEntry checks that a Databricks entry of the server's name that
// was changed outside this CLI, in the file the agent reads now, keeps the server as it does in the file the record names:
// the run is refused, nothing is written, and the record stays.
func TestMcpRemoveKeepsServerWhenMovedConfigHasChangedDatabricksEntry(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	// The moved file holds a copy of the entry that serves another profile now: it was changed outside this CLI.
	moved := t.TempDir()
	movedPath := filepath.Join(moved, ".claude.json")
	data, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(movedPath, data, 0o600))
	editEntry(t, movedPath, entry.Name, func(server map[string]any) { setServeFlag(t, server, "--profile", "other") })
	t.Setenv("CLAUDE_CONFIG_DIR", moved)
	recordedBefore, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	movedBefore, err := os.ReadFile(movedPath)
	require.NoError(t, err)

	stdout, _, err := runMcpCmd(t, "remove", "--names", entry.Name, "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted, "a Databricks entry changed outside this CLI keeps the server, and the run is refused")

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "claude-code", out.Skipped[0].Agent)
	assert.Contains(t, out.Skipped[0].Reason, "changed outside this CLI")
	assert.Contains(t, out.Skipped[0].Reason, "delete the entry from "+filepath.ToSlash(movedPath)+" by hand")
	assert.NotContains(t, out.Skipped[0].Reason, "--force")

	recordedAfter, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, recordedBefore, recordedAfter, "the recorded config is not written")
	movedAfter, err := os.ReadFile(movedPath)
	require.NoError(t, err)
	assert.Equal(t, movedBefore, movedAfter, "the agent's config is not written")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Contains(t, state.Servers[0].Agents, "claude-code", "the record stays")
}

// TestMcpRemoveRemovesEntryFromRecordedFileWhenAgentConfigMoved checks that an agent whose config moved loses its entry
// from the file the record names, the file this CLI wrote, and that the record goes with it.
func TestMcpRemoveRemovesEntryFromRecordedFileWhenAgentConfigMoved(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	// The agent now reads its config from another directory.
	moved := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", moved)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	assert.Empty(t, out.Skipped)
	assert.NotContains(t, jsonServers(t, claudePath), "genie_one_mcp-logfood")
	_, err = os.Stat(filepath.Join(moved, ".claude.json"))
	assert.ErrorIs(t, err, os.ErrNotExist, "the agent's current file is not written")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Empty(t, state.Servers, "the record went with the entry")
}

// TestMcpRemoveLeavesOtherSameNameEntryWhenAgentConfigMoved checks that when the agent's config moved, and the file it
// reads now holds an entry of the same name that this CLI did not write, remove takes its own entry out of the recorded
// file and drops the record. The other entry is left alone, and the skip is reported.
func TestMcpRemoveLeavesOtherSameNameEntryWhenAgentConfigMoved(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "github-logfood", "system.ai.github", "logfood", "https://adb-1.test")
	saveState(t, entry)

	// The user's own server has the same name, in the file the agent reads now.
	moved := t.TempDir()
	movedPath := filepath.Join(moved, ".claude.json")
	require.NoError(t, os.WriteFile(movedPath, []byte(`{"mcpServers":{"github-logfood":{"command":"npx","args":["my-own-github-mcp"]}}}`), 0o600))
	before, err := os.ReadFile(movedPath)
	require.NoError(t, err)
	t.Setenv("CLAUDE_CONFIG_DIR", moved)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "github-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "github-logfood", out.Skipped[0].Name)
	assert.Equal(t, "claude-code", out.Skipped[0].Agent)
	assert.Equal(t, wantOtherEntryReason("github-logfood", movedPath), out.Skipped[0].Reason)

	assert.NotContains(t, jsonServers(t, claudePath), "github-logfood", "our entry is removed from the file it was written to")
	after, err := os.ReadFile(movedPath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the user's entry in the file the agent reads now is not changed")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Empty(t, state.Servers, "the record went with our entry")
}

// TestMcpRemovePrunesWhenRecordedFileIsGone checks that a record whose file no longer exists is pruned, even when the
// agent's config moved to a directory that is not the recorded one.
func TestMcpRemovePrunesWhenRecordedFileIsGone(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	require.NoError(t, os.Remove(claudePath))
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	assert.Empty(t, out.Skipped)
	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Empty(t, state.Servers)
}

// TestMcpRemoveDropsRecordOfAgentThatCannotBeEdited checks that a record for an agent without MCP support is dropped and
// reported, not left to fail every remove --all. A record for an agent outside the registry is kept; see
// TestMcpRemoveKeepsRecordOfUnknownAgent.
func TestMcpRemoveDropsRecordOfAgentThatCannotBeEdited(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	retired := mcp.Entry{
		Name: "dbsql-logfood", Service: "system.ai.dbsql", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand,
		Agents: map[string]mcp.AgentRecord{"retired-agent": {Path: "/x/config.json", Scope: "user"}, "pi": {Path: "/x/pi.json", Scope: "user"}},
	}
	saveState(t, genie, retired)

	stdout, _, err := runMcpCmd(t, "remove", "--all", "--output", "json")
	require.NoError(t, err, "a record that cannot be edited is not a failure")

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Error)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "retired-agent", out.Skipped[0].Agent)
	require.Len(t, out.Removed, 2)
	assert.Equal(t, []string{"pi"}, out.Removed[1].Agents)
	assert.Equal(t, []string{"pi"}, out.Removed[1].RecordOnly, "only the record of the agent without MCP support was dropped")
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	assert.Empty(t, out.Removed[0].RecordOnly)

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1, "the unknown agent's record keeps its server in state")
	assert.Equal(t, []string{"retired-agent"}, slices.Collect(maps.Keys(state.Servers[0].Agents)))
}

// TestMcpRemoveDropsNamedEntryWithoutAgents checks that a named entry left with no agents is removed from state.
func TestMcpRemoveDropsNamedEntryWithoutAgents(t *testing.T) {
	withHome(t)
	saveState(t, mcp.Entry{Name: "dangling", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand})

	stdout, _, err := runMcpCmd(t, "remove", "--names", "dangling", "--output", "json")
	require.NoError(t, err)
	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, "dangling", out.Removed[0].Name)

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Empty(t, state.Servers)
}

// TestMcpRemoveKeepsUnselectedEntryWithoutAgents checks that removing one server does not drop another entry that
// has no agents: only the entries this run selected or emptied are dropped.
func TestMcpRemoveKeepsUnselectedEntryWithoutAgents(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, genie, mcp.Entry{Name: "keep-me", Service: "system.ai.dbsql", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand})

	_, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood")
	require.NoError(t, err)

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, "keep-me", state.Servers[0].Name)
}

// TestMcpRemoveNamesAcceptsLeafService checks that --names takes the leaf of a service name, as add does.
func TestMcpRemoveNamesAcceptsLeafService(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	github := seedInstall(t, claude, "github-logfood", "system.ai.github", "logfood", "https://adb-1.test")
	saveState(t, github)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "github", "--output", "json")
	require.NoError(t, err)
	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, "github-logfood", out.Removed[0].Name)
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

// TestMcpRemoveNamesAgentsForReadOnlyConfig checks that an entry in a config file the user made read-only is kept. The
// skip names --agents as the way out only when --agents named the agent, as add does. The file keeps its bytes.
func TestMcpRemoveNamesAgentsForReadOnlyConfig(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes read-only files")
	}
	cases := []struct {
		name string
		args []string
		hint string
	}{
		{name: "agent named with --agents", args: []string{"--agents", "claude-code"}, hint: "is read-only; make it writable or leave claude-code out of --agents: "},
		{name: "agent not named", args: nil, hint: "is read-only; make it writable: "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := withHome(t)
			claude := agents.ByName("claude-code")
			claudePath := filepath.Join(home, ".claude.json")
			entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
			// Windows maps the mode to the read-only attribute. Writable again before the home directory is removed.
			t.Cleanup(func() { _ = os.Chmod(claudePath, 0o600) })
			require.NoError(t, os.Chmod(claudePath, 0o400))
			before, err := os.ReadFile(claudePath)
			require.NoError(t, err)
			saveState(t, entry)

			args := append([]string{"remove", "--names", "genie_one_mcp-logfood"}, tc.args...)
			stdout, _, err := runMcpCmd(t, append(args, "--output", "json")...)
			require.ErrorIs(t, err, root.ErrAlreadyPrinted)

			out := readRemoveOutput(t, stdout.String())
			require.Len(t, out.Skipped, 1)
			assert.Contains(t, out.Skipped[0].Reason, tc.hint)
			got, err := os.ReadFile(claudePath)
			require.NoError(t, err)
			assert.Equal(t, before, got)
		})
	}
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

// TestMcpRemoveEmptyProfileIsNoOp checks that removing from a profile with nothing installed exits 0 and removes nothing,
// so that the command can be run again safely.
func TestMcpRemoveEmptyProfileIsNoOp(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	stdout, _, err := runMcpCmd(t, "remove", "--profile", "typo", "--output", "json")
	require.NoError(t, err)
	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Error)
	assert.Empty(t, out.Removed)
	assert.Empty(t, out.Skipped)

	_, stderr, err := runMcpCmd(t, "remove", "--profile", "typo")
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "No Databricks MCP servers to remove.")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
}

// TestMcpRemoveNamesUnmanagedEntrySaysHowToAdopt checks that a --names value with no install record, which an agent
// config holds as an unmanaged entry, says how to adopt the entry.
func TestMcpRemoveNamesUnmanagedEntrySaysHowToAdopt(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	seedInstallEntry(t, claude, seed{"genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test", testDatabricksCommand, ""})

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	out := readRemoveOutput(t, stdout.String())
	assert.Equal(t, "no installed MCP server named genie_one_mcp-logfood; Claude Code has an entry named genie_one_mcp-logfood that runs aitools mcp serve with no install record. To adopt it, so that remove works, run `databricks aitools mcp add --names system.ai.genie_one_mcp --profile logfood --agents claude-code`", out.Error)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPServiceNotFound, out.ErrorCategory)
}

func TestMcpRemoveRefusesAgentWithNothingInstalled(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--agents", "cursor", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	out := readRemoveOutput(t, stdout.String())
	assert.Equal(t, "no selected MCP server is installed for Cursor", out.Error)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPServiceNotFound, out.ErrorCategory)

	// Nothing was removed by the refusal.
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

	// The agent entry is deleted first; then the state cannot be written.
	saveMCPState = func(context.Context, *mcp.State) error { return errors.New("state file is read-only") }
	t.Cleanup(func() { saveMCPState = mcp.SaveState })

	stdout, stderr, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood")
	require.ErrorContains(t, err, "state file is read-only")
	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "Removed genie_one_mcp-logfood", "the results are printed before the error")
	assert.NotContains(t, jsonServers(t, claudePath), "genie_one_mcp-logfood")
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

// TestSelectInstallsServerNameWins checks that a --names value equal to a server name selects that server only: it is
// not also a service or a leaf of other servers, and it is not refused as ambiguous when another schema has the leaf.
func TestSelectInstallsServerNameWins(t *testing.T) {
	cases := []struct {
		name  string
		state []mcp.Entry
		want  []string
	}{
		{
			name: "server name does not also remove the other profile's server of the same service",
			state: []mcp.Entry{
				{Name: "github", Service: "system.ai.github", Profile: "logfood"},
				{Name: "github-dogfood", Service: "system.ai.github", Profile: "dogfood"},
			},
			want: []string{"github"},
		},
		{
			name: "server name is not refused as ambiguous when another schema has the leaf",
			state: []mcp.Entry{
				{Name: "github", Service: "system.ai.github", Profile: "logfood"},
				{Name: "gh-team", Service: "team.ai.github", Profile: "logfood"},
			},
			want: []string{"github"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selected, err := selectInstalls(&mcp.State{Servers: tc.state}, []string{"github"}, "")
			require.NoError(t, err)
			var names []string
			for _, e := range selected {
				names = append(names, e.Name)
			}
			assert.Equal(t, tc.want, names)
		})
	}
}

// TestMcpRemoveFromAnotherDirectoryFindsConfigSetRelative checks that a relative config dir given at add time is
// recorded as an absolute path, so that remove run from another directory still edits the file the entry is in.
func TestMcpRemoveFromAnotherDirectoryFindsConfigSetRelative(t *testing.T) {
	withHome(t)
	work := t.TempDir()
	t.Chdir(work)
	require.NoError(t, os.MkdirAll(filepath.Join(work, "cfg"), 0o755))
	t.Setenv("CLAUDE_CONFIG_DIR", "cfg")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	claudePath := filepath.Join(work, "cfg", ".claude.json")
	require.Contains(t, readJSONFile(t, claudePath)["mcpServers"], "genie_one_mcp-logfood")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.True(t, filepath.IsAbs(state.Servers[0].Agents["claude-code"].Path), "recorded path: %s", state.Servers[0].Agents["claude-code"].Path)

	t.Chdir(t.TempDir())
	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Empty(t, out.Skipped)
	assert.NotContains(t, jsonServers(t, claudePath), "genie_one_mcp-logfood", "the entry is removed from the file it was written to")
}

// TestMcpRemoveTakesEntryFromFileAgentNowReads checks that when the recorded file no longer holds the entry, because the
// user moved the file into the directory the agent now reads, the entry is removed from that file and the record goes.
// It is pruned only when neither file holds it.
func TestMcpRemoveTakesEntryFromFileAgentNowReads(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	moved := t.TempDir()
	require.NoError(t, os.Rename(claudePath, filepath.Join(moved, ".claude.json")))
	t.Setenv("CLAUDE_CONFIG_DIR", moved)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	assert.Empty(t, out.Skipped)
	assert.NotContains(t, jsonServers(t, filepath.Join(moved, ".claude.json")), "genie_one_mcp-logfood", "the entry is gone from the file the agent reads")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Empty(t, state.Servers, "the record went with the entry")
}

// TestSelectInstallsExactServerNameWins checks that --names matches a server name exactly before it matches without
// case. State can hold names that differ only in case, since add adopts the name an agent entry already has.
func TestSelectInstallsExactServerNameWins(t *testing.T) {
	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "Github", Service: "system.ai.github", Profile: "logfood"},
		{Name: "github", Service: "system.ai.github", Profile: "dogfood"},
	}}

	exact, err := selectInstalls(state, []string{"github"}, "")
	require.NoError(t, err)
	require.Len(t, exact, 1)
	assert.Equal(t, "github", exact[0].Name)

	_, err = selectInstalls(state, []string{"GITHUB"}, "")
	require.Error(t, err)
	assert.Equal(t, "GITHUB matches more than one MCP server (Github, github); pass the exact server name", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))

	// Narrowed to one profile, the case-insensitive name is unambiguous again.
	one, err := selectInstalls(state, []string{"GITHUB"}, "logfood")
	require.NoError(t, err)
	require.Len(t, one, 1)
	assert.Equal(t, "Github", one[0].Name)
}

// TestMcpRemoveWithAgentsSkipsRecordWithoutAgents checks that --agents selects only servers installed for those agents.
// An entry with no agent left is not chosen, so it is neither reported as removed nor dropped from state.
func TestMcpRemoveWithAgentsSkipsRecordWithoutAgents(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	dangling := mcp.Entry{Name: "dangling", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand}
	saveState(t, genie, dangling)

	stdout, _, err := runMcpCmd(t, "remove", "--all", "--agents", "claude-code", "--output", "json")
	require.NoError(t, err)
	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, "genie_one_mcp-logfood", out.Removed[0].Name)

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, "dangling", state.Servers[0].Name)

	// Named explicitly, the same entry is a refusal: no server is installed for the agent.
	_, _, err = runMcpCmd(t, "remove", "--names", "dangling", "--agents", "claude-code")
	require.ErrorContains(t, err, "no selected MCP server is installed for Claude Code")
}

// TestMcpRemoveRefusesNamedServerThatAgentsNarrowsToNothing checks that a --names run with two servers refuses the whole
// run when --agents narrows one of them to no agent, as it does for a single name. Nothing is written, so the other
// server is not removed on its own.
func TestMcpRemoveRefusesNamedServerThatAgentsNarrowsToNothing(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	cursor := agents.ByName(agents.NameCursor)
	claudePath := filepath.Join(home, ".claude.json")
	github := seedInstall(t, claude, "github-logfood", "system.ai.github", "logfood", "https://adb-1.test")
	genie := seedInstall(t, cursor, "genie-logfood", "system.ai.genie", "logfood", "https://adb-1.test")
	saveState(t, github, genie)
	claudeBefore, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	cursorPath := genie.Agents[cursor.Name].Path
	cursorBefore, err := os.ReadFile(cursorPath)
	require.NoError(t, err)

	// github-logfood is installed for Claude Code only, so --agents cursor narrows it to nothing.
	stdout, _, err := runMcpCmd(t, "remove", "--names", "github-logfood,genie-logfood", "--agents", "cursor", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted, "a named server that --agents narrows to nothing refuses the run")

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed)
	assert.Equal(t, "nothing was removed: named MCP servers are not installed for Cursor: github-logfood", out.Error)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPServiceNotFound, out.ErrorCategory)

	claudeAfter, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, claudeBefore, claudeAfter, "the Claude Code config is not written")
	cursorAfter, err := os.ReadFile(cursorPath)
	require.NoError(t, err)
	assert.Equal(t, cursorBefore, cursorAfter, "the Cursor config is not written")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Len(t, state.Servers, 2, "no record is dropped")
}

// TestMcpRemoveKeptReasonForDatabricksEntryOmitsForce checks that an entry this CLI wrote, but that serves another
// binding now, is cleared by hand. add refuses such an entry whatever --force says, so the reason does not offer it.
func TestMcpRemoveKeptReasonForDatabricksEntryOmitsForce(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	claudePath := filepath.Join(home, ".claude.json")
	editEntry(t, claudePath, entry.Name, func(server map[string]any) { setServeFlag(t, server, "--profile", "other") })

	stdout, _, err := runMcpCmd(t, "remove", "--names", entry.Name, "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "Claude Code has an entry named genie_one_mcp-logfood that was changed outside this CLI; left in place. "+
		"To remove it, delete the entry from "+filepath.ToSlash(claudePath)+" by hand and run `databricks aitools mcp remove --names genie_one_mcp-logfood` again",
		out.Skipped[0].Reason)
	assert.NotContains(t, out.Skipped[0].Reason, "--force")
}

// jsonServers returns the mcpServers object of the JSON config at path. It is nil once the last server is removed,
// since the removal drops the emptied container.
func jsonServers(t *testing.T, path string) map[string]any {
	t.Helper()
	servers, _ := readJSONFile(t, path)["mcpServers"].(map[string]any)
	return servers
}

// TestMcpRemoveRejectsSelectorThatNamesNothing checks that a --names or --agents value that splits to no name is an
// error, rather than a removal from every server or from every agent.
func TestMcpRemoveRejectsSelectorThatNamesNothing(t *testing.T) {
	withHome(t)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "agents", args: []string{"--all", "--agents", ","}, want: "--agents names no agent"},
		{name: "names", args: []string{"--profile", "logfood", "--names", ","}, want: "--names names no server"},
		{name: "empty names", args: []string{"--profile", "logfood", "--names", ""}, want: "--names names no server"},
		{name: "empty agents", args: []string{"--all", "--agents", ""}, want: "--agents names no agent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runMcpCmd(t, append([]string{"remove"}, tc.args...)...)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// TestMcpRemoveRefusesWhenNamedEntryIsKept checks that a --names run with a named server that is kept removes nothing
// and fails, as a name that matches nothing does, so the removable server is not removed on its own.
func TestMcpRemoveRefusesWhenNamedEntryIsKept(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	github := seedInstall(t, claude, "github-logfood", "system.ai.github", "logfood", "https://adb-1.test")
	saveState(t, genie, github)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	// The kept server is a Databricks entry that was changed outside this CLI: it serves another profile now.
	editEntry(t, claudePath, "github-logfood", func(server map[string]any) { setServeFlag(t, server, "--profile", "other") })
	before, err := os.ReadFile(claudePath)
	require.NoError(t, err)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood,github-logfood", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "github-logfood", out.Skipped[0].Name)
	assert.Contains(t, out.Error, "nothing was removed")
	assert.Contains(t, out.Error, "github-logfood")

	after, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the agent config is not written")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Len(t, state.Servers, 2, "no record is dropped")
}

// TestMcpRemoveKeepsRecordOfUnknownAgent checks that a record for an agent this CLI does not know is kept, and reported
// as skipped, so that a newer CLI that wrote it can still remove it.
func TestMcpRemoveKeepsRecordOfUnknownAgent(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	entry.Agents["future-agent"] = mcp.AgentRecord{Path: "/x/future.json", Scope: "user"}
	saveState(t, entry)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)

	out := readRemoveOutput(t, stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	assert.Empty(t, out.Removed[0].RecordOnly)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "future-agent", out.Skipped[0].Agent)
	assert.Contains(t, out.Skipped[0].Reason, "unknown agent")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, []string{"future-agent"}, slices.Collect(maps.Keys(state.Servers[0].Agents)))
}

// TestMcpRemoveRefusesNamedEntryWithOnlyUnknownAgent checks that a named server whose records are all for agents this
// CLI does not know refuses the run, as a kept server does: the user asked for it, and nothing is removed.
func TestMcpRemoveRefusesNamedEntryWithOnlyUnknownAgent(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	future := mcp.Entry{
		Name: "dbsql-logfood", Service: "system.ai.dbsql", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand,
		Agents: map[string]mcp.AgentRecord{"future-agent": {Path: "/x/future.json", Scope: "user"}},
	}
	saveState(t, genie, future)
	before, err := os.ReadFile(claudePath)
	require.NoError(t, err)

	stdout, _, err := runMcpCmd(t, "remove", "--names", "genie_one_mcp-logfood,dbsql-logfood", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	out := readRemoveOutput(t, stdout.String())
	assert.Empty(t, out.Removed)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "dbsql-logfood", out.Skipped[0].Name)
	assert.Equal(t, "future-agent", out.Skipped[0].Agent)
	assert.Contains(t, out.Skipped[0].Reason, "unknown agent future-agent")
	assert.Contains(t, out.Error, "dbsql-logfood")

	after, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the agent config is not written")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Len(t, state.Servers, 2, "no record is dropped")
}

// TestMcpRemovePickerRemovesTheChosenServers checks that the remove picker, shown in a terminal with no choice given,
// removes the servers that are checked and keeps the others.
func TestMcpRemovePickerRemovesTheChosenServers(t *testing.T) {
	isolateAgents(t)
	home := withHome(t)
	claudePath := filepath.Join(home, ".claude.json")
	require.NoError(t, os.WriteFile(claudePath, []byte(`{"mcpServers": {`+
		`"github-logfood": `+claudeEntry("system.ai.github", "logfood", "")+`, `+
		`"dbsql-logfood": `+claudeEntry("system.ai.dbsql", "logfood", "")+`}}`), 0o600))
	saveState(t, claudeServerEntry(claudePath, "github-logfood", "system.ai.github"), claudeServerEntry(claudePath, "dbsql-logfood", "system.ai.dbsql"))

	origRemove, origProceed := promptRemoveSelection, promptMCPProceed
	t.Cleanup(func() { promptRemoveSelection, promptMCPProceed = origRemove, origProceed })
	promptRemoveSelection = func(entries []mcp.Entry, _ []*agents.Agent) ([]mcp.Entry, error) {
		require.Len(t, entries, 2)
		return entries[1:], nil
	}
	promptMCPProceed = func() (bool, error) { return true, nil }

	require.NoError(t, runInteractive(t, "mcp", "remove"))
	servers := readJSONFile(t, claudePath)["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "github-logfood")
	assert.NotContains(t, servers, "dbsql-logfood")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, "github-logfood", state.Servers[0].Name)
}

// TestMcpRemovePickerRefusesAgentWithNoServer checks that --agents naming an agent with no installed server is refused
// before the picker runs, as a run without a terminal refuses it.
func TestMcpRemovePickerRefusesAgentWithNoServer(t *testing.T) {
	isolateAgents(t)
	withHome(t)
	saveState(t, claudeServerEntry("", "github-logfood", "system.ai.github"))

	origRemove := promptRemoveSelection
	t.Cleanup(func() { promptRemoveSelection = origRemove })
	promptRemoveSelection = func([]mcp.Entry, []*agents.Agent) ([]mcp.Entry, error) {
		t.Error("the picker ran for an agent with no installed server")
		return nil, nil
	}

	err := runInteractive(t, "mcp", "remove", "--agents", "codex")
	require.Error(t, err)
	assert.Equal(t, "no selected MCP server is installed for "+agents.ByName(agents.NameCodex).DisplayName, err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryMCPServiceNotFound, errorCategory(err))
	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Len(t, state.Servers, 1)
}

// TestMcpRemovePickerWithTwoAgentsNeedsOnePickPerServer checks that --agents with two agents accepts a pick installed for
// only one of them: the pick is removed from that agent, and the server installed for the other agent, which is not
// picked, stays.
func TestMcpRemovePickerWithTwoAgentsNeedsOnePickPerServer(t *testing.T) {
	isolateAgents(t)
	home := withHome(t)
	claudePath := filepath.Join(home, ".claude.json")
	require.NoError(t, os.WriteFile(claudePath, []byte(`{"mcpServers": {`+
		`"github-logfood": `+claudeEntry("system.ai.github", "logfood", "")+`, `+
		`"dbsql-logfood": `+claudeEntry("system.ai.dbsql", "logfood", "")+`}}`), 0o600))
	dbsql := claudeServerEntry(claudePath, "dbsql-logfood", "system.ai.dbsql")
	dbsql.Agents[agents.NameCodex] = mcp.AgentRecord{Path: filepath.Join(home, ".codex", "config.toml"), Scope: "user"}
	saveState(t, claudeServerEntry(claudePath, "github-logfood", "system.ai.github"), dbsql)

	origRemove, origProceed := promptRemoveSelection, promptMCPProceed
	t.Cleanup(func() { promptRemoveSelection, promptMCPProceed = origRemove, origProceed })
	promptRemoveSelection = func(entries []mcp.Entry, targets []*agents.Agent) ([]mcp.Entry, error) {
		require.Len(t, targets, 2)
		require.Len(t, entries, 2)
		return entries[:1], nil
	}
	promptMCPProceed = func() (bool, error) { return true, nil }

	require.NoError(t, runInteractive(t, "mcp", "remove", "--agents", "claude-code,codex"))
	servers := readJSONFile(t, claudePath)["mcpServers"].(map[string]any)
	assert.NotContains(t, servers, "github-logfood")
	assert.Contains(t, servers, "dbsql-logfood")
}

// TestMcpRemovePickerDeclinedRemovesNothing checks that a picked removal the user declines at Proceed? removes nothing.
func TestMcpRemovePickerDeclinedRemovesNothing(t *testing.T) {
	isolateAgents(t)
	withHome(t)
	saveState(t, claudeServerEntry("", "github-logfood", "system.ai.github"))

	origRemove, origProceed := promptRemoveSelection, promptMCPProceed
	t.Cleanup(func() { promptRemoveSelection, promptMCPProceed = origRemove, origProceed })
	promptRemoveSelection = func(entries []mcp.Entry, _ []*agents.Agent) ([]mcp.Entry, error) {
		return entries, nil
	}
	promptMCPProceed = func() (bool, error) { return false, nil }

	require.NoError(t, runInteractive(t, "mcp", "remove"))
	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Len(t, state.Servers, 1)
}

// TestMcpRemovePickerEmptyPickRemovesNothing checks that a picker closed with nothing checked removes nothing.
func TestMcpRemovePickerEmptyPickRemovesNothing(t *testing.T) {
	isolateAgents(t)
	withHome(t)
	saveState(t, claudeServerEntry("", "github-logfood", "system.ai.github"))

	origRemove := promptRemoveSelection
	t.Cleanup(func() { promptRemoveSelection = origRemove })
	promptRemoveSelection = func([]mcp.Entry, []*agents.Agent) ([]mcp.Entry, error) {
		return nil, nil
	}

	require.NoError(t, runInteractive(t, "mcp", "remove"))
	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Len(t, state.Servers, 1)
}

// claudeServerEntry is the state entry of a server installed for Claude Code, in the config file at claudePath.
func claudeServerEntry(claudePath, name, service string) mcp.Entry {
	return mcp.Entry{
		Name: name, Service: service, Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand,
		Agents: map[string]mcp.AgentRecord{agents.NameClaudeCode: {Path: claudePath, Scope: "user"}},
	}
}

// runRemoveWithTelemetry runs `mcp remove` with args on a context that buffers telemetry, and returns the remove command,
// so a test can read the context it left, and its stdout.
// The logs are those of the command's logger at logLevel.
func runRemoveWithTelemetry(t *testing.T, stdout, logs *bytes.Buffer, logLevel slog.Level, args ...string) (*cobra.Command, error) {
	t.Helper()
	parent := newMCPTestRoot()
	ctx := telemetry.WithNewLogger(t.Context())
	ctx = env.Set(ctx, "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	ctx = log.NewContext(ctx, slog.New(handler.NewFriendlyHandler(logs, &handler.Options{Level: logLevel})))
	ctx, _ = cmdio.NewTestContextWithStderr(ctx)
	parent.SetOut(stdout)
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs(append([]string{"mcp", "remove"}, args...))
	err := parent.ExecuteContext(ctx)
	remove, _, findErr := parent.Find([]string{"mcp", "remove"})
	require.NoError(t, findErr)
	return remove, err
}

// unsetConfigFile makes DATABRICKS_CONFIG_FILE unset for the test, as opposed to set to "".
func unsetConfigFile(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	require.NoError(t, os.Unsetenv("DATABRICKS_CONFIG_FILE"))
}

// TestMcpRemoveSetsNoUploadWithoutAnEvent checks that a run that removes nothing logs no event, so it sets no config for
// an upload.
func TestMcpRemoveSetsNoUploadWithoutAnEvent(t *testing.T) {
	withHome(t)
	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--all", "--output", "json")
	require.NoError(t, err)
	_, uploads := telemetry.UploadConfig(remove.Context())
	assert.False(t, uploads)
	assert.Empty(t, telemetry.BufferedEvents(remove.Context()))
}

// TestMcpRemoveUploadSkipsDriftedProfile checks that the upload is not authenticated with a profile that a server recorded
// for it no longer matches, as its host here is not the one the server was set up against. The removal is done either
// way.
func TestMcpRemoveUploadSkipsDriftedProfile(t *testing.T) {
	home := withHome(t)
	writeHomeConfig(t, home, "[logfood]\nhost = https://adb-2.test\nauth_type = pat\ntoken = t\n")
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	// Recorded against another host than the profile names now.
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	require.Len(t, telemetry.BufferedEvents(remove.Context()), 1, "the removal is logged")
	_, uploads := telemetry.UploadConfig(remove.Context())
	assert.False(t, uploads, "a drifted profile is not used to upload")
	assert.NotContains(t, jsonServers(t, claudePath), "genie_one_mcp-logfood", "the removal is done")
}

// TestMcpRemoveUploadUsesMatchingProfile checks that a profile whose recorded servers still match is used for the upload,
// read from the default config file when DATABRICKS_CONFIG_FILE is unset.
func TestMcpRemoveUploadUsesMatchingProfile(t *testing.T) {
	home := withHome(t)
	unsetConfigFile(t)
	writeHomeConfig(t, home, "[logfood]\nhost = https://adb-1.test\nauth_type = pat\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	cfg, uploads := telemetry.UploadConfig(remove.Context())
	require.True(t, uploads)
	assert.Equal(t, "logfood", cfg.Profile)
	assert.Equal(t, "https://adb-1.test", cfg.Host)
	assert.Equal(t, "t", cfg.Token)
}

// TestMcpRemoveUploadIgnoresEnvironmentToken checks that the upload authenticates with the profile's own token, never with
// a DATABRICKS_TOKEN of another workspace in the environment.
func TestMcpRemoveUploadIgnoresEnvironmentToken(t *testing.T) {
	home := withHome(t)
	unsetConfigFile(t)
	t.Setenv("DATABRICKS_TOKEN", "token-of-another-workspace")
	writeHomeConfig(t, home, "[logfood]\nhost = https://adb-1.test\nauth_type = pat\ntoken = t\n")
	claude := agents.ByName("claude-code")
	saveState(t, seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test"))

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	cfg, uploads := telemetry.UploadConfig(remove.Context())
	require.True(t, uploads)
	assert.Equal(t, "t", cfg.Token)
}

// TestMcpRemoveUploadUsesConfigFileOfServer checks that a server pinned to a config file uploads with the profile of that
// file, not with a profile of the same name in the file DATABRICKS_CONFIG_FILE names.
func TestMcpRemoveUploadUsesConfigFileOfServer(t *testing.T) {
	home := withHome(t)
	pinned := filepath.Join(home, "pinned.databrickscfg")
	require.NoError(t, os.WriteFile(pinned, []byte("[logfood]\nhost = https://adb-1.test\nauth_type = pat\ntoken = pinned-token\n"), 0o600))
	other := filepath.Join(home, "other.databrickscfg")
	require.NoError(t, os.WriteFile(other, []byte("[dogfood]\nhost = https://adb-2.test\nauth_type = pat\ntoken = other-token\n"), 0o600))
	claude := agents.ByName("claude-code")
	t.Setenv("DATABRICKS_CONFIG_FILE", pinned)
	saveState(t, seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test"))
	t.Setenv("DATABRICKS_CONFIG_FILE", other)

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	cfg, uploads := telemetry.UploadConfig(remove.Context())
	require.True(t, uploads)
	assert.Equal(t, "pinned-token", cfg.Token)
}

// TestMcpRemoveUploadIgnoresSameNamedProfileInOtherFile checks that a profile of the same name in another config file does
// not stop the upload for the server that matches its own file.
func TestMcpRemoveUploadIgnoresSameNamedProfileInOtherFile(t *testing.T) {
	home := withHome(t)
	first := filepath.Join(home, "first.databrickscfg")
	require.NoError(t, os.WriteFile(first, []byte("[logfood]\nhost = https://adb-1.test\nauth_type = pat\ntoken = first-token\n"), 0o600))
	second := filepath.Join(home, "second.databrickscfg")
	require.NoError(t, os.WriteFile(second, []byte("[logfood]\nhost = https://adb-2.test\nauth_type = pat\ntoken = second-token\n"), 0o600))
	claude := agents.ByName("claude-code")
	t.Setenv("DATABRICKS_CONFIG_FILE", first)
	firstEntry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	t.Setenv("DATABRICKS_CONFIG_FILE", second)
	secondEntry := seedInstall(t, claude, "dbsql-logfood", "system.ai.dbsql", "logfood", "https://adb-2.test")
	saveState(t, firstEntry, secondEntry)
	t.Setenv("DATABRICKS_CONFIG_FILE", first)

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	cfg, uploads := telemetry.UploadConfig(remove.Context())
	require.True(t, uploads)
	assert.Equal(t, "first-token", cfg.Token)
}

// TestMcpRemoveUploadSkipsServersOfSeveralProfiles checks that a remove of servers bound to two profiles logs its event but
// uploads nothing, since no one profile's token may stand in for the upload.
func TestMcpRemoveUploadSkipsServersOfSeveralProfiles(t *testing.T) {
	home := withHome(t)
	writeHomeConfig(t, home, "[DEFAULT]\nhost = https://adb-3.test\nauth_type = pat\ntoken = t\n[logfood]\nhost = https://adb-1.test\nauth_type = pat\ntoken = t\n[other]\nhost = https://adb-2.test\nauth_type = pat\ntoken = t\n")
	claude := agents.ByName("claude-code")
	saveState(t,
		seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test"),
		seedInstall(t, claude, "dbsql-other", "system.ai.dbsql", "other", "https://adb-2.test"),
	)

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--all", "--output", "json")
	require.NoError(t, err)
	require.Len(t, telemetry.BufferedEvents(remove.Context()), 1, "the removal is logged")
	_, uploads := telemetry.UploadConfig(remove.Context())
	assert.False(t, uploads, "servers of several profiles have no one profile to upload with")
}

// TestMcpRemoveMetadataFailureSetsUploadWithoutWarning checks that remove resolves the upload config quietly when
// the workspace has no metadata.
func TestMcpRemoveMetadataFailureSetsUploadWithoutWarning(t *testing.T) {
	home := withHome(t)
	server := noAPIServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = t\n")
	claude := agents.ByName("claude-code")
	saveState(t, seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", server.URL))

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelWarn, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	_, uploads := telemetry.UploadConfig(remove.Context())
	assert.True(t, uploads)
	assert.Empty(t, logs.String())
	assert.Contains(t, stdout.String(), `"removed"`)
}

// seedPrunedInstall records an install of genie_one_mcp for logfood in claude-code whose entry the agent config no longer holds.
func seedPrunedInstall(t *testing.T, claudePath string) {
	t.Helper()
	claude := agents.ByName("claude-code")
	saveState(t, seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test"))
	require.NoError(t, os.WriteFile(claudePath, []byte(`{"mcpServers":{}}`), 0o600))
}

// TestMcpRemovePrunedEntryReportsChanged checks that an entry the agent config no longer holds is reported CHANGED, since
// its record is dropped, and FAILED when the state cannot be saved, as a record-only removal is.
func TestMcpRemovePrunedEntryReportsChanged(t *testing.T) {
	home := withHome(t)
	claudePath := filepath.Join(home, ".claude.json")

	seedPrunedInstall(t, claudePath)
	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)
	events := telemetry.BufferedEvents(remove.Context())
	require.Len(t, events, 1)
	require.NotNil(t, events[0].AitoolsMcpEvent)
	require.Len(t, events[0].AitoolsMcpEvent.Servers, 1)
	assert.Equal(t, protos.AitoolsTargetOutcomeChanged, events[0].AitoolsMcpEvent.Servers[0].Outcome)

	seedPrunedInstall(t, claudePath)
	saveMCPState = func(context.Context, *mcp.State) error { return errors.New("state file is read-only") }
	t.Cleanup(func() { saveMCPState = mcp.SaveState })
	stdout.Reset()
	remove, err = runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	events = telemetry.BufferedEvents(remove.Context())
	require.Len(t, events, 1)
	require.NotNil(t, events[0].AitoolsMcpEvent)
	require.Len(t, events[0].AitoolsMcpEvent.Servers, 1)
	assert.Equal(t, protos.AitoolsTargetOutcomeFailed, events[0].AitoolsMcpEvent.Servers[0].Outcome)
}

// TestMcpRemoveFailedStateReadAfterAnotherEventLogsNothing checks that a remove that cannot read the state logs nothing
// of its own, even when the run already buffered another event, and does not read the state it never read.
func TestMcpRemoveFailedStateReadAfterAnotherEventLogsNothing(t *testing.T) {
	withHome(t)
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".state.json"), 0o700))

	parent := newMCPTestRoot()
	parent.SetOut(&bytes.Buffer{})
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "remove", "--all", "--output", "json"})
	ctx := telemetry.WithNewLogger(t.Context())
	telemetry.Log(ctx, protos.DatabricksCliLog{CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1}})
	ctx, _ = cmdio.NewTestContextWithStderr(ctx)
	require.Error(t, parent.ExecuteContext(ctx))
	remove, _, findErr := parent.Find([]string{"mcp", "remove"})
	require.NoError(t, findErr)
	events := telemetry.BufferedEvents(remove.Context())
	require.Len(t, events, 1)
	assert.NotNil(t, events[0].CliTestEvent)
}

// TestMcpRemoveRecordOnlyReportsFailedWhenStateIsNotSaved checks that a removal that only drops state records is reported
// FAILED when the state cannot be saved, since nothing was removed.
func TestMcpRemoveRecordOnlyReportsFailedWhenStateIsNotSaved(t *testing.T) {
	withHome(t)
	recordOnly := mcp.Entry{
		Name: "dbsql-logfood", Service: "system.ai.dbsql", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand,
		Agents: map[string]mcp.AgentRecord{"pi": {Path: "/x/config.json", Scope: "user"}},
	}
	saveState(t, recordOnly)
	saveMCPState = func(context.Context, *mcp.State) error { return errors.New("state file is read-only") }
	t.Cleanup(func() { saveMCPState = mcp.SaveState })

	var stdout, logs bytes.Buffer
	remove, err := runRemoveWithTelemetry(t, &stdout, &logs, log.LevelDebug, "--names", "dbsql-logfood", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	events := telemetry.BufferedEvents(remove.Context())
	require.Len(t, events, 1)
	require.NotNil(t, events[0].AitoolsMcpEvent)
	require.Len(t, events[0].AitoolsMcpEvent.Servers, 1)
	assert.Equal(t, protos.AitoolsTargetOutcomeFailed, events[0].AitoolsMcpEvent.Servers[0].Outcome)
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, events[0].AitoolsMcpEvent.Servers[0].ErrorCategory)
}
