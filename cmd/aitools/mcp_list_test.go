package aitools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDatabricksCommand = "/opt/databricks/bin/databricks"

// seedInstall writes the server for an agent with Upsert, as `add` does, and returns the state record.
func seedInstall(t *testing.T, a *agents.Agent, name, service, profileName, host string) mcp.Entry {
	t.Helper()
	args := []string{"aitools", "mcp", "serve", "--profile", profileName, "--service", service, "--agent", a.Name}
	path, _, err := agentconfig.Upsert(t.Context(), a, name, testDatabricksCommand, args, nil, false)
	require.NoError(t, err)
	return mcp.Entry{
		Name:    name,
		Service: service,
		Profile: profileName,
		Host:    host,
		Command: testDatabricksCommand,
		Agents:  map[string]mcp.AgentRecord{a.Name: {Path: path, Scope: "user"}},
	}
}

// repointEntry sets the command of a Claude Code server in its config file, as a user editing the file
// by hand would. Upsert refuses a command that is not the databricks binary, so it cannot write this.
func repointEntry(t *testing.T, path, name, command string) {
	t.Helper()
	doc := readJSONFile(t, path)
	server := doc["mcpServers"].(map[string]any)[name].(map[string]any)
	server["command"] = command
	data, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func saveState(t *testing.T, entries ...mcp.Entry) {
	t.Helper()
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: entries}))
}

func TestEntryStatus(t *testing.T) {
	rec := mcp.Entry{Name: "github-logfood", Command: testDatabricksCommand}
	ours := map[string]any{"command": testDatabricksCommand, "args": []any{"aitools", "mcp", "serve"}}
	otherCommand := map[string]any{"command": "/usr/bin/other", "args": []any{"aitools", "mcp", "serve"}}
	disk := map[string]map[string]map[string]any{
		"claude-code": {"github-logfood": ours},
		"codex":       {"github-logfood": otherCommand},
	}

	assert.Equal(t, agentStatusInstalled, entryStatus(rec, "claude-code", disk))
	assert.Equal(t, agentStatusMissingOnDisk, entryStatus(rec, "cursor", disk))
	assert.Equal(t, agentStatusModified, entryStatus(rec, "codex", disk), "a different command is not ours")

	moved := rec
	moved.Command = "/usr/local/bin/databricks"
	assert.Equal(t, agentStatusModified, entryStatus(moved, "claude-code", disk), "a recorded command that no longer matches is modified")
}

func TestUnmanagedKind(t *testing.T) {
	assert.Equal(t, unmanagedKindUGProxy, unmanagedKind(map[string]any{"command": "ug", "args": []any{"mcp-proxy", "web"}}))
	assert.Equal(t, unmanagedKindUGProxy, unmanagedKind(map[string]any{"command": []any{"ug", "mcp-proxy"}}))
	assert.Equal(t, unmanagedKindAIGateway, unmanagedKind(map[string]any{"url": "https://adb-1.test/ai-gateway/mcp-services/github"}))
	assert.Empty(t, unmanagedKind(map[string]any{"command": "npx", "args": []any{"some-server"}}))
	assert.Empty(t, unmanagedKind(map[string]any{"command": testDatabricksCommand, "args": []any{"aitools", "mcp", "serve"}}))
}

func TestBuildMcpListDriftAndUnmanaged(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, `[logfood]
host = https://adb-1.test
token = t

[dogfood]
host = https://adb-2.test
token = t
`)
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	// A server this CLI did not write, reached through ug, next to ours.
	require.NoError(t, os.WriteFile(claudePath, []byte(`{"mcpServers":{"web_search":{"command":"ug","args":["mcp-proxy"]}}}`), 0o600))

	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	github := seedInstall(t, claude, "github-logfood", "system.ai.github", "logfood", "https://adb-old.test")
	dogfood := seedInstall(t, claude, "genie_one_mcp-dogfood", "system.ai.genie_one_mcp", "dogfood", "https://adb-2.test")
	// Recorded for claude-code but never written to its config file.
	absent := mcp.Entry{
		Name: "dbsql-dogfood", Service: "system.ai.dbsql", Profile: "dogfood", Host: "https://adb-2.test", Command: testDatabricksCommand,
		Agents: map[string]mcp.AgentRecord{"claude-code": {Path: claudePath, Scope: "user"}},
	}
	// Written for a profile that is then removed from the config file, and changed by hand afterwards.
	gone := seedInstall(t, claude, "dbsql-old", "system.ai.dbsql", "old", "https://adb-3.test")
	repointEntry(t, claudePath, "dbsql-old", "/usr/bin/other")

	saveState(t, genie, github, dogfood, absent, gone)

	out, err := buildMcpList(t.Context())
	require.NoError(t, err)

	require.Len(t, out.Profiles, 3)

	dogfoodProfile := out.Profiles[0]
	assert.Equal(t, "dogfood", dogfoodProfile.Profile)
	assert.Equal(t, profileStatusOK, dogfoodProfile.Status)
	assert.Equal(t, "https://adb-2.test", dogfoodProfile.Host)
	assert.Equal(t, []mcpListAgent{{Name: "claude-code", Path: claudePath, Status: agentStatusInstalled}}, dogfoodProfile.Servers[0].Agents)
	assert.Equal(t, []mcpListAgent{{Name: "claude-code", Path: claudePath, Status: agentStatusMissingOnDisk}}, dogfoodProfile.Servers[1].Agents)

	logfood := out.Profiles[1]
	assert.Equal(t, "logfood", logfood.Profile)
	assert.Equal(t, "host_changed", logfood.Status)
	assert.Equal(t, "https://adb-1.test", logfood.Host, "a changed host shows the current host of the profile")
	require.Len(t, logfood.Servers, 2)
	assert.Equal(t, agentStatusInstalled, logfood.Servers[0].Agents[0].Status)
	assert.Equal(t, agentStatusInstalled, logfood.Servers[1].Agents[0].Status)

	old := out.Profiles[2]
	assert.Equal(t, "old", old.Profile)
	assert.Equal(t, "profile_missing", old.Status)
	assert.Equal(t, "https://adb-3.test", old.Host, "a missing profile shows the recorded host")
	assert.Equal(t, agentStatusModified, old.Servers[0].Agents[0].Status)

	assert.Equal(t, []mcpListUnmanaged{{Agent: "claude-code", Name: "web_search", Kind: unmanagedKindUGProxy}}, out.Unmanaged)
	assert.Equal(t, os.Getenv("DATABRICKS_CONFIG_FILE"), out.configPath)
}

func TestBuildMcpListEmpty(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")

	out, err := buildMcpList(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, out.Profiles, "an empty listing is [] in JSON, not null")
	assert.NotNil(t, out.Unmanaged)
	assert.Empty(t, out.Profiles)
	assert.Empty(t, out.Unmanaged)
}
