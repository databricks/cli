package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func StartDefaultServer(t *testing.T) {
	s := testserver.New(t)
	addDefaultHandlers(s)

	t.Setenv("DATABRICKS_DEFAULT_HOST", s.URL)

	// Do not read user's ~/.databrickscfg
	homeDir := t.TempDir()
	t.Setenv(env.HomeEnvVar(), homeDir)
}

func isTruePtr(value *bool) bool {
	return value != nil && *value
}

func PrepareServerAndClient(t *testing.T, config TestConfig, logRequests bool, outputDir string) *databricks.WorkspaceClient {
	cloudEnv := os.Getenv("CLOUD_ENV")

	// If we are running on a cloud environment, use the host configured in the
	// environment.
	if cloudEnv != "" {
		w, err := databricks.NewWorkspaceClient(&databricks.Config{})
		require.NoError(t, err)

		return w
	}

	recordRequests := isTruePtr(config.RecordRequests)

	tokenSuffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	token := "dbapi" + tokenSuffix

	// If we are not recording requests, and no custom server server stubs are configured,
	// use the default shared server.
	if len(config.Server) == 0 && !recordRequests {
		w, err := databricks.NewWorkspaceClient(&databricks.Config{
			Host:  os.Getenv("DATABRICKS_DEFAULT_HOST"),
			Token: token,
		})
		require.NoError(t, err)

		return w
	}

	host := startDedicatedServer(t, config.Server, recordRequests, logRequests, config.IncludeRequestHeaders, outputDir)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  host,
		Token: token,
	})
	require.NoError(t, err)
	return w
}

func startDedicatedServer(t *testing.T,
	stubs []ServerStub,
	recordRequests bool,
	logRequests bool,
	includeHeaders []string,
	outputDir string,
) string {
	s := testserver.New(t)

	if recordRequests {
		requestsPath := filepath.Join(outputDir, "out.requests.txt")
		s.RequestCallback = func(request *testserver.Request) {
			req := getLoggedRequest(request, includeHeaders)
			reqJson, err := json.MarshalIndent(req, "", "  ")
			assert.NoErrorf(t, err, "Failed to json-encode: %#v", req)

			f, err := os.OpenFile(requestsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			assert.NoError(t, err)
			defer f.Close()

			_, err = f.WriteString(string(reqJson) + "\n")
			assert.NoError(t, err)
		}
	}

	if logRequests {
		s.ResponseCallback = func(request *testserver.Request, response *testserver.EncodedResponse) {
			t.Logf("%d %s %s\n%s\n%s",
				response.StatusCode, request.Method, request.URL,
				formatHeadersAndBody("> ", request.Headers, request.Body),
				formatHeadersAndBody("# ", response.Headers, response.Body),
			)
		}
	}

	for ind := range stubs {
		// We want later stubs takes precedence, because then leaf configs take precedence over parent directory configs
		// In gorilla/mux earlier handlers take precedence, so we need to reverse the order
		stub := stubs[len(stubs)-1-ind]
		require.NotEmpty(t, stub.Pattern)
		items := strings.Split(stub.Pattern, " ")
		require.Len(t, items, 2)
		s.Handle(items[0], items[1], func(req testserver.Request) any {
			time.Sleep(stub.Delay)
			return stub.Response
		})
	}

	// The earliest handlers take precedence, add default handlers last
	addDefaultHandlers(s)

	return s.URL
}

type LoggedRequest struct {
	Headers http.Header `json:"headers,omitempty"`
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Body    any         `json:"body,omitempty"`
	RawBody string      `json:"raw_body,omitempty"`
}

func getLoggedRequest(req *testserver.Request, includedHeaders []string) LoggedRequest {
	result := LoggedRequest{
		Method:  req.Method,
		Path:    req.URL.Path,
		Headers: filterHeaders(req.Headers, includedHeaders),
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
