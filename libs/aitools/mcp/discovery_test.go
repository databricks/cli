package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const discoveryTestListPath = "/api/2.1/unity-catalog/mcp-services"

// discoveryTestGitHubProvider is the oauth_provider the API reports for GitHub.
const discoveryTestGitHubProvider = "OAUTH_PROVIDER_GITHUB_MCP"

// discoveryTestServe starts a fake list endpoint that answers request i with pages[i] and
// returns a client for it. It fails the test on a wrong path or query and counts requests.
func discoveryTestServe(t *testing.T, location string, pages ...string) (*databricks.WorkspaceClient, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// The SDK resolves host metadata before the first API call; that is not a list call.
		if r.URL.Path != discoveryTestListPath {
			http.NotFound(rw, r)
			return
		}
		n := int(calls.Add(1))
		assert.Equal(t, "schemas/"+location, r.URL.Query().Get("parent"))
		assert.Equal(t, "FULL", r.URL.Query().Get("view"))
		if n > len(pages) {
			http.Error(rw, `{"error_code":"INTERNAL_ERROR","message":"unexpected call"}`, http.StatusInternalServerError)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(rw, pages[n-1])
	}))
	t.Cleanup(server.Close)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	return w, &calls
}

// discoveryTestPage marshals one list response page.
func discoveryTestPage(t *testing.T, nextPageToken string, services ...catalog.McpService) string {
	t.Helper()
	body, err := json.Marshal(catalog.ListMcpServicesResponse{McpServices: services, NextPageToken: nextPageToken})
	require.NoError(t, err)
	return string(body)
}

// discoveryTestInternal is a Databricks-owned service with no source connection.
func discoveryTestInternal(fullName string) catalog.McpService {
	return catalog.McpService{Name: mcpServicePrefix + fullName}
}

// discoveryTestExternal is a service backed by a source connection with the given options.
func discoveryTestExternal(fullName string, options map[string]string) catalog.McpService {
	return catalog.McpService{
		Name: mcpServicePrefix + fullName,
		Config: &catalog.McpServiceConfig{
			SourceConnection: &catalog.McpServiceConfigSourceConnection{
				Name:    "connections/" + fullName,
				Options: options,
			},
		},
	}
}

func TestDiscoverMakesOneCallForSingleSchemaPage(t *testing.T) {
	w, calls := discoveryTestServe(t, "system.ai",
		discoveryTestPage(t, "", discoveryTestInternal("system.ai.genie_one_mcp"), discoveryTestExternal("system.ai.github", map[string]string{oauthProviderOption: discoveryTestGitHubProvider})),
	)

	services, err := Discover(t.Context(), w, "system.ai")
	require.NoError(t, err)
	assert.Len(t, services, 2)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDiscoverFollowsPagination(t *testing.T) {
	w, calls := discoveryTestServe(t, "main.tools",
		discoveryTestPage(t, "page-2", discoveryTestInternal("main.tools.crm")),
		discoveryTestPage(t, "", discoveryTestInternal("main.tools.billing")),
	)

	services, err := Discover(t.Context(), w, "main.tools")
	require.NoError(t, err)
	require.Len(t, services, 2)
	assert.Equal(t, "main.tools.crm", services[0].FullName)
	assert.Equal(t, "main.tools.billing", services[1].FullName)
	assert.Equal(t, int32(2), calls.Load())
}

func TestDiscoverDetectsSignInFromOptions(t *testing.T) {
	tests := []struct {
		name         string
		svc          catalog.McpService
		wantSignIn   bool
		wantProvider string
	}{
		{
			name:       "internal service has no source connection",
			svc:        discoveryTestInternal("main.tools.crm"),
			wantSignIn: false,
		},
		{
			name:         "source connection with oauth_provider needs sign-in",
			svc:          discoveryTestExternal("main.tools.crm", map[string]string{oauthProviderOption: discoveryTestGitHubProvider}),
			wantSignIn:   true,
			wantProvider: discoveryTestGitHubProvider,
		},
		{
			name:       "source connection without oauth_provider does not need sign-in",
			svc:        discoveryTestExternal("main.tools.crm", map[string]string{"client_id": "abc"}),
			wantSignIn: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := discoveryTestServe(t, "main.tools", discoveryTestPage(t, "", tt.svc))

			services, err := Discover(t.Context(), w, "main.tools")
			require.NoError(t, err)
			require.Len(t, services, 1)
			assert.Equal(t, tt.wantSignIn, services[0].SignInRequired)
			assert.Equal(t, tt.wantProvider, services[0].OAuthProvider)
		})
	}
}

func TestDiscoverDropsDeletedSourceConnections(t *testing.T) {
	deleted := discoveryTestExternal("main.tools.old", map[string]string{oauthProviderOption: discoveryTestGitHubProvider})
	deleted.Config.SourceConnection.IsDeleted = true

	w, _ := discoveryTestServe(t, "main.tools",
		discoveryTestPage(t, "", deleted, discoveryTestInternal("main.tools.crm")),
	)

	services, err := Discover(t.Context(), w, "main.tools")
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Equal(t, "main.tools.crm", services[0].FullName)
	assert.Equal(t, "crm", services[0].Leaf)
}

func TestDiscoverAttachesKnownOnlyInSystemAI(t *testing.T) {
	github := discoveryTestExternal("system.ai.github", map[string]string{oauthProviderOption: discoveryTestGitHubProvider})
	github.Comment = "API comment"
	custom := discoveryTestInternal("system.ai.custom")

	w, _ := discoveryTestServe(t, "system.ai", discoveryTestPage(t, "", github, custom))
	services, err := Discover(t.Context(), w, "system.ai")
	require.NoError(t, err)
	require.Len(t, services, 2)
	require.NotNil(t, services[0].Known)
	assert.Equal(t, "GitHub", services[0].Known.Label)
	assert.Equal(t, "API comment", services[0].Comment)
	assert.Nil(t, services[1].Known)

	w, _ = discoveryTestServe(t, "main.tools", discoveryTestPage(t, "", discoveryTestExternal("main.tools.github", map[string]string{oauthProviderOption: discoveryTestGitHubProvider})))
	services, err = Discover(t.Context(), w, "main.tools")
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Nil(t, services[0].Known)
}

func TestDiscoverTakesLeafFromReturnedNameForMixedCaseLocation(t *testing.T) {
	w, _ := discoveryTestServe(t, "System.AI", discoveryTestPage(t, "", discoveryTestExternal("system.ai.github", map[string]string{oauthProviderOption: discoveryTestGitHubProvider})))
	services, err := Discover(t.Context(), w, "System.AI")
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Equal(t, "github", services[0].Leaf)
	require.NotNil(t, services[0].Known)
	assert.Equal(t, "GitHub", services[0].Known.Label)

	w, _ = discoveryTestServe(t, "Main.Tools", discoveryTestPage(t, "", discoveryTestInternal("main.tools.crm")))
	services, err = Discover(t.Context(), w, "Main.Tools")
	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Equal(t, "crm", services[0].Leaf)
	assert.Nil(t, services[0].Known)
}

func TestDiscoverMapsListingErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantKind ListErrorKind
		wantSDK  error
		wantMsg  string
	}{
		{
			name:     "missing schema is location not found",
			status:   http.StatusNotFound,
			body:     `{"error_code":"NOT_FOUND","message":"Schema 'main.tools' does not exist"}`,
			wantKind: ListErrLocationNotFound,
			wantSDK:  apierr.ErrNotFound,
			wantMsg:  "main.tools is not a schema visible to this profile: Schema 'main.tools' does not exist",
		},
		{
			name:     "forbidden is permission denied",
			status:   http.StatusForbidden,
			body:     `{"error_code":"PERMISSION_DENIED","message":"User does not have USE SCHEMA"}`,
			wantKind: ListErrPermissionDenied,
			wantSDK:  apierr.ErrPermissionDenied,
			wantMsg:  "cannot list MCP services in main.tools (listing needs USE_CATALOG on the catalog and USE_SCHEMA on the schema): User does not have USE SCHEMA",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(tt.status)
				_, _ = io.WriteString(rw, tt.body)
			}))
			t.Cleanup(server.Close)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
			require.NoError(t, err)

			_, err = Discover(t.Context(), w, "main.tools")
			var listErr *ListError
			require.ErrorAs(t, err, &listErr)
			assert.Equal(t, tt.wantKind, listErr.Kind)
			assert.Equal(t, "main.tools", listErr.Location)
			assert.ErrorIs(t, err, tt.wantSDK)
			assert.EqualError(t, err, tt.wantMsg)
		})
	}
}

func TestMapListError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantKind ListErrorKind
	}{
		{name: "rate limited", err: apierr.ErrTooManyRequests, wantKind: ListErrRateLimited},
		{name: "resource exhausted is rate limited", err: apierr.ErrResourceExhausted, wantKind: ListErrRateLimited},
		{name: "resource does not exist is location not found", err: apierr.ErrResourceDoesNotExist, wantKind: ListErrLocationNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var listErr *ListError
			require.ErrorAs(t, mapListError("system.ai", tt.err), &listErr)
			assert.Equal(t, tt.wantKind, listErr.Kind)
		})
	}
}

func TestListErrorMessages(t *testing.T) {
	cause := errors.New("server said no")
	tests := []struct {
		kind ListErrorKind
		want string
	}{
		{kind: ListErrRateLimited, want: "MCP service listing in system.ai was rate limited; retry in a minute: server said no"},
		{kind: "", want: "cannot list MCP services in system.ai: server said no"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			err := &ListError{Kind: tt.kind, Location: "system.ai", Err: cause}
			assert.EqualError(t, err, tt.want)
		})
	}
}

func TestMapListErrorLeavesWorkspaceAndNetworkFailuresUnclassified(t *testing.T) {
	tests := []struct {
		code   string
		status int
	}{
		{code: "ENDPOINT_NOT_FOUND", status: http.StatusNotFound},
		{code: "FEATURE_DISABLED", status: http.StatusNotFound},
		{code: "PRIVATE_LINK_VALIDATION_ERROR", status: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			apiErr := &apierr.APIError{ErrorCode: tt.code, StatusCode: tt.status, Message: "server message"}
			err := mapListError("system.ai", apiErr)

			_, ok := errors.AsType[*ListError](err)
			assert.False(t, ok)
			assert.ErrorIs(t, err, apiErr)
			assert.EqualError(t, err, "cannot list MCP services in system.ai: server message")
		})
	}
}

func TestMapListErrorLeavesOtherErrorsUnclassified(t *testing.T) {
	err := mapListError("system.ai", apierr.ErrUnauthenticated)

	_, ok := errors.AsType[*ListError](err)
	assert.False(t, ok)
	assert.ErrorIs(t, err, apierr.ErrUnauthenticated)
}

func TestGetServiceSignInRequired(t *testing.T) {
	s := &loginTestServer{}
	w := loginTestClient(t, s)

	svc, err := GetService(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.True(t, svc.SignInRequired)
	assert.Equal(t, "OAUTH_PROVIDER_GITHUB_MCP", svc.OAuthProvider)
	require.NotNil(t, svc.Known)
	assert.Equal(t, "GitHub", svc.Known.Label)
}

// TestGetServiceDeletedSourceIsNotFound checks that a service whose source connection was deleted is NOT_FOUND, as
// Discover drops it, so that login and logout treat it the same way as add does.
func TestGetServiceDeletedSourceIsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(rw, `{"name":"mcp-services/system.ai.github","config":{"source_connection":{"is_deleted":true,"options":{"oauth_provider":"OAUTH_PROVIDER_GITHUB_MCP"}}}}`)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	_, err = GetService(t.Context(), w, loginTestService)
	require.ErrorIs(t, err, apierr.ErrNotFound)
}
