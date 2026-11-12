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
	"runtime"
	"strconv"
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
			name: "name must be a valid server name", flags: mcpAddFlags{name: "a.b"}, names: []string{"github"},
			want: `--name "a.b" is not a valid server name: use lowercase letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{
			name: "name must not contain a double underscore", flags: mcpAddFlags{name: "a__b"}, names: []string{"github"},
			want: `--name "a__b" is not a valid server name: use lowercase letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{
			name: "name must not start with an underscore", flags: mcpAddFlags{name: "_a"}, names: []string{"github"},
			want: `--name "_a" is not a valid server name: use lowercase letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{
			name: "name must not end with a dash", flags: mcpAddFlags{name: "a-"}, names: []string{"github"},
			want: `--name "a-" is not a valid server name: use lowercase letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
		{name: "valid name", flags: mcpAddFlags{name: "my_github-1"}, names: []string{"github"}},
		{
			name: "name must be lowercase", flags: mcpAddFlags{name: "GitHub"}, names: []string{"github"},
			want: `--name "GitHub" is not a valid server name: use lowercase letters, digits, '-' and '_', with no "__" and no leading or trailing '-' or '_'`,
		},
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
	assert.Equal(t, []string{"system.ai.github", "system.ai.genie_one_mcp"}, got, "a leaf and its full name are one service")

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

func TestResolveServices(t *testing.T) {
	cases := []struct {
		name     string
		location string
		names    []string
		wantLoc  string
		wantFull []string
		wantErr  string
	}{
		{name: "leaves resolve in the default schema", names: []string{"github"}, wantLoc: "system.ai", wantFull: []string{"system.ai.github"}},
		{
			name: "full names in one schema give that schema", names: []string{"main.tools.crm", "main.tools.github"},
			wantLoc: "main.tools", wantFull: []string{"main.tools.crm", "main.tools.github"},
		},
		{name: "full names in system.ai give system.ai", names: []string{"system.ai.github", "github"}, wantLoc: "system.ai", wantFull: []string{"system.ai.github"}},
		{
			name: "a leaf with full names outside system.ai is refused, since it would resolve in system.ai", names: []string{"main.tools.crm", "github"},
			wantErr: "leaf names resolve in system.ai, but the full names are in main.tools; pass --location main.tools to resolve the leaves there, or use full names only",
		},
		{
			name: "full names in different schemas need --location", names: []string{"main.tools.crm", "system.ai.github"},
			wantErr: "--names are in different schemas (main.tools, system.ai); run add once for each schema",
		},
		{
			name: "an explicit --location decides, and full names must agree with it", location: "main.tools", names: []string{"system.ai.github"},
			wantErr: "system.ai.github is not in main.tools; pass --location to match its schema",
		},
		{
			name: "an explicit --location is lowercased", location: "Main.Tools", names: []string{"crm"},
			wantLoc: "main.tools", wantFull: []string{"main.tools.crm"},
		},
		{
			name: "full names are case-insensitive and distinct from their leaves", names: []string{"github", "System.AI.GitHub"},
			wantLoc: "system.ai", wantFull: []string{"system.ai.github"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loc, full, err := resolveServices(tc.location, tc.names)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.wantErr, err.Error())
				assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantLoc, loc)
			assert.Equal(t, tc.wantFull, full)
		})
	}
}

func TestPickServices(t *testing.T) {
	services := testServices()

	// The result is one service per name, in the order of the names, so it lines up with the planned servers.
	picked, err := pickServices("system.ai", []string{"system.ai.genie_one_mcp", "system.ai.github"}, services)
	require.NoError(t, err)
	require.Len(t, picked, 2)
	assert.Equal(t, "system.ai.genie_one_mcp", picked[0].FullName)
	assert.Equal(t, "system.ai.github", picked[1].FullName)
	assert.True(t, picked[1].SignInRequired)

	picked, err = pickServices("System.AI", []string{"System.AI.GitHub"}, services)
	require.NoError(t, err)
	assert.Equal(t, "system.ai.github", picked[0].FullName)

	_, err = pickServices("system.ai", []string{"system.ai.foo"}, services)
	require.Error(t, err)
	assert.Equal(t, "system.ai.foo is not an MCP service in system.ai. Available: system.ai.genie_one_mcp, system.ai.github", err.Error())
	assert.Equal(t, categoryServiceNotFound, errorCategory(err))
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
	assert.Equal(t, categoryPermissionDenied, errorCategory(err))

	got, ok := errors.AsType[*mcp.ListError](err)
	require.True(t, ok)
	assert.Equal(t, mcp.ListErrPermissionDenied, got.Kind)
	assert.ErrorIs(t, err, cause)

	rateLimited := describeListError("logfood", &mcp.ListError{Kind: mcp.ListErrRateLimited, Location: "system.ai", Err: cause})
	assert.Equal(t, protos.AitoolsErrorCategoryRateLimited, errorCategory(rateLimited))

	notFound := describeListError("logfood", &mcp.ListError{Kind: mcp.ListErrLocationNotFound, Location: "main.tools", Err: cause})
	assert.Equal(t, categoryLocationNotFound, errorCategory(notFound))

	plain := errors.New("network down")
	assert.Equal(t, plain, describeListError("logfood", plain))
}

// withHome points the home directory at a temp dir, so agent config paths are inside it. The
// agent home overrides are cleared, and PATH is empty, so a developer's own agents do not reach the test.
// Claude Code is installed in the home, so a run that names it with --agents is not refused.
func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	t.Setenv("PATH", t.TempDir())
	for _, name := range agentHomeEnv {
		t.Setenv(name, "")
	}
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o700))
	return home
}

// The variables that move an agent's config directory. Codex and Copilot CLI keep their MCP config there.
const (
	codexHomeEnv   = "CODEX_HOME"
	copilotHomeEnv = "COPILOT_HOME"
)

// agentHomeEnv are the variables that move an agent's config directory.
var agentHomeEnv = []string{"CLAUDE_CONFIG_DIR", codexHomeEnv, copilotHomeEnv, "GEMINI_CLI_HOME", "XDG_CONFIG_HOME", "APPDATA", "DATABRICKS_CONFIG_FILE"}

// withDatabricksBinary makes the command name the databricks command, as the installed CLI has it.
func withDatabricksBinary(t *testing.T) {
	t.Helper()
	saved := binaryArg0
	binaryArg0 = "databricks"
	t.Cleanup(func() { binaryArg0 = saved })
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

// rejectingServer returns a server that rejects every API call with 401, as a workspace does for an expired token.
// A command that reaches its first API call fails with AUTH_FAILED.
func rejectingServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error_code":"UNAUTHENTICATED","message":"Invalid access token."}`))
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

// TestInstallServerRefusesUndetectedExplicitAgent checks that an agent named with --agents that is not
// installed is refused: its config is not written and its directory is not created. The agent that is
// installed still gets the server.
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
	req.refused = refusedAgents(ctx, req.agents)
	plan := addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"}

	out := installServer(ctx, req, plan)
	require.Len(t, out.Agents, 2)

	codex := out.Agents[0]
	assert.Equal(t, statusFailed, codex.Status)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, codex.ErrorCategory)
	assert.Equal(t, "Codex CLI is not installed; install it first or leave it out of --agents", codex.Message)
	assert.NoDirExists(t, filepath.Join(home, ".codex"))

	assert.Equal(t, statusInstalled, out.Agents[1].Status)
	assert.NoError(t, out.Agents[1].err)
}

// defaultBinding returns the binding to profile in the default config file, resolved as the command resolves it.
func defaultBinding(t *testing.T, profile string) addBinding {
	t.Helper()
	path, err := mcp.ResolveConfigFile(t.Context(), "")
	require.NoError(t, err)
	return addBinding{profile: profile, configFile: path}
}

// planNames runs planAdds for the services with the agent entries read as the command reads them.
func planNames(t *testing.T, nameFlag string, b addBinding, state *mcp.State, fullNames []string, targets []*agents.Agent) ([]addPlan, error) {
	t.Helper()
	return planAdds(t.Context(), nameFlag, b, state, fullNames, targets, readAgentEntries(t.Context(), targets))
}

// claudeEntry is a Databricks entry for claude-code that runs the serve command for service and profile, with the
// config file pinned in env when pin is not "".
func claudeEntry(service, profile, pin string) string {
	env := ""
	if pin != "" {
		env = `,"env":{"DATABRICKS_CONFIG_FILE":` + strconv.Quote(pin) + `}`
	}
	return `{"type":"stdio","command":"databricks","args":["aitools","mcp","serve","--profile","` + profile + `","--service","` + service + `","--agent","claude-code"]` + env + `}`
}

func TestPlanAddsNaming(t *testing.T) {
	claude := agents.ByName("claude-code")
	const host = "https://adb-123.test"

	t.Run("new service gets the generated name", func(t *testing.T) {
		withHome(t)
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), &mcp.State{}, []string{"system.ai.genie_one_mcp"}, []*agents.Agent{claude})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Equal(t, "genie_one_mcp-logfood", plans[0].name)
		assert.Nil(t, plans[0].entry)
		assert.Equal(t, "system.ai.genie_one_mcp", plans[0].service.FullName)
	})

	t.Run("existing install keeps its name", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "custom-name", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), state, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "custom-name", plans[0].name)
		assert.NotNil(t, plans[0].entry)
	})

	t.Run("stored entry for the same binding is found in its own config file only", func(t *testing.T) {
		withHome(t)
		other := writeAddConfig(t, "[logfood]\nhost = "+host+"\n")
		// The first entry is for the same service and profile, in another config file. The second is in the default file.
		state := &mcp.State{Servers: []mcp.Entry{
			{Name: "github-other-file", Service: "system.ai.github", Profile: "logfood", Host: host, ConfigFile: other},
			{Name: "github-default-file", Service: "system.ai.github", Profile: "logfood", Host: host},
		}}
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), state, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "github-default-file", plans[0].name)
	})

	t.Run("name used by another state entry gets a suffix", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "genie_one_mcp-logfood", Service: "main.tools.other", Profile: "logfood", Host: host}}}
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), state, []string{"system.ai.genie_one_mcp"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "genie_one_mcp-logfood-2", plans[0].name)
	})

	t.Run("unmanaged entry in an agent file gets a suffix", func(t *testing.T) {
		home := withHome(t)
		unmanaged := `{"mcpServers":{"genie_one_mcp-logfood":{"type":"stdio","command":"ug","args":["mcp-proxy"]}}}`
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(unmanaged), 0o600))
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), &mcp.State{}, []string{"system.ai.genie_one_mcp"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "genie_one_mcp-logfood-2", plans[0].name)
	})

	t.Run("managed entry of another profile keeps its name when state is missing", func(t *testing.T) {
		home := withHome(t)
		// dev.foo and dev-foo sanitize to the same server name, so the entry for dev.foo is not dev-foo's.
		other := `{"mcpServers":{"github-dev-foo":{"type":"stdio","command":"databricks","args":["aitools","mcp","serve","--profile","dev.foo","--service","system.ai.github","--agent","claude-code"]}}}`
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(other), 0o600))
		plans, err := planNames(t, "", defaultBinding(t, "dev-foo"), &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "github-dev-foo-2", plans[0].name)
	})

	t.Run("managed entry of the same binding is reused when state is missing", func(t *testing.T) {
		home := withHome(t)
		same := `{"mcpServers":{"github-dev-foo":{"type":"stdio","command":"databricks","args":["aitools","mcp","serve","--profile","dev.foo","--service","system.ai.github","--agent","claude-code"]}}}`
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(same), 0o600))
		plans, err := planNames(t, "", defaultBinding(t, "dev.foo"), &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "github-dev-foo", plans[0].name)
	})

	t.Run("entry this binding wrote under a --name is reused when state is missing", func(t *testing.T) {
		home := withHome(t)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"mine":`+claudeEntry("system.ai.github", "logfood", "")+`}}`), 0o600))
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "mine", plans[0].name)
	})

	t.Run("entry of this binding pinned to the default file by its absolute path is reused", func(t *testing.T) {
		home := withHome(t)
		b := defaultBinding(t, "logfood")
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"mine":`+claudeEntry("system.ai.github", "logfood", b.configFile)+`}}`), 0o600))
		plans, err := planNames(t, "", b, &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "mine", plans[0].name)
	})

	t.Run("entry pinned to the default file by a relative or ~ path is the default file", func(t *testing.T) {
		home := withHome(t)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"mine":`+claudeEntry("system.ai.github", "logfood", "~/.databrickscfg")+`}}`), 0o600))
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "mine", plans[0].name)
	})

	t.Run("entry for this binding in another config file is not reused", func(t *testing.T) {
		home := withHome(t)
		other := writeAddConfig(t, "[logfood]\nhost = "+host+"\n")
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"mine":`+claudeEntry("system.ai.github", "logfood", other)+`}}`), 0o600))
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "github-logfood", plans[0].name)
	})

	t.Run("entries of this binding named differently in two agents are refused", func(t *testing.T) {
		home := withHome(t)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"one":`+claudeEntry("system.ai.github", "logfood", "")+`}}`), 0o600))
		openCode := filepath.Join(home, ".config", "opencode")
		require.NoError(t, os.MkdirAll(openCode, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(openCode, "opencode.json"), []byte(`{"mcp":{"servers":{"two":{"type":"local","command":["databricks","aitools","mcp","serve","--profile","logfood","--service","system.ai.github","--agent","opencode"]}}}}`), 0o600))
		_, err := planNames(t, "", defaultBinding(t, "logfood"), &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{claude, agents.ByName("opencode")})
		require.Error(t, err)
		assert.Equal(t, categoryNameConflict, errorCategory(err))
	})

	t.Run("OpenCode entry with its variables in environment is reused", func(t *testing.T) {
		home := withHome(t)
		pinned := writeAddConfig(t, "[logfood]\nhost = https://adb-123.test\n")
		openCode := filepath.Join(home, ".config", "opencode")
		require.NoError(t, os.MkdirAll(openCode, 0o700))
		// OpenCode keeps the variables under environment and the command as one array.
		entry := `{"type":"local","command":["databricks","aitools","mcp","serve","--profile","logfood","--service","system.ai.github","--agent","opencode"],"environment":{"DATABRICKS_CONFIG_FILE":` + strconv.Quote(pinned) + `}}`
		require.NoError(t, os.WriteFile(filepath.Join(openCode, "opencode.json"), []byte(`{"mcp":{"servers":{"mine":`+entry+`}}}`), 0o600))
		b := addBinding{profile: "logfood", configFile: pinned}
		plans, err := planNames(t, "", b, &mcp.State{}, []string{"system.ai.github"}, []*agents.Agent{agents.ByName("opencode")})
		require.NoError(t, err)
		assert.Equal(t, "mine", plans[0].name)
	})

	t.Run("name flag for an installed service must match", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		_, err := planNames(t, "other", defaultBinding(t, "logfood"), state, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, "system.ai.github is already installed on profile logfood as github-logfood; remove it before using --name other", err.Error())
		assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
	})

	t.Run("name flag already used by another service is refused", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "taken", Service: "main.tools.other", Profile: "logfood", Host: host}}}
		_, err := planNames(t, "taken", defaultBinding(t, "logfood"), state, []string{"system.ai.genie_one_mcp"}, []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, "--name taken is used by another installed server; pick a different name", err.Error())
	})

	t.Run("stored name held by another binding in an agent is refused", func(t *testing.T) {
		home := withHome(t)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"github-logfood":`+claudeEntry("system.ai.github", "dev-foo", "")+`}}`), 0o600))
		state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		_, err := planNames(t, "", defaultBinding(t, "logfood"), state, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, categoryNameConflict, errorCategory(err))
	})

	t.Run("planning reads the agent entries it is given, not the files", func(t *testing.T) {
		withHome(t)
		// Nothing is on disk. The entries that the command read once decide the names.
		held := map[*agents.Agent]map[string]map[string]any{
			claude: {"genie_one_mcp-logfood": map[string]any{"type": "stdio", "command": "ug", "args": []any{"mcp-proxy"}}},
		}
		plans, err := planAdds(t.Context(), "", defaultBinding(t, "logfood"), &mcp.State{}, []string{"system.ai.genie_one_mcp"}, []*agents.Agent{claude}, held)
		require.NoError(t, err)
		assert.Equal(t, "genie_one_mcp-logfood-2", plans[0].name)
	})

	t.Run("a service whose generated name is empty is refused", func(t *testing.T) {
		withHome(t)
		_, err := planNames(t, "", defaultBinding(t, "数据"), &mcp.State{}, []string{"system.ai.数据"}, []*agents.Agent{claude})
		require.Error(t, err)
		assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
		assert.Equal(t, "cannot make a server name from system.ai.数据 for profile 数据; pass --name", err.Error())
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

// TestMcpAddRefusesRepointedHostBeforeAnyRequest checks that a profile re-pointed at another host is refused before any request reaches that host.
func TestMcpAddRefusesRepointedHostBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test",
	}}}))

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileHostChanged), errorCategory(err))
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
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileHostChanged), errorCategory(err))
	assert.Contains(t, err.Error(), "was set up for workspace 111, but profile \"logfood\" now points at workspace 222")
}

// TestMcpAddRefusesPartialReaddWithOtherConfigFile checks that a second service on a profile whose first entry is pinned to another file is refused as a whole:
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
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileHostChanged), errorCategory(err))
	assert.Contains(t, err.Error(), "was set up with config file")
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	require.Len(t, state.Servers, 1)
	assert.Equal(t, "github-logfood", state.Servers[0].Name)
}

// TestMcpAddNeverUsesEnvTokenForProfileWithoutToken checks that the profile's own credentials are used. The environment's token is not sent to a profile without one.
func TestMcpAddNeverUsesEnvTokenForProfileWithoutToken(t *testing.T) {
	home := withHome(t)
	server := noAPIServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\n")
	t.Setenv("DATABRICKS_TOKEN", "dapi-env-token")

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindAuthFailed), errorCategory(err))
	assert.Contains(t, err.Error(), "databricks auth login --profile logfood")
}

// TestMcpAddInvalidConfigIsNotAuthFailed checks that a configuration the SDK rejects is not an authentication failure, so a login would not fix it.
func TestMcpAddInvalidConfigIsNotAuthFailed(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	t.Setenv("DATABRICKS_RATE_LIMIT", "soon")

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, errorCategory(err))
}

// TestMcpAddRejectsUnknownAgentBeforeProfile checks that --agents is checked with the other flags, before the profile is read or the network is used.
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

// TestInstallServerCodexHomeCountsAsInstalled checks that an agent with a config directory is installed for MCP even when Detected would say otherwise.
func TestInstallServerCodexHomeCountsAsInstalled(t *testing.T) {
	home := withHome(t)
	codexHome := filepath.Join(t.TempDir(), "codex")
	require.NoError(t, os.MkdirAll(codexHome, 0o700))
	t.Setenv(codexHomeEnv, codexHome)

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("codex")}, binary: "databricks"}
	req.refused = refusedAgents(t.Context(), req.agents)
	require.Empty(t, req.refused, "the agent must count as installed, so the refusal path must not refuse it")
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusInstalled, out.Agents[0].Status)
	assert.NoError(t, out.Agents[0].err)
	assert.FileExists(t, filepath.Join(codexHome, "config.toml"))
	assert.NoDirExists(t, filepath.Join(home, ".codex"))
}

// TestInstallServerGeminiDirAloneIsRefused checks that gemini CLI's directory is shared with Antigravity, so ~/.gemini alone does not install Gemini CLI.
func TestInstallServerGeminiDirAloneIsRefused(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini", "antigravity"), 0o700))

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("gemini")}, binary: "databricks"}
	req.refused = refusedAgents(t.Context(), req.agents)
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusFailed, out.Agents[0].Status)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, out.Agents[0].ErrorCategory)
	assert.NoFileExists(t, filepath.Join(home, ".gemini", "settings.json"))
}

// TestInstallServerGeminiWithProjectsJSON checks that gemini CLI is installed once it has written projects.json, as the registry detects it.
func TestInstallServerGeminiWithProjectsJSON(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".gemini", "projects.json"), []byte("{}"), 0o600))

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("gemini")}, binary: "databricks"}
	req.refused = refusedAgents(t.Context(), req.agents)
	require.Empty(t, req.refused, "projects.json must count as installed, so the refusal path must not refuse it")
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusInstalled, out.Agents[0].Status)
	assert.FileExists(t, filepath.Join(home, ".gemini", "settings.json"))
}

// TestInstallServerAntigravityFromItsOwnDir checks that antigravity is installed from its own directory, and its MCP file is written under ~/.gemini/config.
func TestInstallServerAntigravityFromItsOwnDir(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini", "antigravity"), 0o700))

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("antigravity")}, binary: "databricks"}
	req.refused = refusedAgents(t.Context(), req.agents)
	require.Empty(t, req.refused, "its own directory must count as installed, so the refusal path must not refuse it")
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusInstalled, out.Agents[0].Status)
	assert.FileExists(t, filepath.Join(home, ".gemini", "config", "mcp_config.json"))
}

// TestInstallServerCLIOnPathCountsAsInstalled checks that a CLI found only on PATH counts as installed, so its MCP config is written.
func TestInstallServerCLIOnPathCountsAsInstalled(t *testing.T) {
	home := withHome(t)
	dir := t.TempDir()
	name := "codex"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)

	req := addRequest{profile: "logfood", explicit: true, agents: []*agents.Agent{agents.ByName("codex")}, binary: "databricks"}
	req.refused = refusedAgents(t.Context(), req.agents)
	require.Empty(t, req.refused, "a CLI on PATH must count as installed, so the refusal path must not refuse it")
	out := installServer(t.Context(), req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
	require.Len(t, out.Agents, 1)
	assert.Equal(t, statusInstalled, out.Agents[0].Status)
	assert.FileExists(t, filepath.Join(home, ".codex", "config.toml"))
}

// TestInstallServerReassertsFromAgentFile checks that a second install of the same entry reports "re-asserted" from the agent file, even when the state
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

// TestAgentFailureErrorWhenAServiceReachedNoAgent checks that a service that reached no agent fails the command, even when another service was installed.
func TestAgentFailureErrorWhenAServiceReachedNoAgent(t *testing.T) {
	installed := addServerOutput{Agents: []addAgentOutput{{Name: "claude-code", Status: statusInstalled}}}
	mixed := addServerOutput{Agents: []addAgentOutput{
		{Name: "claude-code", Status: statusInstalled},
		{Name: "codex", Status: statusFailed, err: errors.New("bad config")},
	}}
	failedAll := addServerOutput{Agents: []addAgentOutput{{Name: "codex", Status: statusFailed, err: errors.New("bad config")}}}

	// Detected agents only warn while every service reached one of them.
	assert.NoError(t, agentFailureError(addRequest{}, []addServerOutput{installed, mixed}))

	err := agentFailureError(addRequest{}, []addServerOutput{installed, failedAll})
	require.Error(t, err)
	assert.ErrorContains(t, err, "bad config")

	// Agents named with --agents fail the command on any failure.
	assert.Error(t, agentFailureError(addRequest{explicit: true}, []addServerOutput{installed, mixed}))
	assert.NoError(t, agentFailureError(addRequest{explicit: true}, []addServerOutput{installed}))
}

func TestRequireProfileNamesProfileAndPath(t *testing.T) {
	ctx := t.Context()
	path := writeAddConfig(t, "[other]\nhost = https://other.test\n")

	err := requireProfile(ctx, "logfood")
	require.Error(t, err)
	assert.Equal(t, "profile logfood not found in "+filepath.ToSlash(path), err.Error())
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
}

func TestErrorCategoryMapsTypedFailures(t *testing.T) {
	assert.Equal(t, categoryNameConflict,
		errorCategory(fmt.Errorf("claude-code: server %q: %w", "github", agentconfig.ErrNotOwned)))
	assert.Equal(t, protos.AitoolsErrorCategoryTimeout, errorCategory(fmt.Errorf("list: %w", context.DeadlineExceeded)))
	assert.Equal(t, protos.AitoolsErrorCategoryPermissionDenied, errorCategory(&fs.PathError{Op: "open", Path: "x", Err: fs.ErrPermission}))
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindAuthFailed), errorCategory(mcp.AuthError("logfood", errors.New("no token"))))
}

// TestMcpAddRefusesMixedCaseNamesBeforeAnyRequest checks that mixed-case service and location names bind to the same entry as their lowercase forms, so a profile
// re-pointed at another host is refused before any request, whatever the case typed.
func TestMcpAddRefusesMixedCaseNamesBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test",
	}}}))

	err := runAdd(t, "--profile", "logfood", "--location", "System.AI", "--names", "GitHub", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileHostChanged), errorCategory(err))
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// TestMcpAddRefusesNameFlagConflictBeforeAnyRequest checks that a --name that the state gives to another server is refused before any request.
func TestMcpAddRefusesNameFlagConflictBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "taken", Service: "main.tools.other", Profile: "logfood", Host: server.URL,
	}}}))

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--name", "taken", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, "--name taken is used by another installed server; pick a different name", err.Error())
}

// TestMcpAddRefusesNameFlagHeldByOtherBindingBeforeAnyRequest checks that a --name that an agent file gives to another profile's managed server is refused before any request.
func TestMcpAddRefusesNameFlagHeldByOtherBindingBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[dev-foo]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-dev-foo\n")
	other := `{"mcpServers":{"mine":{"type":"stdio","command":"databricks","args":["aitools","mcp","serve","--profile","dev.foo","--service","system.ai.github","--agent","claude-code"]}}}`
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(other), 0o600))

	err := runAdd(t, "--profile", "dev-foo", "--names", "github", "--name", "mine", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
	assert.Contains(t, err.Error(), "--name mine is used by another server")
}

// TestMcpAddRefusesUninstalledAgentBeforeAnyRequest checks that an agent named with --agents that is not installed is refused before any request, once, not per service.
func TestMcpAddRefusesUninstalledAgentBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	err := runAdd(t, "--profile", "logfood", "--names", "github,genie_one_mcp", "--agents", "codex")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, errorCategory(err))
	assert.Equal(t, "Codex CLI is not installed; install it first or leave it out of --agents", err.Error())
	assert.NoDirExists(t, filepath.Join(home, ".codex"))
}

// TestMcpAddMissingProfileWithEntryGetsRecreateHint checks that a profile that is gone while a server is still bound to it gets the recreate hint, not the generic message.
func TestMcpAddMissingProfileWithEntryGetsRecreateHint(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[other]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-other\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test",
	}}}))

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
	assert.Contains(t, err.Error(), "MCP server github-logfood is bound to Databricks profile \"logfood\", which no longer exists in ")
	assert.Contains(t, err.Error(), "databricks auth login --host https://adb-123.test --profile logfood")
}

// TestMcpAddEmptyHostIsProfileNotFound checks that a profile whose host is empty has no workspace, so it is reported as missing before any request.
func TestMcpAddEmptyHostIsProfileNotFound(t *testing.T) {
	home := withHome(t)
	writeHomeConfig(t, home, "[logfood]\nhost =\nauth_type = pat\ntoken = dapi-logfood\n")

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
	assert.Contains(t, err.Error(), "profile logfood not found in ")
}

// TestMcpAddRejectedTokenIsAuthFailed checks that a credential the workspace rejects on the first real request is AUTH_FAILED, with the login command.
func TestMcpAddRejectedTokenIsAuthFailed(t *testing.T) {
	home := withHome(t)
	server := rejectingServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-expired\n")

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindAuthFailed), errorCategory(err))
	assert.Contains(t, err.Error(), "Run: databricks auth login --profile logfood")
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// TestMcpAddRefusesBinaryNotNamedDatabricks checks that the binary must be databricks: an argv[0] with another name, not on PATH, is refused, not replaced.
func TestMcpAddRefusesBinaryNotNamedDatabricks(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	saved := binaryArg0
	binaryArg0 = "dbx-not-on-path"
	t.Cleanup(func() { binaryArg0 = saved })

	_, err := newAddTestCmd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, errorCategory(err))
	assert.Contains(t, err.Error(), "dbx-not-on-path is not the databricks binary")
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// TestMcpAddRefusesStoredNameHeldByOtherBinding checks that a server name stored for this profile is refused, before any request, when an agent
// holds that name for another profile's managed server.
func TestMcpAddRefusesStoredNameHeldByOtherBinding(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n[dev-foo]\nhost = https://adb-456.test\nauth_type = pat\ntoken = dapi-dev\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: server.URL,
	}}}))
	other := `{"mcpServers":{"github-logfood":{"type":"stdio","command":"databricks","args":["aitools","mcp","serve","--profile","dev-foo","--service","system.ai.github","--agent","claude-code"]}}}`
	claudeFile := filepath.Join(home, ".claude.json")
	require.NoError(t, os.WriteFile(claudeFile, []byte(other), 0o600))

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, categoryNameConflict, errorCategory(err))
	assert.ErrorContains(t, err, "github-logfood")
	got, err := os.ReadFile(claudeFile)
	require.NoError(t, err)
	assert.Equal(t, other, string(got))
}

// TestErrorCategoryKeepsCauseOfUncategorizedBindingError checks that a binding failure with no category of its own does not hide the
// cause it wraps, so a filesystem or permission error is still classified. The cause is built here: no binding path
// wraps an OS error with %w that the SDK keeps.
func TestErrorCategoryKeepsCauseOfUncategorizedBindingError(t *testing.T) {
	cause := &fs.PathError{Op: "open", Path: "cfg", Err: fs.ErrPermission}
	err := fmt.Errorf("MCP server github-logfood: %w", errors.Join(&mcp.BindError{Category: mcp.BindUncategorized}, cause))
	assert.Equal(t, protos.AitoolsErrorCategoryPermissionDenied, errorCategory(err))

	// Without a cause the failure stays uncategorized.
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, errorCategory(&mcp.BindError{Category: mcp.BindUncategorized}))
}

// TestMcpAddNamesResolvedConfigFileWhenDotDotFollowsSymlink checks that the config file is named as it resolves, so a path that goes through a symlink
// and then ".." is read from the file that the OS reads.
func TestMcpAddNamesResolvedConfigFileWhenDotDotFollowsSymlink(t *testing.T) {
	base := t.TempDir()
	withHome(t)
	real := filepath.Join(base, "real")
	require.NoError(t, os.MkdirAll(filepath.Join(real, "sub"), 0o700))
	cfg := filepath.Join(real, "cfg")
	require.NoError(t, os.WriteFile(cfg, []byte("[other]\nhost = https://other.test\n"), 0o600))
	if err := os.Symlink(filepath.Join(real, "sub"), filepath.Join(base, "link")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	// The variable is written as typed: filepath.Join would clean the ".." away.
	t.Setenv("DATABRICKS_CONFIG_FILE", base+string(filepath.Separator)+"link"+string(filepath.Separator)+".."+string(filepath.Separator)+"cfg")

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
	resolved, evalErr := filepath.EvalSymlinks(cfg)
	require.NoError(t, evalErr)
	assert.Equal(t, "profile logfood not found in "+filepath.ToSlash(resolved), err.Error())
}

// TestMcpAddRefusesNameThatSanitizesToNothing checks that a service whose generated name is empty is refused before any request, and the message
// says to pass --name.
func TestMcpAddRefusesNameThatSanitizesToNothing(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[数据]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	err := runAdd(t, "--profile", "数据", "--names", "数据", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
	assert.ErrorContains(t, err, "pass --name")
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// TestMcpAddNameNeedsOneService checks that --name is refused with two services, and is checked before any request.
func TestMcpAddNameNeedsOneService(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	err := runAdd(t, "--profile", "logfood", "--names", "github,genie_one_mcp", "--name", "custom", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, "--name needs exactly one service in --names", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
}

// TestMcpAddCountsDistinctServicesForName checks that a leaf and the full name of the same service are one service, so --name is accepted with them.
func TestMcpAddCountsDistinctServicesForName(t *testing.T) {
	home := withHome(t)
	server := rejectingServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	err := runAdd(t, "--profile", "logfood", "--names", "github,system.ai.github", "--name", "custom", "--agents", "claude-code")
	require.Error(t, err)
	// The request is the first thing that can fail; the flags were accepted.
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindAuthFailed), errorCategory(err))
}

// TestMcpAddTakesLocationFromFullNames checks that full names that all name one schema are resolved there when --location is not set.
func TestMcpAddTakesLocationFromFullNames(t *testing.T) {
	home := withHome(t)
	server := rejectingServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	err := runAdd(t, "--profile", "logfood", "--names", "main.tools.github", "--agents", "claude-code")
	require.Error(t, err)
	// The schema was accepted, so the first request was made.
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindAuthFailed), errorCategory(err))
}

// TestMcpAddAgreesWithBind checks that add refuses the same bindings that Bind refuses, with the same message where the failure is about the server.
func TestMcpAddAgreesWithBind(t *testing.T) {
	const service = "system.ai.github"
	cases := []struct {
		name  string
		setup func(t *testing.T, home, host string)
		// sameMessage is false when add reports the profile as missing with its own wording.
		sameMessage bool
	}{
		{
			name: "host changed",
			setup: func(t *testing.T, home, host string) {
				writeHomeConfig(t, home, "[logfood]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-logfood\n")
				saveEntry(t, mcp.Entry{Name: "github-logfood", Service: service, Profile: "logfood", Host: "https://adb-123.test"})
			},
			sameMessage: true,
		},
		{
			name: "workspace changed on the same host",
			setup: func(t *testing.T, home, host string) {
				writeHomeConfig(t, home, "[logfood]\nhost = "+host+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
				saveEntry(t, mcp.Entry{Name: "github-logfood", Service: service, Profile: "logfood", Host: host, WorkspaceID: "111"})
			},
			sameMessage: true,
		},
		{
			name: "entry pinned to another config file",
			setup: func(t *testing.T, home, host string) {
				writeHomeConfig(t, home, "[logfood]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-logfood\n")
				saveEntry(t, mcp.Entry{Name: "github-logfood", Service: service, Profile: "logfood", Host: host, ConfigFile: filepath.Join(t.TempDir(), "other")})
			},
			sameMessage: true,
		},
		{
			name: "profile gone while an entry is bound to it",
			setup: func(t *testing.T, home, host string) {
				writeHomeConfig(t, home, "[other]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-other\n")
				saveEntry(t, mcp.Entry{Name: "github-logfood", Service: service, Profile: "logfood", Host: host})
			},
			sameMessage: true,
		},
		{
			name: "profile gone with no entry",
			setup: func(t *testing.T, home, host string) {
				writeHomeConfig(t, home, "[other]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-other\n")
			},
			sameMessage: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := withHome(t)
			server := noRequestServer(t)
			tc.setup(t, home, server.URL)

			state, err := mcp.LoadState(t.Context())
			require.NoError(t, err)
			var stderr bytes.Buffer
			_, _, bindErr := mcp.Bind(t.Context(), mcp.BindOptions{Profile: "logfood", Service: service, State: state, Stderr: &stderr})
			require.Error(t, bindErr)

			addErr := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
			require.Error(t, addErr)
			assert.Equal(t, errorCategory(bindErr), errorCategory(addErr))
			if tc.sameMessage {
				assert.Equal(t, bindErr.Error(), addErr.Error())
			}
			assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
		})
	}
}

// saveEntry saves the state with one entry.
func saveEntry(t *testing.T, entry mcp.Entry) {
	t.Helper()
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{entry}}))
}
