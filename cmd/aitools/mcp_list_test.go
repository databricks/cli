package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDatabricksCommand = "/opt/databricks/bin/databricks"

// seed describes one server as `add` writes it: its binding, the command it runs and the config file it is pinned to.
type seed struct {
	name, service, profile, host, command, configFile string
}

// seedInstallEntry writes the server for an agent with Upsert and returns the state record. Like `add`, the entry
// and the record are pinned to configFile when it is set.
func seedInstallEntry(t *testing.T, a *agents.Agent, s seed) mcp.Entry {
	t.Helper()
	args := []string{"aitools", "mcp", "serve", "--profile", s.profile, "--service", s.service, "--agent", a.Name}
	var env map[string]string
	if s.configFile != "" {
		env = map[string]string{mcpconfig.ConfigFileEnv: s.configFile}
	}
	path, _, err := agentconfig.Upsert(t.Context(), a, s.name, s.command, args, env, false)
	require.NoError(t, err)
	return mcp.Entry{
		Name:       s.name,
		Service:    s.service,
		Profile:    s.profile,
		Host:       s.host,
		Command:    s.command,
		ConfigFile: s.configFile,
		Agents:     map[string]mcp.AgentRecord{a.Name: {Path: path, Scope: "user"}},
	}
}

// seedInstall writes the server for an agent with the test command, pinned to DATABRICKS_CONFIG_FILE when it is set.
func seedInstall(t *testing.T, a *agents.Agent, name, service, profileName, host string) mcp.Entry {
	t.Helper()
	return seedInstallEntry(t, a, seed{name, service, profileName, host, testDatabricksCommand, os.Getenv("DATABRICKS_CONFIG_FILE")})
}

// editEntry changes the server name in the agent config at path, as a user editing the file by hand would.
func editEntry(t *testing.T, path, name string, edit func(server map[string]any)) {
	t.Helper()
	doc := readJSONFile(t, path)
	edit(doc["mcpServers"].(map[string]any)[name].(map[string]any))
	data, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

// repointEntry sets the command of a Claude Code server, as a user editing the file by hand would. Upsert refuses a
// command that is not the databricks binary, so it cannot write this.
func repointEntry(t *testing.T, path, name, command string) {
	t.Helper()
	editEntry(t, path, name, func(server map[string]any) { server["command"] = command })
}

// setServeFlag changes the value of a `serve` flag in the argument list of a hand-edited entry.
func setServeFlag(t *testing.T, server map[string]any, flag, value string) {
	t.Helper()
	args, ok := server["args"].([]any)
	require.True(t, ok)
	for i := range args {
		if args[i] == flag {
			args[i+1] = value
			return
		}
	}
	require.Failf(t, "flag not found", "%s", flag)
}

func saveState(t *testing.T, entries ...mcp.Entry) {
	t.Helper()
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: entries}))
}

// runMcpCmd runs `mcp list` or `mcp remove` with args and returns its stdout and stderr.
func runMcpCmd(t *testing.T, args ...string) (stdout, stderr *bytes.Buffer, err error) {
	t.Helper()
	parent := newMCPTestRoot()
	ctx, stderrBuf := cmdio.NewTestContextWithStderr(t.Context())
	var stdoutBuf bytes.Buffer
	parent.SetOut(&stdoutBuf)
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs(append([]string{"mcp"}, args...))
	return &stdoutBuf, stderrBuf, parent.ExecuteContext(ctx)
}

func TestMcpHandEditedEntryIsModified(t *testing.T) {
	cases := []struct {
		name string
		edit func(t *testing.T, server map[string]any)
	}{
		{"profile", func(t *testing.T, server map[string]any) {
			setServeFlag(t, server, "--profile", "other")
		}},
		{"service", func(t *testing.T, server map[string]any) {
			setServeFlag(t, server, "--service", "system.ai.github")
		}},
		{"pinned config file", func(t *testing.T, server map[string]any) {
			server["env"] = map[string]any{mcpconfig.ConfigFileEnv: filepath.Join(t.TempDir(), "other.cfg")}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := withHome(t)
			writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
			claude := agents.ByName("claude-code")
			claudePath := filepath.Join(home, ".claude.json")
			entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
			saveState(t, entry)
			editEntry(t, claudePath, entry.Name, func(server map[string]any) { tc.edit(t, server) })

			out, err := buildMcpList(t.Context(), "")
			require.NoError(t, err)
			require.Len(t, out.Profiles, 1)
			require.Len(t, out.Profiles[0].Servers, 1)
			require.Len(t, out.Profiles[0].Servers[0].Agents, 1)
			assert.Equal(t, agentStatusModified, out.Profiles[0].Servers[0].Agents[0].Status, "an entry that serves another binding is not ours")

			stdout, _, err := runMcpCmd(t, "remove", "--names", entry.Name, "--output", "json")
			require.NoError(t, err)
			var removed removeOutput
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &removed), stdout.String())
			assert.Empty(t, removed.Removed)
			require.Len(t, removed.Skipped, 1)
			assert.Equal(t, claude.Name, removed.Skipped[0].Agent)
			assert.Contains(t, readJSONFile(t, claudePath)["mcpServers"], entry.Name, "the hand-edited entry is kept")
		})
	}
}

func TestMcpReAddFromAnotherBinaryKeepsOtherAgentsRemovable(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	codex := agents.ByName("codex")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))

	const name = "genie_one_mcp-logfood"
	const service = "system.ai.genie_one_mcp"
	first := seedInstall(t, claude, name, service, "logfood", "https://adb-1.test")
	// `add` for codex ran from another binary, and the record keeps the command of the last add.
	second := seedInstallEntry(t, codex, seed{name, service, "logfood", "https://adb-1.test", "/opt/other/databricks", first.ConfigFile})
	first.Agents[codex.Name] = second.Agents[codex.Name]
	first.Command = "/opt/other/databricks"
	saveState(t, first)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, out.Profiles, 1)
	for _, a := range out.Profiles[0].Servers[0].Agents {
		assert.Equal(t, agentStatusInstalled, a.Status, a.Name)
	}

	stdout, _, err := runMcpCmd(t, "remove", "--names", name, "--output", "json")
	require.NoError(t, err)
	var removed removeOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &removed), stdout.String())
	require.Len(t, removed.Removed, 1)
	assert.ElementsMatch(t, []string{"claude-code", "codex"}, removed.Removed[0].Agents)
	assert.Empty(t, removed.Skipped)
}

func TestBuildMcpListGroupsProfilesByConfigFile(t *testing.T) {
	withHome(t)
	dir := t.TempDir()
	first := filepath.Join(dir, "first.cfg")
	second := filepath.Join(dir, "second.cfg")
	// The profile of the first file now names another host. The second file is healthy.
	require.NoError(t, os.WriteFile(first, []byte("[logfood]\nhost = https://adb-9.test\ntoken = t\n"), 0o600))
	require.NoError(t, os.WriteFile(second, []byte("[logfood]\nhost = https://adb-2.test\ntoken = t\n"), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", first)

	claude := agents.ByName("claude-code")
	// One server recorded against the old host, one added later against the new host, and one in the second file.
	drifted := seedInstallEntry(t, claude, seed{"genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test", testDatabricksCommand, first})
	current := seedInstallEntry(t, claude, seed{"github-logfood", "system.ai.github", "logfood", "https://adb-9.test", testDatabricksCommand, first})
	healthy := seedInstallEntry(t, claude, seed{"genie_one_mcp-other", "system.ai.genie_one_mcp", "logfood", "https://adb-2.test", testDatabricksCommand, second})
	saveState(t, drifted, current, healthy)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, out.Profiles, 2, "one profile name in two config files is two groups")

	firstPath, err := mcp.ResolveConfigFile(t.Context(), first)
	require.NoError(t, err)
	secondPath, err := mcp.ResolveConfigFile(t.Context(), second)
	require.NoError(t, err)

	pFirst := out.Profiles[0]
	assert.Equal(t, filepath.ToSlash(firstPath), pFirst.ConfigFile)
	assert.Equal(t, "host_changed", pFirst.Status, "the group status is the worst of its servers")
	assert.Equal(t, "https://adb-9.test", pFirst.Host)
	require.Len(t, pFirst.Servers, 2)
	assert.Equal(t, "host_changed", pFirst.Servers[0].Status)
	assert.Equal(t, "ok", pFirst.Servers[1].Status, "a server added against the new host is not broken")

	pSecond := out.Profiles[1]
	assert.Equal(t, filepath.ToSlash(secondPath), pSecond.ConfigFile)
	assert.Equal(t, "ok", pSecond.Status)
	require.Len(t, pSecond.Servers, 1)
	assert.Equal(t, "ok", pSecond.Servers[0].Status)
}

func TestBuildMcpListReportsAgentPathChanged(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	recordedPath := filepath.Join(home, ".claude.json")

	// The agent config now lives elsewhere, so the entry is in a file this command does not read.
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, out.Profiles, 1)
	require.Len(t, out.Profiles[0].Servers, 1)
	assert.Equal(t, []mcpListAgent{{Name: "claude-code", Path: filepath.ToSlash(recordedPath), Status: agentStatusPathChanged}}, out.Profiles[0].Servers[0].Agents)
}

// TestBuildMcpListRecordForUnknownAgentIsUnknown checks that a record for an agent outside the registry is unknown,
// with a warning, rather than a config that moved.
func TestBuildMcpListRecordForUnknownAgentIsUnknown(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")
	saveState(t, mcp.Entry{
		Name: "genie_one_mcp-logfood", Service: "system.ai.genie_one_mcp", Profile: "logfood", Host: "https://adb-1.test",
		Command: testDatabricksCommand, Agents: map[string]mcp.AgentRecord{"retired-agent": {Path: "/x/config.json", Scope: "user"}},
	})

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, out.Profiles, 1)
	require.Len(t, out.Profiles[0].Servers, 1)
	require.Len(t, out.Profiles[0].Servers[0].Agents, 1)
	assert.Equal(t, statusUnknown, out.Profiles[0].Servers[0].Agents[0].Status)
	assert.Contains(t, out.Warnings, "could not read the retired-agent config: retired-agent is not a known agent")
}

func TestBuildMcpListUnknownServerIsWarnedNotOK(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")
	broken := filepath.Join(t.TempDir(), "broken.cfg")
	require.NoError(t, os.WriteFile(broken, []byte("[logfood\n"), 0o600))

	claude := agents.ByName("claude-code")
	gone := seedInstall(t, claude, "dbsql-old", "system.ai.dbsql", "old", "https://adb-3.test")
	unreadable := mcp.Entry{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-1.test",
		Command: testDatabricksCommand, ConfigFile: broken,
	}
	saveState(t, gone, unreadable)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	require.NotEmpty(t, out.Warnings)
	// The warning names the resolved file, which on Windows expands the short name of the temp directory.
	resolved, err := filepath.EvalSymlinks(broken)
	require.NoError(t, err)
	assert.Contains(t, out.Warnings[0], filepath.ToSlash(resolved)+": failed to read Databricks profiles", "the file is warned about once, by name")

	require.Len(t, out.Profiles, 2)
	logfood := out.Profiles[0]
	assert.Equal(t, "logfood", logfood.Profile)
	assert.Equal(t, statusUnknown, logfood.Status, "a profile with an entry that could not be checked is not ok")
	require.Len(t, logfood.Servers, 1)
	assert.Equal(t, statusUnknown, logfood.Servers[0].Status, "an entry that could not be checked is not reported as ok")
	assert.Equal(t, "old", out.Profiles[1].Profile)
	assert.Equal(t, "profile_missing", out.Profiles[1].Status)
}

func TestMcpListExitsZeroWhenAnEntryCannotBeChecked(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")
	broken := filepath.Join(t.TempDir(), "broken.cfg")
	require.NoError(t, os.WriteFile(broken, []byte("[logfood\n"), 0o600))
	saveState(t, mcp.Entry{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-1.test",
		Command: testDatabricksCommand, ConfigFile: broken,
	})

	stdout, _, err := runMcpCmd(t, "list", "--output", "json")
	require.NoError(t, err, "an entry that cannot be checked is a warning, not a failure")
	var out mcpListOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	require.Len(t, out.Profiles, 1)
	require.Len(t, out.Profiles[0].Servers, 1)
	assert.Equal(t, statusUnknown, out.Profiles[0].Servers[0].Status)
	assert.NotEmpty(t, out.Warnings)
}

func TestBuildMcpListWarnsForUnreadableAgentConfig(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	codex := agents.ByName("codex")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".codex", "config.toml"), []byte("[mcp_servers.broken\n"), 0o600))
	claudePath := filepath.Join(home, ".claude.json")
	require.NoError(t, os.WriteFile(claudePath, []byte(`{"mcpServers":{"web_search":{"command":"ug","args":["mcp-proxy"]}}}`), 0o600))

	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	// Recorded for codex, whose config cannot be read.
	github := mcp.Entry{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-1.test",
		Command: testDatabricksCommand, ConfigFile: os.Getenv("DATABRICKS_CONFIG_FILE"),
		Agents: map[string]mcp.AgentRecord{codex.Name: {Path: filepath.Join(home, ".codex", "config.toml"), Scope: "user"}},
	}
	saveState(t, genie, github)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err, "one unreadable agent does not fail the listing")
	require.Len(t, out.Warnings, 1)
	assert.Contains(t, out.Warnings[0], "Codex CLI")

	require.Len(t, out.Profiles, 1)
	servers := out.Profiles[0].Servers
	require.Len(t, servers, 2)
	require.Len(t, servers[0].Agents, 1)
	require.Len(t, servers[1].Agents, 1)
	assert.Equal(t, agentStatusInstalled, servers[0].Agents[0].Status, "the readable agent is listed")
	assert.Equal(t, statusUnknown, servers[1].Agents[0].Status, "the recorded server in the unreadable agent is unknown")
	assert.Equal(t, []mcpListUnmanaged{{Agent: "claude-code", Name: "web_search", Kind: unmanagedKindUGProxy}}, out.Unmanaged)
}

func TestBuildMcpListReportsOrphanDatabricksEntry(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")
	claude := agents.ByName("claude-code")
	// Written by this CLI, but its state record is gone.
	seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	assert.Empty(t, out.Profiles)
	assert.Equal(t, []mcpListUnmanaged{{Agent: "claude-code", Name: "genie_one_mcp-logfood", Kind: unmanagedKindNoRecord}}, out.Unmanaged)
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
		ConfigFile: os.Getenv("DATABRICKS_CONFIG_FILE"),
		Agents:     map[string]mcp.AgentRecord{"claude-code": {Path: claudePath, Scope: "user"}},
	}
	// Written for a profile that is then removed from the config file, and changed by hand afterwards.
	gone := seedInstall(t, claude, "dbsql-old", "system.ai.dbsql", "old", "https://adb-3.test")
	repointEntry(t, claudePath, "dbsql-old", "/usr/bin/other")

	saveState(t, genie, github, dogfood, absent, gone)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, out.Profiles, 3)

	dogfoodProfile := out.Profiles[0]
	assert.Equal(t, "dogfood", dogfoodProfile.Profile)
	assert.Equal(t, statusOK, dogfoodProfile.Status)
	assert.Equal(t, "https://adb-2.test", dogfoodProfile.Host)
	require.Len(t, dogfoodProfile.Servers, 2)
	assert.Equal(t, []mcpListAgent{{Name: "claude-code", Path: filepath.ToSlash(claudePath), Status: agentStatusInstalled}}, dogfoodProfile.Servers[0].Agents)
	assert.Equal(t, []mcpListAgent{{Name: "claude-code", Path: filepath.ToSlash(claudePath), Status: agentStatusMissingOnDisk}}, dogfoodProfile.Servers[1].Agents)

	logfood := out.Profiles[1]
	assert.Equal(t, "logfood", logfood.Profile)
	assert.Equal(t, "host_changed", logfood.Status)
	assert.Equal(t, "https://adb-1.test", logfood.Host, "a changed host shows the current host of the profile")
	require.Len(t, logfood.Servers, 2)
	assert.Equal(t, "ok", logfood.Servers[0].Status)
	require.Len(t, logfood.Servers[0].Agents, 1)
	assert.Equal(t, agentStatusInstalled, logfood.Servers[0].Agents[0].Status)
	assert.Equal(t, "https://adb-1.test", logfood.Servers[0].Host, "a server shows the host it was set up for")
	assert.Equal(t, "host_changed", logfood.Servers[1].Status, "github-logfood was recorded against the old host")
	assert.Equal(t, "https://adb-old.test", logfood.Servers[1].Host, "the recorded host, not the profile's new one")
	require.Len(t, logfood.Servers[1].Agents, 1)
	assert.Equal(t, agentStatusInstalled, logfood.Servers[1].Agents[0].Status)

	old := out.Profiles[2]
	assert.Equal(t, "old", old.Profile)
	assert.Equal(t, "profile_missing", old.Status)
	assert.Equal(t, "https://adb-3.test", old.Host, "a missing profile shows the recorded host")
	require.Len(t, old.Servers, 1)
	require.Len(t, old.Servers[0].Agents, 1)
	assert.Equal(t, agentStatusModified, old.Servers[0].Agents[0].Status)

	assert.Equal(t, []mcpListUnmanaged{{Agent: "claude-code", Name: "web_search", Kind: unmanagedKindUGProxy}}, out.Unmanaged)
	pinned, err := mcp.ResolveConfigFile(t.Context(), os.Getenv("DATABRICKS_CONFIG_FILE"))
	require.NoError(t, err)
	assert.Equal(t, filepath.ToSlash(pinned), out.Profiles[0].ConfigFile, "a pinned profile names its config file")
}

func TestBuildMcpListEmpty(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, out.Profiles, "an empty listing is [] in JSON, not null")
	assert.NotNil(t, out.Unmanaged)
	assert.NotNil(t, out.Warnings)
	assert.Empty(t, out.Profiles)
	assert.Empty(t, out.Unmanaged)
}

func TestUnmanagedKind(t *testing.T) {
	cases := []struct {
		name  string
		entry map[string]any
		want  string
	}{
		{"ug mcp-proxy", map[string]any{"command": "ug", "args": []any{"mcp-proxy", "web"}}, unmanagedKindUGProxy},
		{"ug mcp-proxy as a command array", map[string]any{"command": []any{"ug", "mcp-proxy"}}, unmanagedKindUGProxy},
		{"ai gateway URL", map[string]any{"url": "https://adb-1.test/ai-gateway/mcp-services/github"}, unmanagedKindAIGateway},
		{"other command", map[string]any{"command": "npx", "args": []any{"some-server"}}, ""},
		{"mcp-proxy under another command with an ai gateway URL", map[string]any{"command": "npx", "args": []any{"mcp-proxy"}, "url": "https://adb-1.test/ai-gateway/mcp-services/github"}, unmanagedKindAIGateway},
		{"databricks serve without a record", map[string]any{"command": testDatabricksCommand, "args": []any{"aitools", "mcp", "serve"}}, unmanagedKindNoRecord},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, unmanagedKind(tc.entry))
		})
	}
}

// TestRenderMcpListNamesTheFileAndTheRepair checks the text of the drifts: the file that was checked, the pinned
// variable in the repair hints, and the shell quoting of the profile.
func TestRenderMcpListNamesTheFileAndTheRepair(t *testing.T) {
	withHome(t)
	dir := t.TempDir()
	pinned := filepath.Join(dir, "pinned.cfg")
	require.NoError(t, os.WriteFile(pinned, []byte("[logfood]\nhost = https://adb-1.test\nworkspace_id = 222\ntoken = t\n\n[nohost]\ntoken = t\n"), 0o600))
	pinnedPath, err := mcp.ResolveConfigFile(t.Context(), pinned)
	require.NoError(t, err)
	claude := agents.ByName("claude-code")

	workspaceMoved := seedInstallEntry(t, claude, seed{"genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test", testDatabricksCommand, pinned})
	workspaceMoved.WorkspaceID = "111"
	// The workspace is named in the host's query, not in workspace_id.
	workspaceFromQuery := seedInstallEntry(t, claude, seed{"genie_one_mcp-query", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test/?o=111", testDatabricksCommand, pinned})
	hostMoved := seedInstallEntry(t, claude, seed{"github-moved", "system.ai.github", "logfood", "https://adb-old.test", testDatabricksCommand, pinned})
	// The profile is not in the file at all, and the profile name needs quoting in the hint.
	missing := seedInstallEntry(t, claude, seed{"github-gone", "system.ai.github", "gone profile", "https://adb-old.test", testDatabricksCommand, pinned})
	noHost := seedInstallEntry(t, claude, seed{"dbsql-nohost", "system.ai.dbsql", "nohost", "https://adb-4.test", testDatabricksCommand, pinned})
	saveState(t, workspaceMoved, workspaceFromQuery, hostMoved, missing, noHost)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	renderMcpList(ctx, out, "")
	text := stderr.String()

	shownPath := filepath.ToSlash(pinnedPath)
	assert.Contains(t, text, "Profile logfood  (https://adb-1.test, config file "+shownPath+")")
	assert.Contains(t, text, "broken: it was set up for workspace 111, but profile \"logfood\" now points at workspace 222; run `databricks aitools mcp remove --names genie_one_mcp-logfood` and add it again")
	assert.Contains(t, text, "broken: it was set up for workspace 111, but profile \"logfood\" now points at workspace 222; run `databricks aitools mcp remove --names genie_one_mcp-query` and add it again")
	assert.Contains(t, text, "broken: it was set up for https://adb-old.test, but profile \"logfood\" now points at https://adb-1.test; run `databricks aitools mcp remove --names github-moved` and add it again")
	assert.Contains(t, text, "Profile gone profile  (profile not found in "+shownPath+")")
	// The config path is quoted when it has characters a shell would read, such as the colon of a Windows drive.
	assert.Contains(t, text, "broken: run `databricks aitools mcp remove --names github-gone` or `DATABRICKS_CONFIG_FILE="+shellquote.BashArg(shownPath)+" databricks auth login --host https://adb-old.test --profile 'gone profile'`")
	assert.Contains(t, text, "Profile nohost  (profile has no host in "+shownPath+")")
	assert.NotContains(t, text, "Warning", "every entry in this test can be checked")
}

func TestRenderMcpListWarnsAndNamesUnmanagedRecords(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")
	claude := agents.ByName("claude-code")
	seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	renderMcpList(ctx, out, "")
	assert.Contains(t, stderr.String(), "  Claude Code    genie_one_mcp-logfood databricks aitools mcp serve, no install record")
}

// TestMcpListFiltersToProfile checks that --profile lists the servers of that profile only, with no unmanaged entries,
// since those have no profile, and says so when the profile has no server.
func TestMcpListFiltersToProfile(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n\n[dogfood]\nhost = https://adb-2.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	claudePath := filepath.Join(home, ".claude.json")
	genie := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	other := seedInstall(t, claude, "github-dogfood", "system.ai.github", "dogfood", "https://adb-2.test")
	saveState(t, genie, other)
	doc := readJSONFile(t, claudePath)
	doc["mcpServers"].(map[string]any)["web_search"] = map[string]any{"command": "ug", "args": []any{"mcp-proxy"}}
	data, err := json.Marshal(doc)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(claudePath, data, 0o600))

	stdout, _, err := runMcpCmd(t, "list", "--profile", "logfood", "--output", "json")
	require.NoError(t, err)
	var out mcpListOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	require.Len(t, out.Profiles, 1)
	assert.Equal(t, "logfood", out.Profiles[0].Profile)
	assert.Empty(t, out.Unmanaged)

	_, stderr, err := runMcpCmd(t, "list", "--profile", "typo")
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "No Databricks MCP servers installed by this CLI on profile typo.")
}

// TestMcpListJSONReportsStateError checks that a state file that cannot be read is a JSON document with the error,
// as remove writes it.
func TestMcpListJSONReportsStateError(t *testing.T) {
	withHome(t)
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte("{not json"), 0o600))

	stdout, _, err := runMcpCmd(t, "list", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	var out mcpListOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	assert.NotEmpty(t, out.Error)
	assert.Equal(t, protos.AitoolsErrorCategoryStateInvalid, out.ErrorCategory)
	assert.Empty(t, out.Profiles)
}

// TestBuildMcpListWarnsOnceForUnresolvableConfigFile checks that an entry whose config file cannot be resolved gets one
// warning, for the server, not another for each agent it is installed in. The name has a NUL byte, which no file system
// accepts, on every platform.
func TestBuildMcpListWarnsOnceForUnresolvableConfigFile(t *testing.T) {
	withHome(t)
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	entry.ConfigFile = "unresolvable\x00.cfg"
	saveState(t, entry)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	var warnings []string
	for _, w := range out.Warnings {
		if strings.Contains(w, "genie_one_mcp-logfood") {
			warnings = append(warnings, w)
		}
	}
	require.Len(t, warnings, 1, "one cause, one warning: %v", out.Warnings)
	assert.Contains(t, warnings[0], "MCP server genie_one_mcp-logfood: ")
	require.Len(t, out.Profiles, 1)
	assert.Equal(t, statusUnknown, out.Profiles[0].Servers[0].Status)
	assert.Equal(t, statusUnknown, out.Profiles[0].Servers[0].Agents[0].Status)
}

// TestRenderMcpListLoginHintNamesCheckedFile checks the login hint of an unpinned server whose profile is missing from
// ~/.databrickscfg. The entry is checked against that file, so the hint names it when DATABRICKS_CONFIG_FILE names
// another file: login would otherwise write the profile to the environment's file.
func TestRenderMcpListLoginHintNamesCheckedFile(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")
	claude := agents.ByName("claude-code")
	checked, err := mcp.ResolveConfigFile(t.Context(), "")
	require.NoError(t, err)
	shownChecked := filepath.ToSlash(checked)
	saveState(t, seedInstallEntry(t, claude, seed{"github-logfood", "system.ai.github", "logfood", "https://adb-1.test", testDatabricksCommand, ""}))

	cases := []struct {
		name, environment, login string
	}{
		{
			name:        "environment names another file",
			environment: filepath.Join(t.TempDir(), "other.cfg"),
			login:       "`DATABRICKS_CONFIG_FILE=" + shellquote.BashArg(shownChecked) + " databricks auth login --host https://adb-1.test --profile logfood`",
		},
		{
			name:        "environment names no file",
			environment: "",
			login:       "`databricks auth login --host https://adb-1.test --profile logfood`",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABRICKS_CONFIG_FILE", tc.environment)
			out, err := buildMcpList(t.Context(), "")
			require.NoError(t, err)
			ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
			renderMcpList(ctx, out, "")
			assert.Contains(t, stderr.String(), "or "+tc.login)
		})
	}
}

// TestRenderMcpListPrintsWarningsWhenEmpty checks that a listing with no server and no unmanaged entry still prints its
// warnings in text, as JSON does, for a profile with no server and for no profile.
func TestRenderMcpListPrintsWarningsWhenEmpty(t *testing.T) {
	warning := "could not read the Codex CLI config: broken"
	for _, profileName := range []string{"typo", ""} {
		out := mcpListOutput{Profiles: []mcpListProfile{}, Unmanaged: []mcpListUnmanaged{}, Warnings: []string{warning}}
		ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
		renderMcpList(ctx, out, profileName)
		assert.Contains(t, stderr.String(), "Warning: "+warning, "profile %q", profileName)
	}
}

// TestBuildMcpListWithProfileReadsOnlyItsAgents checks that --profile reads the configs of the agents the profile's
// servers are recorded for, so a broken config of an agent that the profile does not use is not warned about.
func TestBuildMcpListWithProfileReadsOnlyItsAgents(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".codex", "config.toml"), []byte("[mcp_servers.broken\n"), 0o600))
	saveState(t, seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test"))

	out, err := buildMcpList(t.Context(), "logfood")
	require.NoError(t, err)
	assert.Empty(t, out.Warnings, "codex is not used by logfood, so its config is not read")
	require.Len(t, out.Profiles, 1)

	all, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, all.Warnings, 1)
	assert.Contains(t, all.Warnings[0], "Codex CLI")
}

// TestBuildMcpListWarnsInStableOrderForUnknownAgents checks that the warnings for two records of agents outside the
// registry come in name order, not in the order a map yields them. The map is iterated many times, so an unstable
// order would show.
func TestBuildMcpListWarnsInStableOrderForUnknownAgents(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "")
	for range 20 {
		saveState(t, mcp.Entry{
			Name: "genie_one_mcp-logfood", Service: "system.ai.genie_one_mcp", Profile: "logfood", Host: "https://adb-1.test",
			Command: testDatabricksCommand, Agents: map[string]mcp.AgentRecord{
				"zeta-agent":  {Path: "/x/zeta.json", Scope: "user"},
				"alpha-agent": {Path: "/x/alpha.json", Scope: "user"},
			},
		})
		out, err := buildMcpList(t.Context(), "")
		require.NoError(t, err)
		require.Len(t, out.Warnings, 2)
		assert.Equal(t, "could not read the alpha-agent config: alpha-agent is not a known agent", out.Warnings[0])
		assert.Equal(t, "could not read the zeta-agent config: zeta-agent is not a known agent", out.Warnings[1])
	}
}

// TestRenderMcpListSaysHowToClearChangedEntry checks that a server with an entry changed outside this CLI says in list how
// to get it out, the same way remove says it.
func TestRenderMcpListSaysHowToClearChangedEntry(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	claudePath := filepath.Join(home, ".claude.json")
	repointEntry(t, claudePath, entry.Name, "/usr/bin/other")

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	renderMcpList(ctx, out, "")
	assert.Contains(t, stderr.String(), "  changed outside this CLI in Claude Code: delete the entry from "+filepath.ToSlash(claudePath)+
		" by hand and run `databricks aitools mcp remove --names genie_one_mcp-logfood` again, or add the server again with --force\n")
}

// TestProfileHeaderNamesPinnedConfigFile checks that two groups of one profile and host, in different config files, have
// different headers: a pinned group names its file.
func TestProfileHeaderNamesPinnedConfigFile(t *testing.T) {
	first := mcpListProfile{Profile: "logfood", Host: "https://adb-1.test", ConfigFile: "/cfg/first.cfg", Status: statusOK}
	second := mcpListProfile{Profile: "logfood", Host: "https://adb-1.test", ConfigFile: "/cfg/second.cfg", Status: statusOK}
	unpinned := mcpListProfile{Profile: "logfood", Host: "https://adb-1.test", Status: statusOK}

	assert.Equal(t, "Profile logfood  (https://adb-1.test, config file /cfg/first.cfg)", profileHeader(first))
	assert.NotEqual(t, profileHeader(first), profileHeader(second))
	assert.Equal(t, "Profile logfood  (https://adb-1.test)", profileHeader(unpinned))
}

// TestRenderMcpListHintForDatabricksEntryOmitsForce checks the hint for an entry this CLI wrote that serves another
// binding now: add refuses it whatever --force says, so the hint says to delete it by hand and does not offer --force.
func TestRenderMcpListHintForDatabricksEntryOmitsForce(t *testing.T) {
	home := withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
	saveState(t, entry)
	claudePath := filepath.Join(home, ".claude.json")
	editEntry(t, claudePath, entry.Name, func(server map[string]any) { setServeFlag(t, server, "--profile", "other") })

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	renderMcpList(ctx, out, "")
	assert.Contains(t, stderr.String(), "  changed outside this CLI in Claude Code: delete the entry from "+filepath.ToSlash(claudePath)+
		" by hand and run `databricks aitools mcp remove --names genie_one_mcp-logfood` again\n")
	assert.NotContains(t, stderr.String(), "--force")
}

// TestBuildMcpListWarnsOncePerUnreadableProfileFile checks that a config file which cannot be read is one warning, however
// many servers read it. Both servers here read the default file.
func TestBuildMcpListWarnsOncePerUnreadableProfileFile(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".databrickscfg"), []byte("[broken\n"), 0o600))
	saveState(t,
		mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand},
		mcp.Entry{Name: "github-dogfood", Service: "system.ai.github", Profile: "dogfood", Host: "https://adb-2.test", Command: testDatabricksCommand},
	)

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	var warnings []string
	for _, w := range out.Warnings {
		if strings.Contains(w, "failed to read Databricks profiles") {
			warnings = append(warnings, w)
		}
	}
	require.Len(t, warnings, 1, "one file, one warning: %v", out.Warnings)
	assert.NotContains(t, warnings[0], "genie", "the warning is for the file, not for a server")
	for _, p := range out.Profiles {
		assert.Equal(t, statusUnknown, p.Status, "every profile of the file is unknown")
	}
}

// TestBuildMcpListWarnsForOwnershipErrorWithAnotherCause checks that an ownership error is warned about when the entry
// check failed for another reason. Here the profile file cannot be read, and the agent's entry pins a file whose name
// cannot be resolved: two causes, so two warnings.
func TestBuildMcpListWarnsForOwnershipErrorWithAnotherCause(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".databrickscfg"), []byte("[broken\n"), 0o600))
	claude := agents.ByName("claude-code")
	entry := seedInstall(t, claude, "github-logfood", "system.ai.github", "logfood", "https://adb-1.test")
	saveState(t, entry)
	claudePath := filepath.Join(home, ".claude.json")
	editEntry(t, claudePath, entry.Name, func(server map[string]any) {
		server["env"] = map[string]any{mcpconfig.ConfigFileEnv: "unresolvable\x00.cfg"}
	})

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	var ownership []string
	for _, w := range out.Warnings {
		if strings.Contains(w, "MCP server github-logfood: could not check Claude Code") {
			ownership = append(ownership, w)
		}
	}
	require.Len(t, ownership, 1, "the agent's check has its own cause: %v", out.Warnings)
	assert.Equal(t, statusUnknown, out.Profiles[0].Servers[0].Agents[0].Status)
}

// TestBuildMcpListRecordForAgentWithoutMCPIsUnknown checks a record for a registry agent that takes no MCP servers. It
// is not a config that failed to read: the warning says the agent takes no servers, and remove drops the record.
func TestBuildMcpListRecordForAgentWithoutMCPIsUnknown(t *testing.T) {
	withHome(t)
	writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
	pi := agents.ByName("pi")
	require.NotNil(t, pi)
	require.False(t, pi.SupportsMCP())
	saveState(t, mcp.Entry{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-1.test", Command: testDatabricksCommand,
		Agents: map[string]mcp.AgentRecord{pi.Name: {Path: "/x/pi.json", Scope: "user"}},
	})

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	assert.Equal(t, []string{"MCP server github-logfood: Pi does not take MCP servers; remove drops this record"}, out.Warnings)
	require.Len(t, out.Profiles, 1)
	require.Len(t, out.Profiles[0].Servers, 1)
	assert.Equal(t, statusUnknown, out.Profiles[0].Servers[0].Agents[0].Status)
}

// TestBuildMcpListAndRemoveAgreeAfterConfigMoved checks that list reports the entry where remove acts on it, when the
// agent's config directory moved. The entry is either in the file the record names, which the agent no longer reads, or
// in the file the agent reads now, with the record naming the old file.
func TestBuildMcpListAndRemoveAgreeAfterConfigMoved(t *testing.T) {
	cases := []struct {
		name string
		// entryIn is the file that holds the entry: "recorded" or "current".
		entryIn string
		status  string
		// holds is the file that keeps the entry after remove: "" when no file keeps it.
		removedFrom string
	}{
		{name: "entry in the recorded file", entryIn: "recorded", status: agentStatusPathChanged, removedFrom: "recorded"},
		{name: "entry in the file the agent reads now", entryIn: "current", status: agentStatusInstalled, removedFrom: "current"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withHome(t)
			writeDatabricksCfg(t, "[logfood]\nhost = https://adb-1.test\ntoken = t\n")
			claude := agents.ByName("claude-code")
			oldDir := t.TempDir()
			t.Setenv("CLAUDE_CONFIG_DIR", oldDir)
			entry := seedInstall(t, claude, "genie_one_mcp-logfood", "system.ai.genie_one_mcp", "logfood", "https://adb-1.test")
			oldPath := entry.Agents[claude.Name].Path
			saveState(t, entry)

			newDir := t.TempDir()
			newPath := filepath.Join(newDir, ".claude.json")
			t.Setenv("CLAUDE_CONFIG_DIR", newDir)
			if tc.entryIn == "current" {
				// The user moved the config with the directory: the entry is in the new file and the old one is gone.
				data, err := os.ReadFile(oldPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(newPath, data, 0o600))
				require.NoError(t, os.Remove(oldPath))
			} else {
				require.NoError(t, os.WriteFile(newPath, []byte("{}"), 0o600))
			}

			out, err := buildMcpList(t.Context(), "")
			require.NoError(t, err)
			require.Len(t, out.Profiles, 1)
			listed := out.Profiles[0].Servers[0].Agents[0]
			assert.Equal(t, tc.status, listed.Status)
			wantPath := filepath.ToSlash(oldPath)
			if tc.removedFrom == "current" {
				wantPath = filepath.ToSlash(newPath)
			}
			assert.Equal(t, wantPath, listed.Path, "list names the file remove acts on")

			stdout, _, err := runMcpCmd(t, "remove", "--names", entry.Name, "--output", "json")
			require.NoError(t, err)
			removed := readRemoveOutput(t, stdout.String())
			require.Len(t, removed.Removed, 1, "remove takes the entry from the file list named")
			assert.Empty(t, removed.Skipped)

			acted := oldPath
			if tc.removedFrom == "current" {
				acted = newPath
			}
			if data, err := os.ReadFile(acted); err == nil {
				assert.NotContains(t, string(data), entry.Name, "the entry is gone from %s", acted)
			}
		})
	}
}

// listCheckServer answers the GitHub service and the caller's sign-in to it, for --check. The service answers with
// serviceStatus, and the credential with credentialStatus and credentialBody. Other API paths are NOT_FOUND.
func listCheckServer(t *testing.T, serviceStatus, credentialStatus int, credentialBody string) *httptest.Server {
	t.Helper()
	const servicePath = "/api/2.1/unity-catalog/mcp-services/system.ai.github"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case servicePath:
			if serviceStatus != http.StatusOK {
				w.WriteHeader(serviceStatus)
				_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not found"}`))
				return
			}
			_, _ = w.Write([]byte(`{"name":"mcp-services/system.ai.github","config":{"source_connection":{"options":{"oauth_provider":"OAUTH_PROVIDER_GITHUB_MCP"}}}}`))
		case servicePath + "/user-credentials":
			w.WriteHeader(credentialStatus)
			_, _ = w.Write([]byte(credentialBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// listCheckGitHub returns the state record of a sign-in-required GitHub server for the profile at host, pinned to
// configFile when it is set.
func listCheckGitHub(t *testing.T, name, profileName, host, configFile string) mcp.Entry {
	t.Helper()
	entry := seedInstallEntry(t, agents.ByName("claude-code"), seed{name, "system.ai.github", profileName, host, testDatabricksCommand, configFile})
	entry.SignInRequired = true
	return entry
}

// checkedListServer runs buildMcpList and the --check tests, and returns the one server listed for the profile.
func checkedListServer(t *testing.T) (mcpListServer, mcpListOutput) {
	t.Helper()
	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	addChecks(t.Context(), &out)
	require.Len(t, out.Profiles, 1)
	require.Len(t, out.Profiles[0].Servers, 1)
	return out.Profiles[0].Servers[0], out
}

// TestAddChecksWarnsWhenTheSignInCannotBeRead checks that a sign-in whose state cannot be read is unknown, and that the
// failure is a warning with the login command, not silently dropped.
func TestAddChecksWarnsWhenTheSignInCannotBeRead(t *testing.T) {
	home := withHome(t)
	server := listCheckServer(t, http.StatusOK, http.StatusUnauthorized, `{"error_code":"UNAUTHENTICATED","message":"Invalid access token."}`)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	saveState(t, listCheckGitHub(t, "github-logfood", "logfood", server.URL, ""))

	s, out := checkedListServer(t)
	require.NotNil(t, s.Check)
	assert.Equal(t, statusOK, s.Check.Token)
	assert.Equal(t, statusUnknown, s.Check.SignIn)
	require.Len(t, out.Warnings, 1)
	assert.Contains(t, out.Warnings[0], "MCP server github-logfood: ")
	assert.Contains(t, out.Warnings[0], "Run: databricks auth login --profile logfood")
}

// TestAddChecksServiceGoneIsAWarningNotSignInNeeded checks that a service the profile no longer sees is a warning, and that
// its sign-in is unknown rather than needed, since the sign-in is not what is missing.
func TestAddChecksServiceGoneIsAWarningNotSignInNeeded(t *testing.T) {
	home := withHome(t)
	server := listCheckServer(t, http.StatusNotFound, http.StatusNotFound, `{"error_code":"NOT_FOUND","message":"not found"}`)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	saveState(t, listCheckGitHub(t, "github-logfood", "logfood", server.URL, ""))

	s, out := checkedListServer(t)
	require.NotNil(t, s.Check)
	assert.Equal(t, statusUnknown, s.Check.SignIn)
	require.Len(t, out.Warnings, 1)
	assert.Equal(t, "MCP server github-logfood: system.ai.github is not an MCP service visible to profile logfood", out.Warnings[0])
}

// TestAddChecksWarnsWhenProfileCannotAuthenticate checks that a profile whose credential cannot be read is a warning
// naming the profile, and that its servers' sign-ins are not checked.
func TestAddChecksWarnsWhenProfileCannotAuthenticate(t *testing.T) {
	home := withHome(t)
	server := listCheckServer(t, http.StatusOK, http.StatusOK, `{}`)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\n")
	saveState(t, listCheckGitHub(t, "github-logfood", "logfood", server.URL, ""))

	s, out := checkedListServer(t)
	require.NotNil(t, s.Check)
	assert.Equal(t, checkFailed, s.Check.Token)
	assert.Equal(t, statusUnknown, s.Check.SignIn)
	require.Len(t, out.Warnings, 1)
	assert.True(t, strings.HasPrefix(out.Warnings[0], "profile logfood: "), out.Warnings[0])
}

// TestAddChecksReadsEachProfileFromItsOwnConfigFile checks that two groups of one profile name, in two config files, are
// checked with their own file's credentials, and that the config file environment is the caller's again afterwards.
func TestAddChecksReadsEachProfileFromItsOwnConfigFile(t *testing.T) {
	withHome(t)
	signedIn := listCheckServer(t, http.StatusOK, http.StatusOK, `{"provisioning_info":{"state":"ACTIVE"}}`)
	notSignedIn := listCheckServer(t, http.StatusOK, http.StatusNotFound, `{"error_code":"NOT_FOUND","message":"not found"}`)
	dir := t.TempDir()
	first := filepath.Join(dir, "first.cfg")
	second := filepath.Join(dir, "second.cfg")
	require.NoError(t, os.WriteFile(first, []byte("[logfood]\nhost = "+signedIn.URL+"\nauth_type = pat\ntoken = dapi-first\n"), 0o600))
	require.NoError(t, os.WriteFile(second, []byte("[logfood]\nhost = "+notSignedIn.URL+"\nauth_type = pat\ntoken = dapi-second\n"), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", first)
	saveState(t, listCheckGitHub(t, "github-first", "logfood", signedIn.URL, first), listCheckGitHub(t, "github-second", "logfood", notSignedIn.URL, second))

	out, err := buildMcpList(t.Context(), "")
	require.NoError(t, err)
	addChecks(t.Context(), &out)
	require.Len(t, out.Profiles, 2)
	for _, p := range out.Profiles {
		require.Len(t, p.Servers, 1)
		assert.Equal(t, statusOK, p.Servers[0].Check.Token)
	}
	signInByServer := map[string]string{}
	for _, p := range out.Profiles {
		signInByServer[p.Servers[0].Name] = p.Servers[0].Check.SignIn
	}
	assert.Equal(t, map[string]string{"github-first": statusOK, "github-second": checkNeeded}, signInByServer)
	assert.Equal(t, first, os.Getenv("DATABRICKS_CONFIG_FILE"))
	assert.Empty(t, out.Warnings)
}

// TestWithConfigFileSetsTheEnvironmentForTheCallAndRestoresIt checks that the config file of a profile is set in the
// context and in the process environment while a check runs, and that the caller's environment is restored after.
func TestWithConfigFileSetsTheEnvironmentForTheCallAndRestoresIt(t *testing.T) {
	t.Setenv(mcpconfig.ConfigFileEnv, "ambient.cfg")
	var inContext, inProcess string
	err := withConfigFile(t.Context(), "pinned.cfg", func(ctx context.Context) {
		inContext = env.Get(ctx, mcpconfig.ConfigFileEnv)
		inProcess = os.Getenv(mcpconfig.ConfigFileEnv)
	})
	require.NoError(t, err)
	assert.Equal(t, "pinned.cfg", inContext)
	assert.Equal(t, "pinned.cfg", inProcess)
	assert.Equal(t, "ambient.cfg", os.Getenv(mcpconfig.ConfigFileEnv))

	require.NoError(t, os.Unsetenv(mcpconfig.ConfigFileEnv))
	require.NoError(t, withConfigFile(t.Context(), "pinned.cfg", func(context.Context) {}))
	_, set := os.LookupEnv(mcpconfig.ConfigFileEnv)
	assert.False(t, set, "a variable that was unset is unset again")
}

// TestCheckText checks the words that --check adds to a server's line.
func TestCheckText(t *testing.T) {
	cases := []struct {
		name  string
		check *mcpListCheck
		want  string
	}{
		{name: "no check", check: nil, want: ""},
		{name: "signed in and binary ok", check: &mcpListCheck{SignIn: statusOK, Binary: statusOK}, want: "signed in"},
		{name: "sign-in needed", check: &mcpListCheck{SignIn: checkNeeded, Binary: statusOK}, want: "sign-in needed"},
		{name: "sign-in unknown", check: &mcpListCheck{SignIn: statusUnknown, Binary: statusOK}, want: "sign-in unknown"},
		{name: "binary missing", check: &mcpListCheck{Binary: checkFailed}, want: "binary missing"},
		{name: "sign-in needed and binary missing", check: &mcpListCheck{SignIn: checkNeeded, Binary: checkFailed}, want: "sign-in needed, binary missing"},
		{name: "server without sign-in", check: &mcpListCheck{Binary: statusOK}, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, checkText(tc.check))
		})
	}
}

// TestBinaryStatus checks that a recorded absolute path must be on disk, and that a bare name is looked up on PATH.
func TestBinaryStatus(t *testing.T) {
	present := filepath.Join(t.TempDir(), "databricks")
	require.NoError(t, os.WriteFile(present, []byte("binary"), 0o700))
	missing := filepath.Join(t.TempDir(), "databricks")
	found := func(name string) (string, error) {
		if name == "databricks" {
			return present, nil
		}
		return "", exec.ErrNotFound
	}
	notFound := func(string) (string, error) { return "", exec.ErrNotFound }

	assert.Equal(t, statusOK, binaryStatus(present, notFound))
	assert.Equal(t, checkFailed, binaryStatus(missing, found))
	assert.Equal(t, statusOK, binaryStatus("databricks", found))
	assert.Equal(t, checkFailed, binaryStatus("databricks", notFound))
}

// TestSignInStatus checks the sign-in state that --check reports for one service.
func TestSignInStatus(t *testing.T) {
	cases := []struct {
		name       string
		serviceOK  bool
		credential int
		body       string
		want       string
		warns      bool
	}{
		{name: "active", serviceOK: true, credential: http.StatusOK, body: `{"provisioning_info":{"state":"ACTIVE"}}`, want: statusOK},
		{name: "provisioning", serviceOK: true, credential: http.StatusOK, body: `{"provisioning_info":{"state":"PROVISIONING"}}`, want: checkNeeded},
		{name: "not signed in", serviceOK: true, credential: http.StatusNotFound, body: `{"error_code":"NOT_FOUND","message":"not found"}`, want: checkNeeded},
		{name: "rejected token", serviceOK: true, credential: http.StatusUnauthorized, body: `{"error_code":"UNAUTHENTICATED","message":"Invalid access token."}`, want: statusUnknown, warns: true},
		{name: "service gone", serviceOK: false, credential: http.StatusNotFound, body: `{"error_code":"NOT_FOUND","message":"not found"}`, want: statusUnknown, warns: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			serviceStatus := http.StatusOK
			if !tc.serviceOK {
				serviceStatus = http.StatusNotFound
			}
			server := listCheckServer(t, serviceStatus, tc.credential, tc.body)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
			require.NoError(t, err)

			status, err := signInStatus(t.Context(), w, "logfood", "system.ai.github")
			assert.Equal(t, tc.want, status)
			if tc.warns {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestAddChecksUsesTheRequirementTheServiceHasNow checks that --check asks the service whether it needs sign-in, rather
// than trusting the record: a service that now needs sign-in is checked, and one that no longer needs it is not reported
// as needing sign-in.
func TestAddChecksUsesTheRequirementTheServiceHasNow(t *testing.T) {
	t.Run("now requires sign-in", func(t *testing.T) {
		home := withHome(t)
		server := listCheckServer(t, http.StatusOK, http.StatusNotFound, `{"error_code":"NOT_FOUND","message":"not found"}`)
		writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
		entry := listCheckGitHub(t, "github-logfood", "logfood", server.URL, "")
		entry.SignInRequired = false
		saveState(t, entry)

		s, out := checkedListServer(t)
		require.NotNil(t, s.Check)
		assert.Equal(t, checkNeeded, s.Check.SignIn)
		assert.Empty(t, out.Warnings)
	})

	t.Run("no longer requires sign-in", func(t *testing.T) {
		home := withHome(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/2.1/unity-catalog/mcp-services/system.ai.github":
				// The service has no source connection with an OAuth provider any more.
				_, _ = w.Write([]byte(`{"name":"mcp-services/system.ai.github"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(server.Close)
		writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
		saveState(t, listCheckGitHub(t, "github-logfood", "logfood", server.URL, ""))

		s, out := checkedListServer(t)
		require.NotNil(t, s.Check)
		assert.Empty(t, s.Check.SignIn)
		assert.Empty(t, out.Warnings)
	})
}
