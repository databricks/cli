package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testproxy"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func StartDefaultServer(t *testing.T, logRequests bool) {
	s := testserver.New(t)
	testserver.AddDefaultHandlers(s)

	// Log API responses if the -logrequests flag is set.
	if logRequests {
		s.ResponseCallback = logResponseCallback(t)
	}

	t.Setenv("DATABRICKS_DEFAULT_HOST", s.URL)

	// Do not read user's ~/.databrickscfg.
	//
	// We use a custom temporary home directory and cleanup routine here to avoid
	// issues observed with t.TempDir() on Windows, where Go 1.25's test cleanup
	// can fail to remove certain directories (e.g., due to locked or system-managed files).
	// Instead of failing the test, we log any errors encountered during cleanup.
	// This approach ensures test reliability across platforms.
	//
	// See debugging journey in https://github.com/databricks/cli/pull/3575.
	homeDir, err := os.MkdirTemp("", "acceptance-home-dir")
	require.NoError(t, err)
	t.Cleanup(func() {
		err := os.RemoveAll(homeDir)
		if err != nil {
			t.Logf("Failed to remove temporary home directory: %v", err)
			_ = filepath.Walk(homeDir, func(path string, info os.FileInfo, err error) error {
				t.Logf("%s", path)
				return nil
			})
		}
	})
	t.Setenv(env.HomeEnvVar(), homeDir)
}

func isTruePtr(value *bool) bool {
	return value != nil && *value
}

func PrepareServerAndClient(t *testing.T, config TestConfig, logRequests bool, outputDir string) (*sdkconfig.Config, iam.User) {
	cloudEnv := os.Getenv("CLOUD_ENV")
	recordRequests := isTruePtr(config.RecordRequests)

	// Use a unique token for each test. This allows us to maintain
	// separate state for each test in fake workspaces.
	tokenSuffix := strings.ReplaceAll(uuid.NewString(), "-", "")

	var token string
	var testUser iam.User
	if isTruePtr(config.IsServicePrincipal) {
		token = testserver.ServicePrincipalTokenPrefix + tokenSuffix
		testUser = testserver.TestUserSP
	} else {
		token = testserver.UserNameTokenPrefix + tokenSuffix
		testUser = testserver.TestUser
	}

	if cloudEnv != "" {
		w, err := databricks.NewWorkspaceClient()
		require.NoError(t, err)

		user, err := w.CurrentUser.Me(context.Background())
		require.NoError(t, err, "Failed to get current user")

		cfg := w.Config

		// If we are running in a cloud environment AND we are recording requests,
		// start a dedicated server to act as a reverse proxy to a real Databricks workspace.
		if recordRequests {
			host := startProxyServer(t, logRequests, config.IncludeRequestHeaders, outputDir)
			cfg = &sdkconfig.Config{
				Host:  host,
				Token: token,
			}
		}

		return cfg, *user
	}

	// If we are not recording requests, and no custom server stubs are configured,
	// use the default shared server.
	if len(config.Server) == 0 && !recordRequests {
		cfg := &sdkconfig.Config{
			Host:  os.Getenv("DATABRICKS_DEFAULT_HOST"),
			Token: token,
		}

		return cfg, testUser
	}

	// Default case. Start a dedicated local server for the test with the server stubs configured
	// as overrides.
	host := startLocalServer(t, config.Server, recordRequests, logRequests, config.IncludeRequestHeaders, outputDir)
	cfg := &sdkconfig.Config{
		Host:  host,
		Token: token,
	}

	// For the purposes of replacements, use testUser for local runs.
	// Note, users might have overriden /api/2.0/preview/scim/v2/Me but that should not affect the replacement:
	return cfg, testUser
}

func recordRequestsCallback(t *testing.T, includeHeaders []string, outputDir string) func(request *testserver.Request) {
	mu := sync.Mutex{}

	return func(request *testserver.Request) {
		mu.Lock()
		defer mu.Unlock()

		req := getLoggedRequest(request, includeHeaders)
		reqJson, err := json.MarshalIndent(req, "", "  ")
		assert.NoErrorf(t, err, "Failed to json-encode: %#v", req)

		requestsPath := filepath.Join(outputDir, "out.requests.txt")
		f, err := os.OpenFile(requestsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		assert.NoError(t, err)
		defer f.Close()

		_, err = f.WriteString(string(reqJson) + "\n")
		assert.NoError(t, err)
	}
}

func logResponseCallback(t *testing.T) func(request *testserver.Request, response *testserver.EncodedResponse) {
	mu := sync.Mutex{}

	return func(request *testserver.Request, response *testserver.EncodedResponse) {
		mu.Lock()
		defer mu.Unlock()

		t.Logf("%d %s %s\n%s\n%s",
			response.StatusCode, request.Method, request.URL,
			formatHeadersAndBody("> ", request.Headers, request.Body),
			formatHeadersAndBody("# ", response.Headers, response.Body),
		)
	}
}

func startLocalServer(t *testing.T,
	stubs []ServerStub,
	recordRequests bool,
	logRequests bool,
	includeHeaders []string,
	outputDir string,
) string {
	s := testserver.New(t)

	// Record API requests in out.requests.txt if RecordRequests is true
	// in test.toml
	if recordRequests {
		s.RequestCallback = recordRequestsCallback(t, includeHeaders, outputDir)
	}

	// Log API responses if the -logrequests flag is set.
	if logRequests {
		s.ResponseCallback = logResponseCallback(t)
	}

	for ind := range stubs {
		// We want later stubs takes precedence, because then leaf configs take precedence over parent directory configs
		// In gorilla/mux earlier handlers take precedence, so we need to reverse the order
		stub := stubs[len(stubs)-1-ind]
		require.NotEmpty(t, stub.Pattern)
		items := strings.Split(stub.Pattern, " ")
		require.Len(t, items, 2)
		s.Handle(items[0], items[1], func(req testserver.Request) any {
			if stub.Delay > 0 {
				ctx := req.Context

				timer := time.NewTimer(stub.Delay)
				defer timer.Stop()

				select {
				case <-timer.C:
					break
				case <-ctx.Done():
					// Client canceled/connection closed; just exit.
					// Optional: log the reason (context deadline, cancellation, etc.)
					t.Logf("request canceled: %v", ctx.Err())
					return nil
				}
			}

			return stub.Response
		})
	}

	// The earliest handlers take precedence, add default handlers last
	testserver.AddDefaultHandlers(s)
	return s.URL
}

func startProxyServer(t *testing.T,
	logRequests bool,
	includeHeaders []string,
	outputDir string,
) string {
	s := testproxy.New(t)

	// Always record requests for a proxy server.
	s.RequestCallback = recordRequestsCallback(t, includeHeaders, outputDir)

	// Log API responses if the -logrequests flag is set.
	if logRequests {
		s.ResponseCallback = logResponseCallback(t)
	}

	return s.URL
}

type LoggedRequest struct {
	Headers http.Header            `json:"headers,omitempty"`
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	Q       map[string]interface{} `json:"q,omitempty"`
	Body    any                    `json:"body,omitempty"`
	RawBody string                 `json:"raw_body,omitempty"`
}

func getLoggedRequest(req *testserver.Request, includedHeaders []string) LoggedRequest {
	result := LoggedRequest{
		Method:  req.Method,
		Path:    req.URL.Path,
		Headers: filterHeaders(req.Headers, includedHeaders),
	}

	if req.URL.RawQuery != "" {
		queryParams := req.URL.Query()
		result.Q = make(map[string]interface{})
		for key, values := range queryParams {
			if len(values) == 1 {
				result.Q[key] = values[0]
			} else {
				result.Q[key] = values
			}
		}
	}

	if json.Valid(req.Body) {
		result.Body = json.RawMessage(req.Body)
	} else {
		result.RawBody = string(req.Body)
	}

	return result
}

func filterHeaders(h http.Header, includedHeaders []string) http.Header {
	headers := make(http.Header)
	for k, v := range h {
		if !slices.Contains(includedHeaders, k) {
			continue
		}
		headers[k] = v
	}
	return headers
}

func formatHeadersAndBody(prefix string, headers http.Header, body []byte) string {
	var result []string
	for key, values := range headers {
		if len(values) == 1 {
			result = append(result, fmt.Sprintf("%s%s: %s", prefix, key, values[0]))
		} else {
			result = append(result, fmt.Sprintf("%s%s: %s", prefix, key, values))
		}
	}
	if len(body) > 0 {
		var s string
		if utf8.Valid(body) {
			s = string(body)
		} else {
			s = fmt.Sprintf("[Binary %d bytes]", len(body))
		}
		s = strings.ReplaceAll(s, "\n", "\n"+prefix)
		result = append(result, prefix+s)
	}
	return strings.Join(result, "\n")
}
