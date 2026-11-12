package aitools

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMCPRemoveFlags(t *testing.T) {
	assert.NoError(t, validateMCPRemoveFlags(false, []string{"github-logfood"}, ""))
	assert.NoError(t, validateMCPRemoveFlags(false, nil, "logfood"))
	assert.NoError(t, validateMCPRemoveFlags(true, nil, ""))
	assert.NoError(t, validateMCPRemoveFlags(true, nil, "logfood"))

	err := validateMCPRemoveFlags(false, nil, "")
	require.Error(t, err)
	assert.Equal(t, "pass --names, --profile or --all; interactive selection is not available yet", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))

	err = validateMCPRemoveFlags(true, []string{"github-logfood"}, "")
	require.Error(t, err)
	assert.Equal(t, "--all cannot be combined with --names", err.Error())
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
	assert.Equal(t, protos.AitoolsErrorCategory("MCP_SERVICE_NOT_FOUND"), errorCategory(err))
}

// newRemoveTestCmd runs `mcp remove` with args and returns its stdout.
func newRemoveTestCmd(t *testing.T, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	parent := &cobra.Command{Use: "databricks", SilenceErrors: true, SilenceUsage: true}
	output := flags.OutputText
	parent.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	parent.PersistentFlags().StringP("profile", "p", "", "profile")
	parent.AddCommand(NewMcpCmd())

	var stdout bytes.Buffer
	parent.SetOut(&stdout)
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs(append([]string{"mcp", "remove"}, args...))
	return &stdout, parent.ExecuteContext(t.Context())
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	return doc
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

	stdout, err := newRemoveTestCmd(t, "--names", "genie_one_mcp-logfood,dbsql-logfood", "--output", "json")
	require.NoError(t, err)

	var out removeOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
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
	_, _, err := agentconfig.Upsert(t.Context(), claude, "genie_one_mcp-logfood", "/usr/bin/other", []string{"serve"}, nil, true)
	require.NoError(t, err)
	before, err := os.ReadFile(claudePath)
	require.NoError(t, err)

	stdout, err := newRemoveTestCmd(t, "--names", "genie_one_mcp-logfood", "--output", "json")
	require.NoError(t, err)

	var out removeOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	assert.Empty(t, out.Removed)
	require.Len(t, out.Skipped, 1)
	assert.Equal(t, "genie_one_mcp-logfood", out.Skipped[0].Name)

	after, err := os.ReadFile(claudePath)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the agent config is not written")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Contains(t, state.Servers[0].Agents, "claude-code", "the record stays, so list keeps showing the entry")
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

	stdout, err := newRemoveTestCmd(t, "--names", "genie_one_mcp-logfood", "--agents", "claude-code,codex", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var out removeOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	require.Len(t, out.Removed, 1)
	assert.Equal(t, []string{"claude-code"}, out.Removed[0].Agents)
	require.Len(t, out.Skipped, 1)
	assert.Contains(t, out.Skipped[0].Reason, "could not update Codex CLI")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, []string{"codex"}, keysOf(state.Servers[0].Agents))
}

func TestMcpRemoveFailsWithoutSelection(t *testing.T) {
	withHome(t)
	stdout, err := newRemoveTestCmd(t, "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var out removeOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, out.ErrorCategory)
	assert.Equal(t, "pass --names, --profile or --all; interactive selection is not available yet", out.Error)
	assert.Empty(t, out.Removed)
	assert.Empty(t, out.Skipped)
}

func keysOf(m map[string]mcp.AgentRecord) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
