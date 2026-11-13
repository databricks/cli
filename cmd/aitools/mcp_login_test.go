package aitools

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpSignInRequiresFullServiceName(t *testing.T) {
	for _, service := range []string{"github", "system.github", "system..github", "a.b.c.d"} {
		t.Run(service, func(t *testing.T) {
			var out mcpSignInOutput
			err := runMcpSignIn(cmdio.MockDiscard(t.Context()), &bytes.Buffer{}, "", service, mcpLogin, &out)
			require.Error(t, err)
			assert.Equal(t, `service must be a full name such as system.ai.github, got "`+service+`"`, err.Error())
			assert.Equal(t, protos.AitoolsErrorCategoryInvalidFlags, errorCategory(err))
		})
	}
}

// loginTestServer returns a server that answers every API call with the status and body given. The SDK still reads
// host metadata when it configures a profile, so paths outside /api/ are not found.
func loginTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestMcpLoginRefusesDriftBeforeAnyRequest checks that login refuses a profile whose host changed since the server was
// installed, with the same message as add and serve, before it sends a request to the profile's host.
func TestMcpLoginRefusesDriftBeforeAnyRequest(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	saveEntry(t, mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test"})

	out := mcpSignInOutput{Service: "system.ai.github"}
	err := runMcpSignIn(t.Context(), io.Discard, "logfood", "system.ai.github", mcpLogin, &out)
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileHostChanged), errorCategory(err))
	assert.Contains(t, err.Error(), "MCP server github-logfood was set up for https://adb-123.test, but profile \"logfood\" now points at ")
	assert.Contains(t, err.Error(), "aitools mcp remove --names github-logfood")
}

// TestMcpLoginMissingProfileMatchesAdd checks that login names a missing profile the way add and serve do, with the same
// message, rather than with a message of its own.
func TestMcpLoginMissingProfileMatchesAdd(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[other]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-other\n")

	out := mcpSignInOutput{Service: "system.ai.github"}
	loginErr := runMcpSignIn(t.Context(), io.Discard, "logfood", "system.ai.github", mcpLogin, &out)
	require.Error(t, loginErr)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindProfileNotFound), errorCategory(loginErr))

	addErr := runAdd(t, "--profile", "logfood", "--names", "github", "--agents", "claude-code")
	require.Error(t, addErr)
	assert.Equal(t, addErr.Error(), loginErr.Error())
}

// TestMcpLoginRejectedTokenIsAuthFailed checks that a credential the workspace rejects on the first API call is
// AUTH_FAILED with the login command for the profile, as it is in add.
func TestMcpLoginRejectedTokenIsAuthFailed(t *testing.T) {
	home := withHome(t)
	server := loginTestServer(t, http.StatusUnauthorized, `{"error_code":"UNAUTHENTICATED","message":"Invalid access token."}`)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-expired\n")

	out := mcpSignInOutput{Service: "system.ai.github"}
	err := runMcpSignIn(t.Context(), io.Discard, "logfood", "system.ai.github", mcpLogin, &out)
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(mcp.BindAuthFailed), errorCategory(err))
	assert.Contains(t, err.Error(), "Run: databricks auth login --profile logfood")
}

// TestMcpLogoutUnknownServiceIsAnError checks that logout of a service the profile cannot see fails, rather than
// reporting that the caller is not signed in, so that a typo in the name is seen.
func TestMcpLogoutUnknownServiceIsAnError(t *testing.T) {
	home := withHome(t)
	server := loginTestServer(t, http.StatusNotFound, `{"error_code":"NOT_FOUND","message":"not found"}`)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	out := mcpSignInOutput{Service: "system.ai.gihtub"}
	err := runMcpSignIn(t.Context(), io.Discard, "logfood", "system.ai.gihtub", mcpLogout, &out)
	require.Error(t, err)
	assert.Equal(t, protos.AitoolsErrorCategory(categoryServiceNotFound), errorCategory(err))
	assert.Contains(t, err.Error(), "system.ai.gihtub is not an MCP service visible to profile logfood")
	assert.Empty(t, out.State)
}

// TestSignInInstalledSignsInToEveryServer checks that add --login attempts every installed sign-in server after one of
// them fails. The failed server gets the login hint, and the failure is returned, so the others are still signed in.
func TestSignInInstalledSignsInToEveryServer(t *testing.T) {
	const (
		githubPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
		geniePath  = "/api/2.1/unity-catalog/mcp-services/system.ai.genie_one_mcp/user-credentials"
	)
	var mu sync.Mutex
	var reads []string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		mu.Lock()
		reads = append(reads, r.URL.Path)
		mu.Unlock()
		rw.Header().Set("Content-Type", "application/json")
		if r.URL.Path == githubPath {
			rw.WriteHeader(http.StatusForbidden)
			_, _ = rw.Write([]byte(`{"error_code":"PERMISSION_DENIED","message":"no access"}`))
			return
		}
		_, _ = rw.Write([]byte(`{"provisioning_info":{"state":"ACTIVE"}}`))
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	req := addRequest{profile: "logfood", workspaceID: "123", agents: []*agents.Agent{agents.ByName("claude-code")}}
	plans := []addPlan{
		{service: mcp.Service{FullName: "system.ai.github", SignInRequired: true}},
		{service: mcp.Service{FullName: "system.ai.genie_one_mcp", SignInRequired: true}},
	}
	installed := []addAgentOutput{{Name: "claude-code", Status: statusInstalled}}
	servers := []addServerOutput{
		{Name: "github-logfood", Service: "system.ai.github", SignInRequired: true, Agents: installed},
		{Name: "genie-logfood", Service: "system.ai.genie_one_mcp", SignInRequired: true, Agents: installed},
	}

	var errOut bytes.Buffer
	err = signInInstalled(t.Context(), &errOut, w, req, plans, servers)
	require.Error(t, err)
	assert.Equal(t, []string{githubPath, geniePath}, reads)
	assert.Contains(t, errOut.String(), "system.ai.github needs sign-in. Run: databricks aitools mcp login system.ai.github --profile logfood\n")
	assert.NotContains(t, errOut.String(), "system.ai.genie_one_mcp needs sign-in")
}
