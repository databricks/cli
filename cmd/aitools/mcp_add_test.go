package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
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
			name: "login and no-login conflict", flags: mcpAddFlags{login: true, noLogin: true}, names: []string{"github"},
			want: "--login and --no-login cannot be used together",
		},
		{name: "login alone", flags: mcpAddFlags{login: true}, names: []string{"github"}},
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
	assert.Equal(t, protos.AitoolsErrorCategoryMCPServiceNotFound, errorCategory(err))
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
	assert.Equal(t, protos.AitoolsErrorCategoryMCPPermissionDenied, errorCategory(err))

	got, ok := errors.AsType[*mcp.ListError](err)
	require.True(t, ok)
	assert.Equal(t, mcp.ListErrPermissionDenied, got.Kind)
	assert.ErrorIs(t, err, cause)

	rateLimited := describeListError("logfood", &mcp.ListError{Kind: mcp.ListErrRateLimited, Location: "system.ai", Err: cause})
	assert.Equal(t, protos.AitoolsErrorCategoryRateLimited, errorCategory(rateLimited))

	notFound := describeListError("logfood", &mcp.ListError{Kind: mcp.ListErrLocationNotFound, Location: "main.tools", Err: cause})
	assert.Equal(t, protos.AitoolsErrorCategoryMCPLocationNotFound, errorCategory(notFound))

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
// it contacts the profile's host never reaches it. Telemetry is off: its auth resolution, which starts with the command,
// reads the host's metadata, a request the refusal does not make.
func noRequestServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("DATABRICKS_CLI_DISABLE_TELEMETRY", "1")
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
		profile:     "logfood",
		agentSource: agentsNamed,
		agents:      []*agents.Agent{agents.ByName("codex"), agents.ByName("claude-code")},
		binary:      "databricks",
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
// readHeld returns the server entries of the targets that readAgentEntries reads, for tests that do not check unreadable configs.
func readHeld(ctx context.Context, targets []*agents.Agent) agentEntries {
	held, _ := readAgentEntries(ctx, targets)
	return held
}

func planNames(t *testing.T, nameFlag string, b addBinding, state *mcp.State, fullNames []string, targets []*agents.Agent) ([]addPlan, error) {
	t.Helper()
	in := planInput{binding: b, nameFlag: nameFlag, targets: targets, held: readHeld(t.Context(), targets)}
	return planAdds(t.Context(), in, state, fullNames)
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
		assert.Equal(t, "system.ai.genie_one_mcp", plans[0].service.FullName)
	})

	t.Run("existing install keeps its name", func(t *testing.T) {
		withHome(t)
		state := &mcp.State{Servers: []mcp.Entry{{Name: "custom-name", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), state, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		assert.Equal(t, "custom-name", plans[0].name)
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
		assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(err))
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

	t.Run("stored name held by another binding in a detected agent fails only that agent", func(t *testing.T) {
		home := withHome(t)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"github-logfood":`+claudeEntry("system.ai.github", "dev-foo", "")+`}}`), 0o600))
		state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host}}}
		plans, err := planNames(t, "", defaultBinding(t, "logfood"), state, []string{"system.ai.github"}, []*agents.Agent{claude})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Equal(t, "github-logfood", plans[0].name)
		require.Contains(t, plans[0].failed, claude)
		assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(plans[0].failed[claude]))
	})

	t.Run("planning reads the agent entries it is given, not the files", func(t *testing.T) {
		withHome(t)
		// Nothing is on disk. The entries that the command read once decide the names.
		held := map[*agents.Agent]map[string]map[string]any{
			claude: {"genie_one_mcp-logfood": map[string]any{"type": "stdio", "command": "ug", "args": []any{"mcp-proxy"}}},
		}
		in := planInput{binding: defaultBinding(t, "logfood"), targets: []*agents.Agent{claude}, held: held}
		plans, err := planAdds(t.Context(), in, &mcp.State{}, []string{"system.ai.genie_one_mcp"})
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

// newMCPTestRoot returns a root command with the persistent --profile and --output flags that root
// defines, and the mcp group.
func newMCPTestRoot() *cobra.Command {
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
	parent := newMCPTestRoot()
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
	parent := newMCPTestRoot()
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

// TestInstallServerReassertsFromAgentFile checks that a second install of the same entry reports "re-asserted" from the agent file, even when the state
// that recorded it is gone.
func TestInstallServerReassertsFromAgentFile(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude"), 0o700))

	req := addRequest{profile: "logfood", agentSource: agentsNamed, agents: []*agents.Agent{agents.ByName("claude-code")}, binary: "databricks"}
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
	assert.Error(t, agentFailureError(addRequest{agentSource: agentsNamed}, []addServerOutput{installed, mixed}))
	assert.NoError(t, agentFailureError(addRequest{agentSource: agentsNamed}, []addServerOutput{installed}))
}

func TestErrorCategoryMapsTypedFailures(t *testing.T) {
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict,
		errorCategory(fmt.Errorf("claude-code: server %q: %w", "github", agentconfig.ErrNotOwned)))
	assert.Equal(t, protos.AitoolsErrorCategoryTimeout, errorCategory(fmt.Errorf("list: %w", context.DeadlineExceeded)))
	assert.Equal(t, protos.AitoolsErrorCategoryPermissionDenied, errorCategory(&fs.PathError{Op: "open", Path: "x", Err: fs.ErrPermission}))
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindAuthFailed), errorCategory(mcp.AuthError("logfood", "", errors.New("no token"))))
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
	assert.Contains(t, err.Error(), "profile \"logfood\", which has no host in ")
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

// TestMcpAddBindingFailureExitsOne checks that a binding failure is an ordinary failure of add: it exits 1 in text and
// in JSON mode. Exit 2 is serve's status for binding failures, which an agent restarts on.
func TestMcpAddBindingFailureExitsOne(t *testing.T) {
	home := withHome(t)
	withDatabricksBinary(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test",
	}}}))
	args := []string{"--profile", "logfood", "--names", "github", "--agents", "claude-code"}

	_, err := newAddTestCmd(t, args...)
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileHostChanged), errorCategory(err))
	assert.Equal(t, 1, root.ExitStatus(err))

	_, err = newAddTestCmd(t, append(args, "--output", "json")...)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 1, root.ExitStatus(err))
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
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(err))
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
	assert.Contains(t, err.Error(), "which does not exist in "+filepath.ToSlash(resolved)+". Set it up with: ")
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
// setupHostChanged prepares the case "host changed" for a profile in the default config file.
func setupHostChanged(t *testing.T, home, host string) {
	writeHomeConfig(t, home, "[logfood]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-logfood\n")
	saveEntry(t, mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test"})
}

// setupWorkspaceChanged prepares the case "workspace changed on the same host" for a profile in the default config file.
func setupWorkspaceChanged(t *testing.T, home, host string) {
	writeHomeConfig(t, home, "[logfood]\nhost = "+host+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	saveEntry(t, mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host, WorkspaceID: "111"})
}

// setupOtherConfigFile prepares the case "entry pinned to another config file" for a profile in the default config file.
func setupOtherConfigFile(t *testing.T, home, host string) {
	writeHomeConfig(t, home, "[logfood]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-logfood\n")
	saveEntry(t, mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host, ConfigFile: filepath.Join(t.TempDir(), "other")})
}

// setupProfileGoneWithEntry prepares the case "profile gone while an entry is bound to it" for a profile in the default config file.
func setupProfileGoneWithEntry(t *testing.T, home, host string) {
	writeHomeConfig(t, home, "[other]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-other\n")
	saveEntry(t, mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host})
}

// setupProfileGoneWithoutEntry prepares the case "profile gone with no entry" for a profile in the default config file.
func setupProfileGoneWithoutEntry(t *testing.T, home, host string) {
	writeHomeConfig(t, home, "[other]\nhost = "+host+"\nauth_type = pat\ntoken = dapi-other\n")
}

func TestMcpAddAgreesWithBind(t *testing.T) {
	const service = "system.ai.github"
	cases := []struct {
		name  string
		setup func(t *testing.T, home, host string)
	}{
		{
			name:  "host changed",
			setup: setupHostChanged,
		},
		{
			name:  "workspace changed on the same host",
			setup: setupWorkspaceChanged,
		},
		{
			name:  "entry pinned to another config file",
			setup: setupOtherConfigFile,
		},
		{
			name:  "profile gone while an entry is bound to it",
			setup: setupProfileGoneWithEntry,
		},
		{
			name:  "profile gone with no entry",
			setup: setupProfileGoneWithoutEntry,
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
			assert.Equal(t, bindErr.Error(), addErr.Error())
			assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
		})
	}
}

// saveEntry saves the state with one entry.
func saveEntry(t *testing.T, entry mcp.Entry) {
	t.Helper()
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{entry}}))
}

// TestPlanAddsRefusesRecoveredNameHeldByAnotherServer checks that a name recovered from an agent entry, which has no state record, is refused
// when another state entry holds it. Two servers with one name would overwrite each other's state.
func TestPlanAddsRefusesRecoveredNameHeldByAnotherServer(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"github-logfood":`+claudeEntry("system.ai.github", "logfood", "")+`}}`), 0o600))
	state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.genie_one_mcp", Profile: "logfood", Host: "https://adb-123.test"}}}

	_, err := planNames(t, "", defaultBinding(t, "logfood"), state, []string{"system.ai.github"}, []*agents.Agent{agents.ByName("claude-code")})
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(err))
	assert.Equal(t, "system.ai.github is installed as github-logfood, but another server in the state holds github-logfood; remove that entry first", err.Error())
}

// TestMcpAddRefusesUnmanagedEntryBeforeAnyRequest checks that the stored name of a server is refused before any request when an agent holds that
// name as an entry this CLI did not write. The agent's write would refuse it only after the workspace was contacted.
func TestMcpAddRefusesUnmanagedEntryBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: server.URL,
	}}}))
	claudeFile := filepath.Join(home, ".claude.json")
	unmanaged := `{"mcpServers":{"github-logfood":{"type":"stdio","command":"npx","args":["-y","other-server"]}}}`
	require.NoError(t, os.WriteFile(claudeFile, []byte(unmanaged), 0o600))

	err := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(err))
	assert.ErrorContains(t, err, "Claude Code already has a server named github-logfood that this CLI did not write; pass --force to replace it")
	got, err := os.ReadFile(claudeFile)
	require.NoError(t, err)
	assert.Equal(t, unmanaged, string(got))
}

// TestMcpAddAccountProfileRefusedBeforeAnyRequest checks that a profile with no host, such as an account-level profile, is refused before any
// request, with the same error as a missing profile in bind.
func TestMcpAddAccountProfileRefusedBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	noRequestServer(t)
	writeHomeConfig(t, home, "[account]\naccount_id = 123\n")

	err := runAdd(t, "--profile", "account", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
	assert.Contains(t, err.Error(), "profile \"account\", which has no host in ")
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// TestMcpAddMissingProfileWithoutEntryMatchesBind checks that a profile that is missing when no server is installed is reported by the shared
// check, so add and bind word the failure the same way.
func TestMcpAddMissingProfileWithoutEntryMatchesBind(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[other]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-other\n")

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	_, _, bindErr := mcp.Bind(t.Context(), mcp.BindOptions{Profile: "logfood", Service: "system.ai.github", State: state, Stderr: io.Discard})
	require.Error(t, bindErr)

	err = runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
	assert.Equal(t, bindErr.Error(), err.Error())
	// Nothing is installed for the profile, so the message does not say the server is bound to it.
	assert.NotContains(t, err.Error(), "remove the server")
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// TestRecordInstallsSavesServersThatCanBeSaved checks that a server whose state record fails does not stop the other servers from being
// recorded. The agent files of every server were written, so their state must be saved, and the failure is still returned.
func TestRecordInstallsSavesServersThatCanBeSaved(t *testing.T) {
	ctx := t.Context()
	withHome(t)
	// A config file name with a NUL byte cannot be resolved, so the record of the first service fails.
	require.NoError(t, mcp.SaveState(ctx, &mcp.State{Servers: []mcp.Entry{{
		Name: "bad-logfood", Service: "system.ai.bad", Profile: "logfood", Host: "https://adb-123.test", ConfigFile: "bad\x00name",
	}}}))
	configFile, err := mcp.ResolveConfigFile(ctx, "")
	require.NoError(t, err)
	req := addRequest{profile: "logfood", host: "https://adb-456.test", configFile: configFile}
	plans := []addPlan{
		{service: mcp.Service{FullName: "system.ai.bad"}, name: "bad-logfood"},
		{service: mcp.Service{FullName: "system.ai.github"}, name: "github-logfood"},
	}
	servers := []addServerOutput{
		{Name: "bad-logfood", Service: "system.ai.bad", Agents: []addAgentOutput{{Name: "claude-code", Status: statusInstalled, Path: "/x/.claude.json"}}},
		{Name: "github-logfood", Service: "system.ai.github", Agents: []addAgentOutput{{Name: "claude-code", Status: statusInstalled, Path: "/x/.claude.json"}}},
	}

	err = recordInstalls(ctx, req, plans, servers)
	require.Error(t, err)
	assert.ErrorContains(t, err, "MCP server bad-logfood")

	state, loadErr := mcp.LoadState(ctx)
	require.NoError(t, loadErr)
	var names []string
	for _, entry := range state.Servers {
		names = append(names, entry.Name)
	}
	assert.Equal(t, []string{"bad-logfood", "github-logfood"}, names)
}

// TestAgentFailureErrorKeepsEveryDistinctFailure checks that an agent's failures are all reported when they differ, and that one failure that repeats for
// each service, such as a refusal, is reported once.
func TestAgentFailureErrorKeepsEveryDistinctFailure(t *testing.T) {
	refusal := errors.New("Claude Code is not installed")
	denied := fmt.Errorf("write ~/.claude.json: %w", fs.ErrPermission)
	servers := []addServerOutput{
		{Agents: []addAgentOutput{{Name: "claude-code", Status: statusFailed, err: refusal}}},
		{Agents: []addAgentOutput{{Name: "claude-code", Status: statusFailed, err: denied}}},
		{Agents: []addAgentOutput{{Name: "claude-code", Status: statusFailed, err: refusal}}},
	}

	err := agentFailureError(addRequest{agentSource: agentsNamed}, servers)
	require.Error(t, err)
	assert.ErrorContains(t, err, "write ~/.claude.json")
	assert.Equal(t, 1, strings.Count(err.Error(), "Claude Code is not installed"), err.Error())
}

// TestResolveServicesMalformedFullNameIsNotALocationError checks that a name with an empty part is reported as a malformed service name, not as a
// --location error, since no --location was passed.
func TestResolveServicesMalformedFullNameIsNotALocationError(t *testing.T) {
	for _, name := range []string{"system..github", "system.ai."} {
		t.Run(name, func(t *testing.T) {
			_, _, err := resolveServices("", []string{name})
			require.Error(t, err)
			assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
			assert.Equal(t, name+" is not a service name: use a leaf such as github or a full name such as system.ai.github", err.Error())
		})
	}
}

// TestPlanAddsDetectedAgentConflictFailsOnlyThatAgent checks that a name conflict in an agent that was only detected fails that agent for the server, and
// the other agents still take it. The same conflict in an agent named with --agents refuses the run. When every agent fails, the run is refused.
func TestPlanAddsDetectedAgentConflictFailsOnlyThatAgent(t *testing.T) {
	ctx := t.Context()
	home := withHome(t)
	claude := agents.ByName("claude-code")
	codex := agents.ByName("codex")
	// claude-code holds the stored name of this server for another server, so the name conflicts in claude-code only.
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"github-logfood":`+claudeEntry("system.ai.other", "logfood", "")+`}}`), 0o600))
	state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test"}}}
	targets := []*agents.Agent{claude, codex}
	in := planInput{binding: defaultBinding(t, "logfood"), targets: targets, held: readHeld(ctx, targets)}

	plans, err := planAdds(ctx, in, state, []string{"system.ai.github"})
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, "github-logfood", plans[0].name)
	require.Contains(t, plans[0].failed, claude)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(plans[0].failed[claude]))
	assert.NotContains(t, plans[0].failed, codex)
	assert.NoError(t, planningRefusals(plans, targets))

	in.explicit = true
	_, err = planAdds(ctx, in, state, []string{"system.ai.github"})
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(err))

	onlyClaude := []*agents.Agent{claude}
	in = planInput{binding: defaultBinding(t, "logfood"), targets: onlyClaude, held: readHeld(ctx, onlyClaude)}
	plans, err = planAdds(ctx, in, state, []string{"system.ai.github"})
	require.NoError(t, err)
	refusal := planningRefusals(plans, onlyClaude)
	require.Error(t, refusal)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(refusal))
}

// TestPlanAddsUnmanagedEntryIsReplacedOnlyWithForce checks that a stored name held by an entry this CLI did not write is refused at planning,
// and that --force lets the write replace it.
func TestPlanAddsUnmanagedEntryIsReplacedOnlyWithForce(t *testing.T) {
	ctx := t.Context()
	claude := agents.ByName("claude-code")
	state := &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test"}}}
	held := agentEntries{claude: {"github-logfood": map[string]any{"type": "stdio", "command": "npx", "args": []any{"-y", "other"}}}}
	targets := []*agents.Agent{claude}

	in := planInput{binding: defaultBinding(t, "logfood"), targets: targets, held: held, explicit: true}
	_, err := planAdds(ctx, in, state, []string{"system.ai.github"})
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryMCPNameConflict, errorCategory(err))

	in.force = true
	plans, err := planAdds(ctx, in, state, []string{"system.ai.github"})
	require.NoError(t, err)
	assert.Empty(t, plans[0].failed)
}

// TestPickServicesNamesSchemaWithNoServices checks the message for a schema that has no MCP services, which has no list of available services.
func TestPickServicesNamesSchemaWithNoServices(t *testing.T) {
	_, err := pickServices("system.ai", []string{"system.ai.github"}, nil)
	require.Error(t, err)
	assert.Equal(t, "system.ai.github is not an MCP service in system.ai, which has none", err.Error())
	assert.Equal(t, protos.AitoolsErrorCategoryMCPServiceNotFound, errorCategory(err))
}

// TestInstallServerWritesOnlyAgentsInstalledForMCP checks that an agent named with --agents is written when it counts as installed for MCP, and that
// the write goes to the file its config lives in. An agent that does not count is refused, and nothing is created for it.
func TestInstallServerWritesOnlyAgentsInstalledForMCP(t *testing.T) {
	tests := []struct {
		name    string
		agent   string
		homeEnv string // moves the MCP config out of the agent's directory
		homeDir string // the directory under home that homeEnv names
		dirs    []string
		files   map[string]string
		onPath  bool
		// written is the config file, under home, that the write creates; empty when the agent is refused.
		written string
		// absent is the path, under home, that must not exist after the run.
		absent string
	}{
		{name: "codex home directory exists", agent: "codex", homeEnv: codexHomeEnv, homeDir: "codex-home", dirs: []string{"codex-home"}, written: "codex-home/config.toml"},
		{name: "codex home directory missing although ~/.codex exists", agent: "codex", homeEnv: codexHomeEnv, homeDir: "codex-home", dirs: []string{".codex"}, absent: "codex-home"},
		{name: "gemini with projects.json", agent: "gemini", dirs: []string{".gemini"}, files: map[string]string{".gemini/projects.json": "{}"}, written: ".gemini/settings.json"},
		{name: "gemini directory alone is refused", agent: "gemini", dirs: []string{".gemini/antigravity"}, absent: ".gemini/settings.json"},
		{name: "antigravity from its own directory", agent: "antigravity", dirs: []string{".gemini/antigravity"}, written: ".gemini/config/mcp_config.json"},
		// The CLI on PATH is a stand-in that exits 0 and writes nothing, so the direct writer writes the config.
		{name: "cli on PATH", agent: "codex", onPath: true, written: ".codex/config.toml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			home := withHome(t)
			if tt.homeEnv != "" {
				t.Setenv(tt.homeEnv, filepath.Join(home, tt.homeDir))
			}
			for _, dir := range tt.dirs {
				require.NoError(t, os.MkdirAll(filepath.Join(home, dir), 0o700))
			}
			for name, contents := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(home, name), []byte(contents), 0o600))
			}
			if tt.onPath {
				writeFakeCLI(t, agents.ByName(tt.agent).Binary)
			}

			req := addRequest{profile: "logfood", agentSource: agentsNamed, agents: []*agents.Agent{agents.ByName(tt.agent)}, binary: "databricks"}
			req.refused = refusedAgents(ctx, req.agents)
			out := installServer(ctx, req, addPlan{service: mcp.Service{FullName: "system.ai.github"}, name: "github"})
			require.Len(t, out.Agents, 1)

			if tt.written == "" {
				assert.Equal(t, statusFailed, out.Agents[0].Status)
				assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, out.Agents[0].ErrorCategory)
			} else {
				assert.Equal(t, statusInstalled, out.Agents[0].Status)
				assert.NoError(t, out.Agents[0].err)
				assert.FileExists(t, filepath.Join(home, filepath.FromSlash(tt.written)))
			}
			if tt.absent != "" {
				_, err := os.Stat(filepath.Join(home, filepath.FromSlash(tt.absent)))
				assert.ErrorIs(t, err, fs.ErrNotExist)
			}
		})
	}
}

// writeFakeCLI puts an executable named binary in a directory that is the only one on PATH, so the agent counts as installed.
func writeFakeCLI(t *testing.T, binary string) {
	t.Helper()
	dir := t.TempDir()
	name := binary
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)
}

// runInteractiveAdd runs `aitools mcp add` as a terminal does. The prompts are the seams, which the test overrides.
func runInteractiveAdd(t *testing.T, args ...string) error {
	t.Helper()
	return runInteractive(t, append([]string{"mcp", "add"}, args...)...)
}

// runInteractive runs the aitools command with args as a terminal does, under the mcp group.
func runInteractive(t *testing.T, args ...string) error {
	t.Helper()
	withDatabricksBinary(t)
	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(test.Done)
	go drainReader(test.Stdout)
	go drainReader(test.Stderr)
	parent := newMCPTestRoot()
	parent.SetOut(&bytes.Buffer{})
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs(args)
	return parent.ExecuteContext(ctx)
}

// TestMcpAddNameOnThePickerPathIsRefusedBeforeConnecting checks that --name without --names is refused before the
// profile is read, so no request reaches the workspace.
func TestMcpAddNameOnThePickerPathIsRefusedBeforeConnecting(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	failOnPrompt(t)

	err := runInteractiveAdd(t, "--profile", "logfood", "--name", "github")
	require.Error(t, err)
	assert.Equal(t, "--name needs exactly one service; leave out --name to pick services in the picker", err.Error())
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// TestMcpAddPickerRefusesRepointedHostOfPickedServiceBeforeWriting checks that a run that picks its services lists the
// workspace, then refuses a picked service whose recorded entry names another host, before anything is written.
func TestMcpAddPickerRefusesRepointedHostOfPickedServiceBeforeWriting(t *testing.T) {
	home := withHome(t)
	server := listingServer(t, "system.ai.github")
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	require.NoError(t, mcp.SaveState(t.Context(), &mcp.State{Servers: []mcp.Entry{{
		Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test",
	}}}))
	failOnPrompt(t)
	promptServiceSelection = func(string, []serviceRow) (servicePick, error) {
		return servicePick{services: []mcp.Service{{FullName: "system.ai.github", Leaf: "github"}}}, nil
	}

	err := runInteractiveAdd(t, "--profile", "logfood", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileHostChanged), errorCategory(err))
	assert.Contains(t, err.Error(), "was set up for https://adb-123.test, but profile \"logfood\" now points at "+server.URL)
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
}

// schemaListingServer returns a workspace whose schemas list the services named for them, by the parent of the listing.
// Each service is a full name, none sign-in-required.
func schemaListingServer(t *testing.T, schemas map[string][]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/api/2.1/unity-catalog/mcp-services" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var services []string
		for _, name := range schemas[strings.TrimPrefix(r.URL.Query().Get("parent"), "schemas/")] {
			services = append(services, `{"name":"mcp-services/`+name+`"}`)
		}
		_, _ = w.Write([]byte(`{"mcp_services":[` + strings.Join(services, ",") + `]}`))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestMcpAddPickerInstallsAcrossSchemas checks that a terminal run with a profile and no --names installs the services it
// picks in two schemas, reached through "Another schema...": both entries reach the agent file and the state.
func TestMcpAddPickerInstallsAcrossSchemas(t *testing.T) {
	home := withHome(t)
	server := schemaListingServer(t, map[string][]string{
		"system.ai":  {"system.ai.github"},
		"main.tools": {"main.tools.slack"},
	})
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	failOnPrompt(t)
	promptServiceSelection = func(location string, _ []serviceRow) (servicePick, error) {
		if location == "system.ai" {
			return servicePick{services: []mcp.Service{{FullName: "system.ai.github", Leaf: "github"}}, another: true}, nil
		}
		return servicePick{services: []mcp.Service{{FullName: "main.tools.slack", Leaf: "slack"}}}, nil
	}
	promptLocation = func() (string, error) { return "main.tools", nil }
	promptAgentSelection = func(_ context.Context, _ string, choices []agentChoice) ([]*agents.Agent, error) {
		return []*agents.Agent{choices[0].agent}, nil
	}
	proceeds := 0
	promptProceed = func() (bool, error) {
		proceeds++
		return true, nil
	}

	require.NoError(t, runInteractiveAdd(t, "--profile", "logfood"))
	assert.Equal(t, 1, proceeds, "the picked plan is confirmed once")

	agentFile, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	require.NoError(t, err)
	assert.Contains(t, string(agentFile), `"github-logfood"`)
	assert.Contains(t, string(agentFile), `"main-tools-slack-logfood"`)

	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	var recorded []string
	for _, e := range state.Servers {
		recorded = append(recorded, e.Service)
	}
	assert.ElementsMatch(t, []string{"system.ai.github", "main.tools.slack"}, recorded)
}

// TestMcpAddPickerRefusesMissingProfileBeforeAnyRequest checks that a run that picks its services reports a profile the
// config file does not have, or has with no host, as PROFILE_NOT_FOUND, before any request.
func TestMcpAddPickerRefusesMissingProfileBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	failOnPrompt(t)

	err := runInteractiveAdd(t, "--profile", "logfdood", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
	assert.Contains(t, err.Error(), "Databricks profile \"logfdood\", which does not exist in ")
	assert.Contains(t, err.Error(), "Set it up with: databricks auth login --profile logfdood")

	writeHomeConfig(t, home, "[logfood]\nhost =\nauth_type = pat\ntoken = dapi-logfood\n")
	err = runInteractiveAdd(t, "--profile", "logfood", "--agents", "claude-code")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(err))
	assert.Contains(t, err.Error(), "Databricks profile \"logfood\", which has no host in ")
}

// TestMcpAddPickerWithNoAgentMakesNoRequest checks that a run that picks its services ends when no agent is detected,
// before the workspace is contacted.
func TestMcpAddPickerWithNoAgentMakesNoRequest(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.RemoveAll(filepath.Join(home, ".claude")))
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	failOnPrompt(t)

	err := runInteractiveAdd(t, "--profile", "logfood")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, errorCategory(err))
	assert.Contains(t, err.Error(), "no MCP-capable coding agents detected")
}

// TestMcpAddWithoutTerminalReportsNoAgentBeforeProfile checks that a run without a terminal and with no agent reports
// that before the profile, which here would be ambiguous with two profiles.
func TestMcpAddWithoutTerminalReportsNoAgentBeforeProfile(t *testing.T) {
	home := withHome(t)
	require.NoError(t, os.RemoveAll(filepath.Join(home, ".claude")))
	writeHomeConfig(t, home, "[logfood]\nhost = https://logfood.test\nauth_type = pat\ntoken = dapi-logfood\n"+
		"[dogfood]\nhost = https://dogfood.test\nauth_type = pat\ntoken = dapi-dogfood\n")

	err := runAdd(t, "--names", "github")
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, errorCategory(err))
	assert.Contains(t, err.Error(), "no MCP-capable coding agents detected")
}

// TestMcpAddInTerminalRefusesWhenAgentCheckFails checks that a terminal with no agent installed for MCP, and an agent
// whose check fails, is refused in chooseAgents: the failure is named, nothing is picked and the check is not repeated.
func TestMcpAddInTerminalRefusesWhenAgentCheckFails(t *testing.T) {
	withHome(t)
	t.Setenv(env.HomeEnvVar(), "")
	for _, name := range agentHomeEnv {
		t.Setenv(name, "")
	}
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		t.Error("the run asked for agents with none installed")
		return nil, nil
	}

	claude := agents.ByName(agents.NameClaudeCode)
	_, checkErr := claude.InstalledForMCP(ctx)
	require.Error(t, checkErr, "the check needs a home directory")

	_, _, err := chooseAgents(ctx, nil, true)
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, errorCategory(err))
	assert.Contains(t, err.Error(), "no MCP-capable coding agents detected")
	assert.Contains(t, stderr.String(), fmt.Sprintf("Not found: %s (%s)", claude.DisplayName, checkErr))
}

// failOnPrompt makes every picker and question of an interactive run fail the test, so a test can show what a
// run asks. It restores the seams when the test ends.
func failOnPrompt(t *testing.T) {
	t.Helper()
	origProceed, origAgents, origServices, origProfile := promptProceed, promptAgentSelection, promptServiceSelection, selectProfile
	origSignIn := promptSignInNow
	t.Cleanup(func() {
		promptProceed, promptAgentSelection, promptServiceSelection, selectProfile = origProceed, origAgents, origServices, origProfile
		promptSignInNow = origSignIn
	})
	promptProceed = func() (bool, error) { t.Error("the run asked to confirm the plan"); return false, nil }
	promptAgentSelection = func(context.Context, string, []agentChoice) ([]*agents.Agent, error) {
		t.Error("the run asked for agents")
		return nil, nil
	}
	promptServiceSelection = func(string, []serviceRow) (servicePick, error) {
		t.Error("the run asked for services")
		return servicePick{}, nil
	}
	selectProfile = func(context.Context, profile.SelectConfig) (string, error) {
		t.Error("the run asked for a profile")
		return "", nil
	}
	promptSignInNow = func(string) (bool, error) {
		t.Error("the run asked to sign in")
		return false, nil
	}
}

// listingServer returns a workspace whose schema lists the MCP services in names, none of them sign-in-required.
// A run that binds a profile with a workspace_id lists the services and makes no other call.
func listingServer(t *testing.T, names ...string) *httptest.Server {
	t.Helper()
	services := make([]string, 0, len(names))
	for _, name := range names {
		services = append(services, `{"name":"mcp-services/`+name+`"}`)
	}
	return listingServerJSON(t, services...)
}

// signInListingServer returns a workspace whose schema lists the sign-in-required service name, whose source connection
// has an OAuth provider. The service is not signed in, so its credential read answers NOT_FOUND.
func signInListingServer(t *testing.T, name string) *httptest.Server {
	t.Helper()
	return listingServerJSON(t, `{"name":"mcp-services/`+name+`","config":{"source_connection":{"name":"github","options":{"oauth_provider":"github"}}}}`)
}

// listingServerJSON returns a workspace whose schema lists services, each a JSON MCP service. A credential read answers
// NOT_FOUND, as for a caller not signed in; any other request is a failure.
func listingServerJSON(t *testing.T, services ...string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			// The SDK reads host metadata before it calls the API; there is none here.
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/user-credentials") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not found"}`))
			return
		}
		if r.URL.Path != "/api/2.1/unity-catalog/mcp-services" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"mcp_services":[` + strings.Join(services, ",") + `]}`))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestMcpAddFullySpecifiedRunAsksNothing checks that a terminal run with the profile, the services and the agents
// given asks for nothing and installs.
func TestMcpAddFullySpecifiedRunAsksNothing(t *testing.T) {
	home := withHome(t)
	server := listingServer(t, "system.ai.dbsql")
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	failOnPrompt(t)

	require.NoError(t, runInteractiveAdd(t, "--profile", "logfood", "--names", "dbsql", "--agents", "claude-code"))
	claude, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	require.NoError(t, err)
	assert.Contains(t, string(claude), "dbsql-logfood")
}

// TestMcpAddFullySpecifiedRunAsksOnlyTheSignInQuestion checks that a terminal run with the profile, the services and the
// agents given asks the sign-in question of a sign-in-required server, and nothing else.
func TestMcpAddFullySpecifiedRunAsksOnlyTheSignInQuestion(t *testing.T) {
	home := withHome(t)
	server := signInListingServer(t, "system.ai.github")
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	failOnPrompt(t)
	var asked []string
	promptSignInNow = func(service string) (bool, error) {
		asked = append(asked, service)
		return false, nil
	}

	require.NoError(t, runInteractiveAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code"))
	assert.Equal(t, []string{"system.ai.github"}, asked)
	claude, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	require.NoError(t, err)
	assert.Contains(t, string(claude), "github-logfood")
}

// TestMcpAddDeclinedPlanWritesNothing checks that declining the confirm of a picked agent leaves the agent's
// config unwritten.
func TestMcpAddDeclinedPlanWritesNothing(t *testing.T) {
	home := withHome(t)
	server := listingServer(t, "system.ai.dbsql")
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-logfood\n")
	failOnPrompt(t)
	origAgents, origProceed := promptAgentSelection, promptProceed
	t.Cleanup(func() { promptAgentSelection, promptProceed = origAgents, origProceed })
	promptAgentSelection = func(_ context.Context, _ string, choices []agentChoice) ([]*agents.Agent, error) {
		return []*agents.Agent{agents.ByName(agents.NameClaudeCode)}, nil
	}
	var confirmed bool
	promptProceed = func() (bool, error) {
		confirmed = true
		return false, nil
	}

	require.NoError(t, runInteractiveAdd(t, "--profile", "logfood", "--names", "dbsql"))
	assert.True(t, confirmed)
	assert.NoFileExists(t, filepath.Join(home, ".claude.json"))
	state, err := mcp.LoadState(t.Context())
	require.NoError(t, err)
	assert.Empty(t, state.Servers)
}

// credentialServer returns a workspace whose credential reads of the services in active answer ACTIVE, and that counts
// its API reads in reads. The others answer NOT_FOUND, as the API does until the caller signs in.
func credentialServer(t *testing.T, reads *atomic.Int32, active ...string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		reads.Add(1)
		for _, service := range active {
			if r.URL.Path == "/api/2.1/unity-catalog/mcp-services/"+service+"/user-credentials" {
				_, _ = w.Write([]byte(`{"provisioning_info":{"state":"ACTIVE"}}`))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not found"}`))
	}))
	t.Cleanup(server.Close)
	return server
}

// signInRun returns the requests and plans of two sign-in-required servers that reached claude-code, and the workspace
// they sign in to. The servers are github and genie_one_mcp.
func signInRun(t *testing.T, reads *atomic.Int32, active ...string) (*databricks.WorkspaceClient, addRequest, []addPlan, []addServerOutput) {
	t.Helper()
	server := credentialServer(t, reads, active...)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	req := addRequest{profile: "logfood", workspaceID: "123", agents: []*agents.Agent{agents.ByName(agents.NameClaudeCode)}}
	plans := []addPlan{
		{service: mcp.Service{FullName: "system.ai.github", SignInRequired: true}},
		{service: mcp.Service{FullName: "system.ai.genie_one_mcp", SignInRequired: true}},
	}
	installed := []addAgentOutput{{Name: agents.NameClaudeCode, Status: statusInstalled}}
	servers := []addServerOutput{
		{Name: "github-logfood", Service: "system.ai.github", SignInRequired: true, Agents: installed},
		{Name: "genie-logfood", Service: "system.ai.genie_one_mcp", SignInRequired: true, Agents: installed},
	}
	return w, req, plans, servers
}

// askSignIn overrides the sign-in question for a test, and returns the services it was asked about.
func askSignIn(t *testing.T, answer func(service string) (bool, error)) *[]string {
	t.Helper()
	orig := promptSignInNow
	t.Cleanup(func() { promptSignInNow = orig })
	var asked []string
	promptSignInNow = func(service string) (bool, error) {
		asked = append(asked, service)
		return answer(service)
	}
	return &asked
}

// TestSignInInstalledAsksEachServer checks that each server not signed in is asked about in order, and that a declined
// one keeps its server installed and prints the login command.
func TestSignInInstalledAsksEachServer(t *testing.T) {
	w, req, plans, servers := signInRun(t, new(atomic.Int32))
	req.askLogin = true
	asked := askSignIn(t, func(string) (bool, error) { return false, nil })

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	require.NoError(t, signInInstalled(ctx, io.Discard, w, req, plans, servers))
	assert.Equal(t, []string{"system.ai.github", "system.ai.genie_one_mcp"}, *asked)
	assert.Contains(t, stderr.String(), "system.ai.github needs sign-in. Run: ")
	assert.Contains(t, stderr.String(), "system.ai.genie_one_mcp needs sign-in. Run: ")
}

// TestSignInInstalledSkipsTheQuestionWhenSignedIn checks that a server already signed in is not asked about, and
// that its sign-in is recorded as ACTIVE.
func TestSignInInstalledSkipsTheQuestionWhenSignedIn(t *testing.T) {
	var reads atomic.Int32
	w, req, plans, servers := signInRun(t, &reads, "system.ai.github")
	req.askLogin = true
	asked := askSignIn(t, func(string) (bool, error) { return false, nil })

	ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
	var errOut bytes.Buffer
	require.NoError(t, signInInstalled(ctx, &errOut, w, req, plans, servers))
	assert.Equal(t, []string{"system.ai.genie_one_mcp"}, *asked)
	require.NotNil(t, servers[0].SignIn)
	assert.Equal(t, signInStateActive, servers[0].SignIn.State)
	assert.Equal(t, "Already signed in to system.ai.github on profile logfood.\n", errOut.String())
	// Each server is read once, for its question. The sign-in of the signed-in server is not started, so it is not read again.
	assert.Equal(t, int32(2), reads.Load())
}

// TestSignInInstalledAbortPrintsTheRestOfTheHints checks that an abort at the question ends the sign-ins, prints the
// login hint of each server not signed in, and returns the abort.
// TestSignedInGivesUpOnAHungCredentialRead checks that a credential read that never answers ends at profileCheckTimeout,
// so the sign-in question is asked, instead of holding it for the SDK's whole retry budget.
func TestSignedInGivesUpOnAHungCredentialRead(t *testing.T) {
	origTimeout := profileCheckTimeout
	profileCheckTimeout = 100 * time.Millisecond
	t.Cleanup(func() { profileCheckTimeout = origTimeout })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only the credential read hangs; the SDK's host metadata read at setup answers at once.
		if !strings.HasSuffix(r.URL.Path, "/user-credentials") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "dapi-test"})
	require.NoError(t, err)

	start := time.Now()
	assert.False(t, signedIn(t.Context(), w, "system.ai.github"))
	assert.Less(t, time.Since(start), 10*time.Second)
}

func TestSignInInstalledAbortPrintsTheRestOfTheHints(t *testing.T) {
	var reads atomic.Int32
	w, req, plans, servers := signInRun(t, &reads)
	req.askLogin = true
	asked := askSignIn(t, func(string) (bool, error) { return false, huh.ErrUserAborted })

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	err := signInInstalled(ctx, io.Discard, w, req, plans, servers)
	require.ErrorIs(t, err, huh.ErrUserAborted)
	assert.Equal(t, protos.AitoolsErrorCategoryUserAborted, errorCategory(err))
	assert.Equal(t, []string{"system.ai.github"}, *asked, "the sign-ins stop at the abort")
	// The only read is the one that asked about github: the hints of the rest are printed without a request.
	assert.Equal(t, int32(1), reads.Load())
	assert.Contains(t, stderr.String(), "system.ai.github needs sign-in. Run: ")
	assert.Contains(t, stderr.String(), "system.ai.genie_one_mcp needs sign-in. Run: ")
}

// TestPlanAddsReplacesAForeignEntryAndReassertsOwnEntry checks that a plan names what the write does to an entry the
// agent already holds: a replace for an entry this CLI did not write, a re-assert for one of this binding.
func TestPlanAddsReplacesAForeignEntryAndReassertsOwnEntry(t *testing.T) {
	home := withHome(t)
	claude := agents.ByName(agents.NameClaudeCode)
	targets := []*agents.Agent{claude}
	config := filepath.Join(home, ".claude.json")
	b := defaultBinding(t, "logfood")

	// A foreign entry under the name, replaced under --force.
	require.NoError(t, os.WriteFile(config, []byte(`{"mcpServers":{"github-logfood":{"type":"stdio","command":"other"}}}`), 0o600))
	in := planInput{binding: b, nameFlag: "github-logfood", force: true, targets: targets, held: readHeld(t.Context(), targets)}
	plans, err := planAdds(t.Context(), in, &mcp.State{}, []string{"system.ai.github"})
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.True(t, plans[0].replaced[claude])
	assert.False(t, plans[0].reasserted[claude])
	req := addRequest{profile: "logfood", agents: targets}
	assert.Equal(t, []string{"Plan (profile logfood):", fmt.Sprintf("  %-14s %s", claude.DisplayName, "replace github-logfood")}, addPlanLines(req, plans))

	// An entry of this binding under the name is re-asserted.
	require.NoError(t, os.WriteFile(config, []byte(`{"mcpServers":{"github-logfood":`+claudeEntry("system.ai.github", "logfood", "")+`}}`), 0o600))
	in.held = readHeld(t.Context(), targets)
	in.nameFlag = ""
	in.force = false
	in.binary = "databricks"
	plans, err = planAdds(t.Context(), in, &mcp.State{}, []string{"system.ai.github"})
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.True(t, plans[0].reasserted[claude])
	assert.False(t, plans[0].replaced[claude])
	assert.False(t, plans[0].updated[claude])

	// The same entry under a binary that moved is updated, not re-asserted: the write changes its command.
	in.binary = "/opt/databricks/bin/databricks"
	plans, err = planAdds(t.Context(), in, &mcp.State{}, []string{"system.ai.github"})
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.True(t, plans[0].updated[claude])
	assert.False(t, plans[0].reasserted[claude])
	req = addRequest{profile: "logfood", agents: targets}
	assert.Equal(t, []string{"Plan (profile logfood):", fmt.Sprintf("  %-14s %s", claude.DisplayName, "update github-logfood")}, addPlanLines(req, plans))
}

// TestValidateMCPAddFlagsNameNeedsServicesFromNames checks that --name without --names is refused before the
// workspace is read, in a terminal too, where the picker would otherwise choose the services.
func TestValidateMCPAddFlagsNameNeedsServicesFromNames(t *testing.T) {
	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	err := validateMCPAddFlags(ctx, mcpAddFlags{name: "github"}, nil, nil, false)
	require.Error(t, err)
	assert.Equal(t, "--name needs exactly one service; leave out --name to pick services in the picker", err.Error())
	assert.NoError(t, validateMCPAddFlags(ctx, mcpAddFlags{name: "github"}, []string{"github"}, nil, false))
}

func TestChooseAgentsSources(t *testing.T) {
	withHome(t)
	writeFakeCLI(t, agents.ByName(agents.NameCodex).Binary)
	ctx := cmdio.MockDiscard(t.Context())
	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })

	var offered []agentChoice
	promptAgentSelection = func(_ context.Context, _ string, choices []agentChoice) ([]*agents.Agent, error) {
		offered = choices
		return []*agents.Agent{choices[0].agent}, nil
	}

	named := []*agents.Agent{agents.ByName(agents.NameCodex)}
	got, source, err := chooseAgents(ctx, named, true)
	require.NoError(t, err)
	assert.Equal(t, agentsNamed, source)
	assert.Equal(t, named, got)
	assert.Empty(t, offered, "named agents are not asked for")

	got, source, err = chooseAgents(ctx, nil, true)
	require.NoError(t, err)
	assert.Equal(t, agentsPicked, source)
	require.Len(t, got, 1)
	// The picker offers installed agents only.
	require.NotEmpty(t, offered)
	for _, c := range offered {
		installed, err := c.agent.InstalledForMCP(ctx)
		require.NoError(t, err)
		assert.True(t, installed, c.agent.Name)
	}
}

// TestChooseAgentsNamesTheAgentsNotFound checks that the picker lists the installed agents and names the others above it.
func TestChooseAgentsNamesTheAgentsNotFound(t *testing.T) {
	withHome(t)
	writeFakeCLI(t, agents.ByName(agents.NameCodex).Binary)
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	origPrompt := promptAgentSelection
	t.Cleanup(func() { promptAgentSelection = origPrompt })
	promptAgentSelection = func(_ context.Context, _ string, choices []agentChoice) ([]*agents.Agent, error) {
		return []*agents.Agent{choices[0].agent}, nil
	}

	_, _, err := chooseAgents(ctx, nil, true)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Not found: ")
	assert.Contains(t, stderr.String(), agents.ByName(agents.NameCursor).DisplayName)
}

// TestAddPlanLinesSayWhenConfigCannotBeRead checks that an agent whose config cannot be read is shown as such, with the
// write that will fail, not as an add.
func TestAddPlanLinesSayWhenConfigCannotBeRead(t *testing.T) {
	claude := agents.ByName(agents.NameClaudeCode)
	req := addRequest{
		profile:    "logfood",
		agents:     []*agents.Agent{claude},
		unreadable: map[*agents.Agent]error{claude: errors.New("parse error")},
	}
	plans := []addPlan{{service: mcp.Service{FullName: "system.ai.github"}, name: "github-logfood"}}
	assert.Equal(t, []string{"Plan (profile logfood):", fmt.Sprintf("  %-14s cannot be read: parse error; the write will fail", claude.DisplayName)}, addPlanLines(req, plans))
}

func TestAddPlanLines(t *testing.T) {
	claude := agents.ByName(agents.NameClaudeCode)
	codex := agents.ByName(agents.NameCodex)
	req := addRequest{profile: "logfood", agents: []*agents.Agent{claude, codex}}
	plans := []addPlan{
		{service: mcp.Service{FullName: "system.ai.github"}, name: "github-logfood"},
		{
			service:    mcp.Service{FullName: "system.ai.genie_one_mcp"},
			name:       "genie_one_mcp-logfood",
			reasserted: map[*agents.Agent]bool{claude: true},
		},
	}
	want := []string{
		"Plan (profile logfood):",
		fmt.Sprintf("  %-14s %s", claude.DisplayName, "add github-logfood; re-assert genie_one_mcp-logfood"),
		fmt.Sprintf("  %-14s %s", codex.DisplayName, "add github-logfood, genie_one_mcp-logfood"),
	}
	assert.Equal(t, want, addPlanLines(req, plans))

	// A server that an agent cannot take is skipped for that agent, and a refused agent is not written.
	req.refused = map[*agents.Agent]error{codex: errNotInstalled}
	plans[0].failed = map[*agents.Agent]error{claude: errors.New("conflict")}
	want = []string{
		"Plan (profile logfood):",
		fmt.Sprintf("  %-14s %s", claude.DisplayName, "re-assert genie_one_mcp-logfood; skip github-logfood"),
		fmt.Sprintf("  %-14s %s", codex.DisplayName, "not installed; skipped"),
	}
	assert.Equal(t, want, addPlanLines(req, plans))

	// An agent whose check failed is refused with the reason, not as not installed.
	req.refused = map[*agents.Agent]error{codex: errors.New("no home")}
	want = []string{
		"Plan (profile logfood):",
		fmt.Sprintf("  %-14s %s", claude.DisplayName, "re-assert genie_one_mcp-logfood; skip github-logfood"),
		fmt.Sprintf("  %-14s %s", codex.DisplayName, "cannot be checked: no home; skipped"),
	}
	assert.Equal(t, want, addPlanLines(req, plans))
}

func TestValidateMCPAddFlagsInTerminalAsksForMissingChoices(t *testing.T) {
	ctx, test := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	defer test.Done()
	require.NoError(t, validateMCPAddFlags(ctx, mcpAddFlags{}, nil, nil, false))
}
