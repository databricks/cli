package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
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
			want: `--name "a.b" is not a valid server name: use letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{
			name: "name must not contain a double underscore", flags: mcpAddFlags{name: "a__b"}, names: []string{"github"},
			want: `--name "a__b" is not a valid server name: use letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{
			name: "name must not start with an underscore", flags: mcpAddFlags{name: "_a"}, names: []string{"github"},
			want: `--name "_a" is not a valid server name: use letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{
			name: "name must not end with a dash", flags: mcpAddFlags{name: "a-"}, names: []string{"github"},
			want: `--name "a-" is not a valid server name: use letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{name: "valid name", flags: mcpAddFlags{name: "my_github-1"}, names: []string{"github"}},
		{name: "valid name in mixed case", flags: mcpAddFlags{name: "GitHub"}, names: []string{"github"}},
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

func TestServiceNames(t *testing.T) {
	got, err := serviceNames("system.ai", []string{"github", "System.AI.GitHub", "system.ai.genie_one_mcp"})
	require.NoError(t, err)
	assert.Equal(t, []string{"system.ai.github", "System.AI.GitHub", "system.ai.genie_one_mcp"}, got)

	_, err = serviceNames("system.ai", []string{"main.tools.crm"})
	require.Error(t, err)
	assert.Equal(t, "main.tools.crm is not in system.ai; pass --location to match its schema", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))

	// A two-part name is neither a leaf nor a full name, so it is not read as a leaf in the location.
	_, err = serviceNames("system.ai", []string{"ai.github"})
	require.Error(t, err)
	assert.Equal(t, "ai.github is not a service name: use a leaf such as github or a full name such as system.ai.github", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
}

func TestPickServices(t *testing.T) {
	services := testServices()

	picked, err := pickServices("system.ai", []string{"system.ai.github", "system.ai.genie_one_mcp", "system.ai.github"}, services)
	require.NoError(t, err)
	require.Len(t, picked, 2)
	assert.Equal(t, "system.ai.github", picked[0].FullName)
	assert.Equal(t, "system.ai.genie_one_mcp", picked[1].FullName)

	picked, err = pickServices("System.AI", []string{"System.AI.GitHub"}, services)
	require.NoError(t, err)
	assert.Equal(t, "system.ai.github", picked[0].FullName)

	_, err = pickServices("system.ai", []string{"system.ai.foo"}, services)
	require.Error(t, err)
	assert.Equal(t, "system.ai.foo is not an MCP service in system.ai. Available: system.ai.genie_one_mcp, system.ai.github", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategory("MCP_SERVICE_NOT_FOUND"), errorCategory(err))
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

// withHome points the home directory at a temp dir, so agent config paths are inside it. The
// agent home overrides are cleared, so a developer's own settings do not reach the test.
func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	for _, name := range agentHomeEnv {
		t.Setenv(name, "")
	}
	return home
}

// agentHomeEnv are the variables that move an agent's config directory.
var agentHomeEnv = []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME", "GEMINI_CLI_HOME", "XDG_CONFIG_HOME", "APPDATA", "DATABRICKS_CONFIG_FILE"}

// withDatabricksBinary makes os.Args[0] the databricks command, as the installed CLI has it.
func withDatabricksBinary(t *testing.T) {
	t.Helper()
	saved := os.Args[0]
	os.Args[0] = "databricks"
	t.Cleanup(func() { os.Args[0] = saved })
}

// noRequestServer returns a server that fails the test on any request. A command that refuses before
// it contacts the profile's host never reaches it.
func noRequestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request before the refusal: %s %s", r.Method, r.URL)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	return server
}

// noAPIServer returns a server that fails the test on any API call. The SDK still reads host metadata
// and OIDC discovery when it configures a profile, so those paths are allowed.
func noAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		t.Errorf("unexpected API request: %s %s", r.Method, r.URL)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	return server
}

// writeHomeConfig writes the default Databricks config file, ~/.databrickscfg, under home. Entries that are
// not pinned to a file use that file.
func writeHomeConfig(t *testing.T, home, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(home, ".databrickscfg"), []byte(contents), 0o600))
}

// writeAddConfig writes a Databricks config file elsewhere and points DATABRICKS_CONFIG_FILE at it.
func writeAddConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", path)
	return path
}

// runAdd runs `aitools mcp add` with args and returns its error.
func runAdd(t *testing.T, args ...string) error {
	t.Helper()
	withDatabricksBinary(t)
	_, err := newAddTestCmd(t, args...)
	return err
}

// An agent named with --agents that is not installed is refused: its config is not written and its
// directory is not created. The agent that is installed still gets the server.
func TestInstallServerRefusesUndetectedExplicitAgent(t *testing.T) {
	ctx := t.Context()
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o700))
	req := addRequest{
		profile:  "logfood",
		explicit: true,
		agents:   []*agents.Agent{agents.ByName("codex"), agents.ByName("claude-code")},
		binary:   "databricks",
	}
	plan := addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"}

	out := installServer(ctx, req, plan)
	require.Len(t, out.Agents, 2)

	codex := out.Agents[0]
	assert.Equal(t, statusFailed, codex.Status)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, codex.ErrorCategory)
	assert.Equal(t, "Codex CLI is not installed (no "+filepath.ToSlash(filepath.Join(home, ".codex"))+"); install it first or leave it out of --agents", codex.Message)
	assert.NoDirExists(t, filepath.Join(home, ".codex"))

	assert.Equal(t, statusInstalled, out.Agents[1].Status)
	assert.NoError(t, out.Agents[1].err)
}

func TestPlanAddsNaming(t *testing.T) {
	ctx := t.Context()
	claude := agents.ByName("claude-code")
	const host = "https://adb-123.test"

	t.Run("new service gets the generated name", func(t *testing.T) {
		withHome(t)
		plans, err := planAdds(ctx, "", "logfood", &mcp.State{}, testServices()[:1], []*agents.Agent{claude})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Equal(t, "genie_one_mcp-logfood", plans[0].name)
		assert.Nil(t, plans[0].entry)
	})

	t.Run("existing install keeps its name", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "custom-name", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		plans, err := planAdds(ctx, "", "logfood", state, testServices()[1:], []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "custom-name", plans[0].name)
		assert.NotNil(t, plans[0].entry)
	})

	t.Run("name used by another state entry gets a suffix", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "genie_one_mcp-logfood", Service: "main.tools.other", Profile: "logfood", Host: host}}}
		plans, err := planAdds(ctx, "", "logfood", state, testServices()[:1], []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "genie_one_mcp-logfood-2", plans[0].name)
	})

	t.Run("unmanaged entry in an agent file gets a suffix", func(t *testing.T) {
		home := withHome(t)
		unmanaged := `{"mcpServers":{"genie_one_mcp-logfood":{"type":"stdio","command":"ug","args":["mcp-proxy"]}}}`
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(unmanaged), 0o600))
		plans, err := planAdds(ctx, "", "logfood", &mcp.State{}, testServices()[:1], []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "genie_one_mcp-logfood-2", plans[0].name)
	})

	t.Run("name flag for an installed service must match", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		_, err := planAdds(ctx, "other", "logfood", state, testServices()[1:], []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, "system.ai.github is already installed on profile logfood as github-logfood; remove it before using --name other", err.Error())
	})

	t.Run("name flag already used by another service is refused", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "taken", Service: "main.tools.other", Profile: "logfood", Host: host}}}
		_, err := planAdds(ctx, "taken", "logfood", state, testServices()[:1], []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, "--name taken is used by another installed server; pick a different name", err.Error())
	})
}

func TestAddConfigFileResolvesAsBindDoes(t *testing.T) {
	ctx := t.Context()
	home := withHome(t)

	// Unset: the default file is resolved, but nothing is pinned into the agent entries.
	path, pinned, err := addConfigFile(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".databrickscfg"), path)
	assert.Empty(t, pinned)

	// A leading "~" is the home directory, not a directory named "~" under the working directory.
	t.Setenv("DATABRICKS_CONFIG_FILE", "~/cfg/.databrickscfg")
	path, pinned, err = addConfigFile(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "cfg", ".databrickscfg"), path)
	assert.Equal(t, path, pinned)

	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join("rel", ".databrickscfg"))
	path, pinned, err = addConfigFile(ctx)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(path), path)
	assert.Equal(t, path, pinned)
}

func TestCursorApprovalHintNamesBinary(t *testing.T) {
	assert.Equal(t,
		"Cursor may ask you to approve github-logfood. Run: cursor-agent mcp enable github-logfood (the Cursor docs call the CLI agent), or approve it in Cursor's MCP settings.",
		cursorApprovalHint("github-logfood"))
}

// newAddTestRoot returns a root command with the persistent --profile and --output flags that root
// defines, and the mcp group.
func newAddTestRoot() *cobra.Command {
	parent := &cobra.Command{Use: "databricks", SilenceErrors: true, SilenceUsage: true}
	output := flags.OutputText
	parent.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	parent.PersistentFlags().StringP("profile", "p", "", "profile to use")
	parent.AddCommand(NewMcpCmd())
	return parent
}

// newAddTestCmd runs `aitools mcp add` under a root, as root does.
func newAddTestCmd(t *testing.T, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	parent := newAddTestRoot()
	var stdout bytes.Buffer
	parent.SetOut(&stdout)
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs(append([]string{"mcp", "add"}, args...))
	return &stdout, parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
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

// A profile re-pointed at another host is refused before any request reaches that host.
func TestMcpAddRefusesRepointedHostBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test",
	}}}))

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory("PROFILE_HOST_CHANGED"), errorCategory(err))
	assert.Contains(t, err.Error(), "was set up for https://adb-123.test, but profile \"logfood\" now points at "+server.URL)
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

func TestMcpAddRefusesWorkspaceChangeOnSameHostBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: server.URL, WorkspaceID: "111",
	}}}))

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory("PROFILE_HOST_CHANGED"), errorCategory(err))
	assert.Contains(t, err.Error(), "was set up for workspace 111, but profile \"logfood\" now points at workspace 222")
}

// A second service on a profile whose first entry is pinned to another file is refused as a whole:
// no agent file is written and the state keeps its one entry.
func TestMcpAddRefusesPartialReaddWithOtherConfigFile(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: server.URL,
		// Unpinned: bound to the default file, which is not the one DATABRICKS_CONFIG_FILE names below.
	}}}))
	writeAddConfig(t, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	err := runAdd(t, "--profile", "logfood", "--names", "genie_one_mcp,github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory("PROFILE_HOST_CHANGED"), errorCategory(err))
	assert.Contains(t, err.Error(), "was set up with config file")
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, "github-logfood", state.Servers[0].Name)
}

// The profile's own credentials are used. The environment's token is not sent to a profile without one.
func TestMcpAddNeverUsesEnvTokenForProfileWithoutToken(t *testing.T) {
	home := withHome(t)
	server := noAPIServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\n")
	t.Setenv("DATABRICKS_TOKEN", "dapi-env-token")

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory("AUTH_FAILED"), errorCategory(err))
	assert.Contains(t, err.Error(), "databricks auth login --profile logfood")
}

// A configuration the SDK rejects is not an authentication failure, so a login would not fix it.
func TestMcpAddInvalidConfigIsNotAuthFailed(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	t.Setenv("DATABRICKS_RATE_LIMIT", "soon")

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, errorCategory(err))
}

// --agents is checked with the other flags, before the profile is read or the network is used.
func TestMcpAddRejectsUnknownAgentBeforeProfile(t *testing.T) {
	withHome(t)
	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "pi")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, errorCategory(err))
}

func TestMcpAddJSONWriteFailureKeepsRunError(t *testing.T) {
	withHome(t)
	withDatabricksBinary(t)
	parent := newAddTestRoot()
	parent.SetOut(failingWriter{})
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "add", "--names", "github", "--output", "json"})

	err := parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
	require.Error(t, err)
	assert.ErrorContains(t, err, "--output json requires --agents")
	assert.ErrorContains(t, err, "disk full")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

// An agent with a config directory is installed for MCP even when Detected would say otherwise.
func TestInstallServerCodexHomeCountsAsInstalled(t *testing.T) {
	home := withHome(t)
	codexHome := filepath.Join(t.TempDir(), "codex")
	require.NoError(t, os.MkdirAll(codexHome, 0o700))
	t.Setenv("CODEX_HOME", codexHome)

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("codex")}, binary: "databricks"}
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusInstalled, out.Agents[0].Status)
	assert.NoError(t, out.Agents[0].err)
	assert.FileExists(t, filepath.Join(codexHome, "config.toml"))
	assert.NoDirExists(t, filepath.Join(home, ".codex"))
}

// Gemini CLI is installed once its settings directory exists, whether or not projects.json does.
func TestInstallServerGeminiWithoutProjectsJSON(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini"), 0o700))

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("gemini")}, binary: "databricks"}
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusInstalled, out.Agents[0].Status)
	assert.FileExists(t, filepath.Join(home, ".gemini", "settings.json"))
}

// Antigravity's MCP file is in ~/.gemini/config, so that directory is what says it is installed.
func TestInstallServerAntigravityUsesItsMCPDirectory(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini", "config"), 0o700))

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("antigravity")}, binary: "databricks"}
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusInstalled, out.Agents[0].Status)
	assert.FileExists(t, filepath.Join(home, ".gemini", "config", "mcp_config.json"))
}

// A second install of the same entry reports "re-asserted" from the agent file, even when the state
// that recorded it is gone.
func TestInstallServerReassertsFromAgentFile(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o700))

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("claude-code")}, binary: "databricks"}
	plan := addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github-logfood"}

	first := installServer(t.Context(), req, plan)
	require.Len(t, first.Agents, 1)
	assert.Equal(t, statusInstalled, first.Agents[0].Status)

	second := installServer(t.Context(), req, plan)
	require.Len(t, second.Agents, 1)
	assert.Equal(t, statusReasserted, second.Agents[0].Status)
}

func TestAgentFailureErrorWhenNothingInstalled(t *testing.T) {
	ok := addServerOutput{Agents: []addAgentOutput{{Name: "claude-code", Status: statusInstalled}}}
	failed := addServerOutput{Agents: []addAgentOutput{{Name: "codex", Status: statusFailed, err: errors.New("bad config")}}}

	// Detected agents only warn, unless none of them was installed.
	assert.NoError(t, agentFailureError(addRequest{}, []addServerOutput{ok, failed}))
	err := agentFailureError(addRequest{}, []addServerOutput{failed})
	require.Error(t, err)
	assert.ErrorContains(t, err, "bad config")

	// Agents named with --agents fail the command on any failure.
	assert.Error(t, agentFailureError(addRequest{explicit: true}, []addServerOutput{ok, failed}))
	assert.NoError(t, agentFailureError(addRequest{explicit: true}, []addServerOutput{ok}))
}

func TestRequireProfileNamesProfileAndPath(t *testing.T) {
	ctx := t.Context()
	path := writeAddConfig(t, "[other]\nhost = https://other.test\n")

	err := requireProfile(ctx, "logfood")
	require.Error(t, err)
	assert.Equal(t, "profile logfood not found in "+filepath.ToSlash(path), err.Error())
	assert.Equal(t, protos.AitoolsErrorCategory("PROFILE_NOT_FOUND"), errorCategory(err))
}

func TestErrorCategoryMapsTypedFailures(t *testing.T) {
	assert.Equal(t, protos.AitoolsErrorCategory("MCP_NAME_CONFLICT"),
		errorCategory(fmt.Errorf("claude-code: server %q: %w", "github", agentconfig.ErrNotOwned)))
	assert.Equal(t, protos.AitoolsErrorCategoryTimeout, errorCategory(fmt.Errorf("list: %w", context.DeadlineExceeded)))
	assert.Equal(t, protos.AitoolsErrorCategoryPermissionDenied, errorCategory(&fs.PathError{Op: "open", Path: "x", Err: fs.ErrPermission}))
	assert.Equal(t, protos.AitoolsErrorCategory("AUTH_FAILED"), errorCategory(mcp.AuthError("logfood", errors.New("no token"))))
}
