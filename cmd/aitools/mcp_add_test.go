package aitools

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMCPAddFlags(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cases := []struct {
		name       string
		flags      mcpAddFlags
		names      []string
		agentNames []string
		jsonMode   bool
		want       string
	}{
		{name: "text with names only is valid", names: []string{"github"}},
		{name: "text without names", want: "--names is required so the command runs without interactive prompts"},
		{
			name: "json needs agents", names: []string{"github"}, jsonMode: true,
			want: "--output json requires --agents so the command runs without interactive prompts",
		},
		{
			name: "json needs names and agents", jsonMode: true,
			want: "--output json requires --names and --agents so the command runs without interactive prompts",
		},
		{name: "json with both", names: []string{"github"}, agentNames: []string{"codex"}, jsonMode: true},
		{
			name: "name needs one service", flags: mcpAddFlags{name: "x"}, names: []string{"a", "b"},
			want: "--name needs exactly one service in --names",
		},
		{
			name: "name must be a valid server name", flags: mcpAddFlags{name: "a.b"}, names: []string{"github"},
			want: `--name "a.b" may only contain letters, digits, '_' and '-'`,
		},
		{name: "valid name", flags: mcpAddFlags{name: "my_github-1"}, names: []string{"github"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMCPAddFlags(ctx, tc.flags, tc.names, tc.agentNames, tc.jsonMode)
			if tc.want == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.want, err.Error())
			assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
		})
	}
}

func TestValidateLocation(t *testing.T) {
	assert.NoError(t, validateLocation("system.ai"))
	assert.NoError(t, validateLocation("Main.Tools"))
	for _, bad := range []string{"system", "system.", ".ai", "a.b.c"} {
		err := validateLocation(bad)
		require.Error(t, err, bad)
		assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
	}
}

func testServices() []mcp.Service {
	return []mcp.Service{
		{FullName: "system.ai.genie_one_mcp", Leaf: "genie_one_mcp"},
		{FullName: "system.ai.github", Leaf: "github", SignInRequired: true},
	}
}

func TestPickServices(t *testing.T) {
	services := testServices()

	picked, err := pickServices("system.ai", []string{"github", "system.ai.genie_one_mcp", "github"}, services)
	require.NoError(t, err)
	require.Len(t, picked, 2)
	assert.Equal(t, "system.ai.github", picked[0].FullName)
	assert.Equal(t, "system.ai.genie_one_mcp", picked[1].FullName)

	picked, err = pickServices("System.AI", []string{"GitHub"}, services)
	require.NoError(t, err)
	assert.Equal(t, "system.ai.github", picked[0].FullName)

	_, err = pickServices("system.ai", []string{"system.ai.foo"}, services)
	require.Error(t, err)
	assert.Equal(t, "system.ai.foo is not an MCP service in system.ai. Available: system.ai.genie_one_mcp, system.ai.github", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategory("MCP_SERVICE_NOT_FOUND"), errorCategory(err))

	_, err = pickServices("system.ai", []string{"main.tools.crm"}, services)
	require.Error(t, err)
	assert.Equal(t, "main.tools.crm is not in system.ai; pass --location to match its schema", err.Error())
}

func TestResolveMCPAgents(t *testing.T) {
	got, err := resolveMCPAgents([]string{"codex", "claude-code", "codex"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "codex", got[0].Name)
	assert.Equal(t, "claude-code", got[1].Name)

	_, err = resolveMCPAgents([]string{"nope"})
	require.Error(t, err)
	assert.Equal(t, `unknown agent "nope". Supported: claude-code, cursor, codex, opencode, copilot, antigravity, gemini`, err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, errorCategory(err))

	_, err = resolveMCPAgents([]string{"pi"})
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "pi does not support MCP servers. Supported: "), err.Error())
}

func TestDescribeListErrorNamesProfileAndKeepsCause(t *testing.T) {
	cause := errors.New("403 Forbidden")
	listErr := &mcp.ListError{Kind: mcp.ListErrPermissionDenied, Location: "main.tools", Err: cause}

	err := describeListError("logfood", listErr)
	assert.Equal(t, "profile logfood: "+listErr.Error(), err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, errorCategory(err))

	got, ok := errors.AsType[*mcp.ListError](err)
	require.True(t, ok)
	assert.Equal(t, mcp.ListErrPermissionDenied, got.Kind)
	assert.ErrorIs(t, err, cause)

	rateLimited := describeListError("logfood", &mcp.ListError{Kind: mcp.ListErrRateLimited, Location: "system.ai", Err: cause})
	assert.Equal(t, protos.AitoolsErrorCategoryRateLimited, errorCategory(rateLimited))

	plain := errors.New("network down")
	assert.Equal(t, plain, describeListError("logfood", plain))
}

// withHome points the home directory at a temp dir, so agent config paths are inside it.
func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CODEX_HOME", "")
	return home
}

func TestPlanAddsNaming(t *testing.T) {
	ctx := t.Context()
	claude := agents.ByName("claude-code")
	host := "https://adb-123.test"

	t.Run("new service gets the generated name", func(t *testing.T) {
		withHome(t)
		plans, err := planAdds(ctx, "", "system.ai", "logfood", host, &mcp.State{}, testServices()[:1], []*agents.Agent{claude})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Equal(t, "genie_one_mcp-logfood", plans[0].name)
		assert.Nil(t, plans[0].entry)
	})

	t.Run("existing install keeps its name", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "custom-name", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		plans, err := planAdds(ctx, "", "system.ai", "logfood", host, state, testServices()[1:], []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "custom-name", plans[0].name)
		assert.NotNil(t, plans[0].entry)
	})

	t.Run("name used by another state entry gets a suffix", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "genie_one_mcp-logfood", Service: "main.tools.other", Profile: "logfood", Host: host}}}
		plans, err := planAdds(ctx, "", "system.ai", "logfood", host, state, testServices()[:1], []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "genie_one_mcp-logfood-2", plans[0].name)
	})

	t.Run("unmanaged entry in an agent file gets a suffix", func(t *testing.T) {
		home := withHome(t)
		unmanaged := `{"mcpServers":{"genie_one_mcp-logfood":{"type":"stdio","command":"ug","args":["mcp-proxy"]}}}`
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(unmanaged), 0o600))
		plans, err := planAdds(ctx, "", "system.ai", "logfood", host, &mcp.State{}, testServices()[:1], []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "genie_one_mcp-logfood-2", plans[0].name)
	})

	t.Run("name flag for an installed service must match", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		_, err := planAdds(ctx, "other", "system.ai", "logfood", host, state, testServices()[1:], []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, "system.ai.github is already installed on profile logfood as github-logfood; remove it before using --name other", err.Error())
	})

	t.Run("name flag already used by another service is refused", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "taken", Service: "main.tools.other", Profile: "logfood", Host: host}}}
		_, err := planAdds(ctx, "taken", "system.ai", "logfood", host, state, testServices()[:1], []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, "--name taken is used by another installed server; pick a different name", err.Error())
	})

	t.Run("host change is refused, never rebound", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://old.test"}}}
		_, err := planAdds(ctx, "", "system.ai", "logfood", host, state, testServices()[1:], []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, protos.AitoolsErrorCategory("PROFILE_HOST_CHANGED"), errorCategory(err))
		assert.Contains(t, err.Error(), "was set up for https://old.test, but profile \"logfood\" now points at "+host)
	})
}

func TestPinnedConfigFile(t *testing.T) {
	ctx := t.Context()
	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	got, err := pinnedConfigFile(ctx)
	require.NoError(t, err)
	assert.Empty(t, got)

	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join("rel", ".databrickscfg"))
	got, err = pinnedConfigFile(ctx)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got), got)
}

func TestCursorApprovalHintNamesBinary(t *testing.T) {
	assert.Equal(t,
		"Cursor may ask you to approve github-logfood. Run: cursor-agent mcp enable github-logfood (the Cursor docs call the CLI agent), or approve it in Cursor's MCP settings.",
		cursorApprovalHint("github-logfood"))
}

// newAddTestCmd builds `aitools mcp add` under a root with the --output flag, as root does.
func newAddTestCmd(t *testing.T, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	parent := &cobra.Command{Use: "databricks", SilenceErrors: true, SilenceUsage: true}
	output := flags.OutputText
	parent.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	parent.AddCommand(NewMcpCmd())

	var stdout bytes.Buffer
	parent.SetOut(&stdout)
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs(append([]string{"mcp", "add"}, args...))
	return &stdout, parent.ExecuteContext(t.Context())
}

func TestMcpAddJSONReportsMissingFlagsAsDocument(t *testing.T) {
	withHome(t)
	stdout, err := newAddTestCmd(t, "--names", "github", "--output", "json")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var out addOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	assert.Equal(t, "INVALID_FLAGS", string(out.ErrorCategory))
	assert.Equal(t, "--output json requires --agents so the command runs without interactive prompts", out.Error)
	assert.Empty(t, out.Servers)
	assert.Equal(t, "global", out.Scope)
}
