package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpSignInRequiresFullServiceName(t *testing.T) {
	// A name with a character that is URL syntax, or a control character, is refused before any request: the request
	// path would name another resource.
	for _, service := range []string{"github", "system.github", "system..github", "a.b.c.d", "system.ai.git#hub", "system.ai.git?hub", "system.ai.git/hub", "system.ai.git%hub", "system.ai.git\nhub"} {
		t.Run(service, func(t *testing.T) {
			var out mcpSignInOutput
			err := runMcpSignIn(cmdio.MockDiscard(t.Context()), &bytes.Buffer{}, "", service, mcpLogin, &out)
			require.Error(t, err)
			assert.Equal(t, fmt.Sprintf("service must be a full name such as system.ai.github, got %q", service), err.Error())
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
	assert.Equal(t, categoryServiceNotFound, errorCategory(err))
	assert.Contains(t, err.Error(), "system.ai.gihtub is not an MCP service visible to profile logfood")
	assert.Empty(t, out.State)
}

// TestMcpLogoutRemovesCredentialOfDeletedService checks that logout removes a credential whose service is gone from the
// profile's view, because its source connection was deleted. The service lookup fails, but the credential is still there.
func TestMcpLogoutRemovesCredentialOfDeletedService(t *testing.T) {
	home := withHome(t)
	const (
		servicePath    = "/api/2.1/unity-catalog/mcp-services/system.ai.github"
		credentialPath = servicePath + "/user-credentials"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == servicePath && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"name":"mcp-services/system.ai.github","config":{"source_connection":{"is_deleted":true}}}`))
		case r.URL.Path == credentialPath && r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not found"}`))
		}
	}))
	t.Cleanup(server.Close)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	out := mcpSignInOutput{Service: "system.ai.github"}
	err := runMcpSignIn(t.Context(), io.Discard, "logfood", "system.ai.github", mcpLogout, &out)
	require.NoError(t, err)
	assert.Equal(t, signOutStateDeleted, out.State)
}

// TestMcpLogoutEndpointNotFoundIsNotAServiceError checks that a 404 for an endpoint the API does not have is not read as a
// service the profile cannot see, which would send the user to look for a typo in the name.
func TestMcpLogoutEndpointNotFoundIsNotAServiceError(t *testing.T) {
	home := withHome(t)
	server := loginTestServer(t, http.StatusNotFound, `{"error_code":"ENDPOINT_NOT_FOUND","message":"endpoint is not available"}`)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")

	out := mcpSignInOutput{Service: "system.ai.github"}
	err := runMcpSignIn(t.Context(), io.Discard, "logfood", "system.ai.github", mcpLogout, &out)
	require.Error(t, err)
	assert.NotEqual(t, categoryServiceNotFound, errorCategory(err))
	assert.NotContains(t, err.Error(), "is not an MCP service visible")
}

// TestSignInInstalledSignsInToEveryServer checks that add --login attempts every installed sign-in server after one of
// them fails. The failure is returned, and the other servers are still signed in.
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

	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	var errOut bytes.Buffer
	err = signInInstalled(ctx, &errOut, w, req, plans, servers)
	require.Error(t, err)
	assert.Equal(t, []string{githubPath, geniePath}, reads)
	// The workspace refused the sign-in with PERMISSION_DENIED, which a login does not fix, so no login hint is printed.
	assert.NotContains(t, stderr.String(), "needs sign-in")
}

// TestSignInInstalledRecordsTheSignInOfEachServer checks that each server of add --login records its sign-in, with the state
// that login reports: a server that does not need sign-in, one that is signed in, and one whose sign-in failed. A server
// that reached no agent gets no sign-in, since none was attempted.
func TestSignInInstalledRecordsTheSignInOfEachServer(t *testing.T) {
	t.Setenv("BROWSER", "none")
	prevPoll := signInPollInterval
	t.Cleanup(func() { signInPollInterval = prevPoll })
	signInPollInterval = time.Millisecond
	const (
		githubPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		if r.URL.Path == githubPath {
			_, _ = rw.Write([]byte(`{"provisioning_info":{"state":"FAILED"}}`))
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
		{service: mcp.Service{FullName: "system.ai.plain"}},
		{service: mcp.Service{FullName: "system.ai.unused", SignInRequired: true}},
	}
	installed := []addAgentOutput{{Name: "claude-code", Status: statusInstalled}}
	servers := []addServerOutput{
		{Name: "github-logfood", Service: "system.ai.github", SignInRequired: true, Agents: installed},
		{Name: "genie-logfood", Service: "system.ai.genie_one_mcp", SignInRequired: true, Agents: installed},
		{Name: "plain-logfood", Service: "system.ai.plain", Agents: installed},
		{Name: "unused-logfood", Service: "system.ai.unused", SignInRequired: true, Agents: []addAgentOutput{{Name: "claude-code", Status: statusFailed}}},
	}

	ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
	var errOut bytes.Buffer
	require.Error(t, signInInstalled(ctx, &errOut, w, req, plans, servers))

	require.NotNil(t, servers[0].SignIn)
	assert.Equal(t, signInStateFailed, servers[0].SignIn.State)
	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, servers[0].SignIn.ErrorCategory)
	assert.Contains(t, servers[0].SignIn.Error, "sign-in to system.ai.github failed on profile logfood")
	require.NotNil(t, servers[1].SignIn)
	assert.Equal(t, signInStateActive, servers[1].SignIn.State)
	assert.Empty(t, servers[1].SignIn.Error)
	require.NotNil(t, servers[2].SignIn)
	assert.Equal(t, signInStateNotRequired, servers[2].SignIn.State)
	assert.Nil(t, servers[3].SignIn)
}

// signInInstalledOne signs in to one installed sign-in server of the test's workspace and returns the error and what the
// command wrote to stderr, on either stream. The server answers with status and body for the credential.
func signInInstalledOne(t *testing.T, status int, body string) (string, error) {
	t.Helper()
	// BROWSER=none keeps the test from opening a browser; the page is only printed.
	t.Setenv("BROWSER", "none")
	const githubPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == githubPath {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(status)
			_, _ = rw.Write([]byte(body))
			return
		}
		rw.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	req := addRequest{profile: "logfood", workspaceID: "123", agents: []*agents.Agent{agents.ByName("claude-code")}}
	plans := []addPlan{{service: mcp.Service{FullName: "system.ai.github", SignInRequired: true}}}
	servers := []addServerOutput{{
		Name: "github-logfood", Service: "system.ai.github", SignInRequired: true,
		Agents: []addAgentOutput{{Name: "claude-code", Status: statusInstalled}},
	}}
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	var errOut bytes.Buffer
	err = signInInstalled(ctx, &errOut, w, req, plans, servers)
	return errOut.String() + stderr.String(), err
}

// TestSignInTimeoutNamesTheLastFailedCheck checks that a sign-in that times out after failed polls names the last failure in
// its error, so that the user can see why the credential did not become ACTIVE, and that its state is TIMEOUT.
func TestSignInTimeoutNamesTheLastFailedCheck(t *testing.T) {
	t.Setenv("BROWSER", "none")
	prevPoll, prevWait := signInPollInterval, signInWaitTimeout
	t.Cleanup(func() { signInPollInterval, signInWaitTimeout = prevPoll, prevWait })
	signInPollInterval, signInWaitTimeout = time.Millisecond, 100*time.Millisecond

	const githubPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	var mu sync.Mutex
	gets := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != githubPath {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		mu.Lock()
		gets++
		first := gets == 1
		mu.Unlock()
		rw.Header().Set("Content-Type", "application/json")
		if first {
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write([]byte(`{"error_code":"NOT_FOUND","message":"Credential is not found"}`))
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"try later"}`))
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
	state, _, _, err := signIn(ctx, io.Discard, w, "logfood", "", "https://example.test/login", mcp.Service{FullName: "system.ai.github", SignInRequired: true},
		func(context.Context, string) error { return nil })
	require.Error(t, err)
	assert.Equal(t, signInStateTimeout, state)
	_, timedOut := errors.AsType[*mcp.SignInTimeoutError](err)
	assert.True(t, timedOut)
	assert.Contains(t, err.Error(), "(the last check failed: ")
	assert.Contains(t, err.Error(), "try later")
}

// TestSignInInstalledHintIsNotRepeatedWhenTheErrorNamesTheCommand checks that a sign-in that failed does not print the
// login hint again, since its error already names the command to run.
func TestSignInInstalledHintIsNotRepeatedWhenTheErrorNamesTheCommand(t *testing.T) {
	output, err := signInInstalledOne(t, http.StatusOK, `{"provisioning_info":{"state":"FAILED"}}`)
	require.ErrorIs(t, err, mcp.ErrSignInFailed)
	assert.Contains(t, err.Error(), "Run: databricks aitools mcp login system.ai.github --profile logfood")
	assert.NotContains(t, output, "needs sign-in")
}

// TestSignInInstalledNoLoginHintForFailureLoginDoesNotFix checks that a sign-in the workspace refuses gets no login hint:
// logging in again would not fix a refusal, and the hint would be printed in JSON mode too.
func TestSignInInstalledNoLoginHintForFailureLoginDoesNotFix(t *testing.T) {
	output, err := signInInstalledOne(t, http.StatusForbidden, `{"error_code":"PERMISSION_DENIED","message":"no access"}`)
	require.ErrorIs(t, err, apierr.ErrPermissionDenied)
	assert.NotContains(t, output, "needs sign-in")
}

// TestSignInInstalledAuthFailureNamesTheAuthCommand checks that a credential the workspace rejects gets the auth command
// in its error, and no login hint, which would not fix it.
func TestSignInInstalledAuthFailureNamesTheAuthCommand(t *testing.T) {
	output, err := signInInstalledOne(t, http.StatusUnauthorized, `{"error_code":"UNAUTHENTICATED","message":"Invalid access token."}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Run: databricks auth login --profile logfood")
	assert.NotContains(t, output, "mcp login")
	assert.NotContains(t, output, "needs sign-in")
}

// TestMcpLoginHint checks the login command that a hint prints: it can be pasted into a shell as written, and a server
// pinned to a config file sets DATABRICKS_CONFIG_FILE, so that the login reads the file the profile is in.
func TestMcpLoginHint(t *testing.T) {
	tests := []struct {
		name, profile, pinnedFile, want string
	}{
		{
			name:    "plain profile",
			profile: "logfood",
			want:    "databricks aitools mcp login system.ai.github --profile logfood",
		},
		{
			name:    "profile name that needs quoting",
			profile: "my profile",
			want:    "databricks aitools mcp login system.ai.github --profile 'my profile'",
		},
		{
			name:       "pinned to a config file",
			profile:    "logfood",
			pinnedFile: "/work/cfg/logfood.cfg",
			want:       "DATABRICKS_CONFIG_FILE=/work/cfg/logfood.cfg databricks aitools mcp login system.ai.github --profile logfood",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mcpLoginHint("system.ai.github", tt.profile, tt.pinnedFile))
		})
	}
}

// TestMcpLoginJSONURLIsNamedOnlyWhenThePageIsShown checks that the document names the sign-in URL when a page was shown,
// or when the sign-in failed and the URL is how to finish it. A failure before the page is shown names no URL.
func TestMcpLoginJSONURLIsNamedOnlyWhenThePageIsShown(t *testing.T) {
	t.Setenv("BROWSER", "none")
	prevPoll := signInPollInterval
	t.Cleanup(func() { signInPollInterval = prevPoll })
	signInPollInterval = time.Millisecond
	cases := []struct {
		name           string
		credentialCode int
		credentialBody string
		wantURL        bool
	}{
		{
			name:           "the credential cannot be read before the page",
			credentialCode: http.StatusForbidden,
			credentialBody: `{"error_code":"PERMISSION_DENIED","message":"no access"}`,
		},
		{
			name:           "the sign-in fails after the page",
			credentialCode: http.StatusOK,
			credentialBody: `{"provisioning_info":{"state":"FAILED"}}`,
			wantURL:        true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := withHome(t)
			server := listCheckServer(t, http.StatusOK, tc.credentialCode, tc.credentialBody)
			writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\nworkspace_id = 123\n")

			out := mcpSignInOutput{Service: "system.ai.github"}
			err := runMcpSignIn(t.Context(), io.Discard, "logfood", "system.ai.github", mcpLogin, &out)
			require.Error(t, err)
			if tc.wantURL {
				assert.Equal(t, server.URL+"/mcp-service-login?name=system.ai.github&o=123", out.URL)
			} else {
				assert.Empty(t, out.URL)
			}
		})
	}
}

// TestMcpLoginJSONNamesProfileWhenBindingFails checks that a failure after the profile is resolved still names the profile
// in the document, and that a state the command never reached is left out rather than sent empty.
func TestMcpLoginJSONNamesProfileWhenBindingFails(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	saveEntry(t, mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test"})

	parent := newMCPTestRoot()
	var stdout bytes.Buffer
	parent.SetOut(&stdout)
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "login", "system.ai.github", "--profile", "logfood", "--output", "json"})
	err := parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var out mcpSignInOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	assert.Equal(t, "logfood", out.Profile)
	assert.Equal(t, "PROFILE_HOST_CHANGED", string(out.ErrorCategory))
	assert.NotContains(t, stdout.String(), `"state"`)
}

// TestMcpLoginJSONNamesProfileWhenStateCannotBeRead checks that a failure to read the state, which comes after the profile is
// resolved, still names the profile in the document.
func TestMcpLoginJSONNamesProfileWhenStateCannotBeRead(t *testing.T) {
	withHome(t)
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".state.json"), []byte("{not json"), 0o600))

	parent := newMCPTestRoot()
	var stdout bytes.Buffer
	parent.SetOut(&stdout)
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "login", "system.ai.github", "--profile", "logfood", "--output", "json"})
	err = parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var out mcpSignInOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out), stdout.String())
	assert.Equal(t, "logfood", out.Profile)
	assert.Equal(t, protos.AitoolsErrorCategoryStateInvalid, out.ErrorCategory)
}

// TestMcpSignInJSONWriteFailureKeepsRunError checks that a document that cannot be written does not hide why the command
// failed, as add and remove keep it.
func TestMcpSignInJSONWriteFailureKeepsRunError(t *testing.T) {
	home := withHome(t)
	server := noRequestServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-logfood\n")
	saveEntry(t, mcp.Entry{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: "https://adb-123.test"})

	parent := newMCPTestRoot()
	parent.SetOut(failingWriter{})
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "login", "system.ai.github", "--profile", "logfood", "--output", "json"})
	err := parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
	require.Error(t, err)
	assert.ErrorContains(t, err, "disk full")
	assert.ErrorContains(t, err, "MCP server github-logfood was set up for https://adb-123.test")
}
