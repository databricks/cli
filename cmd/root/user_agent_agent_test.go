package root

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// All known agent env vars. Must be unset in tests to avoid interference
// from the host environment (e.g., running tests inside Claude Code).
var agentEnvVars = []string{
	"ANTIGRAVITY_AGENT",
	"CLAUDECODE",
	"CLINE_ACTIVE",
	"CODEX_CI",
	"COPILOT_CLI",
	"CURSOR_AGENT",
	"GEMINI_CLI",
	"OPENCLAW_SHELL",
	"OPENCODE",
}

// unsetAgentEnv removes all known agent env vars from the environment.
// The SDK uses os.LookupEnv, so setting to empty is not enough; the vars
// must be fully unset.
func unsetAgentEnv(t *testing.T) {
	t.Helper()
	for _, v := range agentEnvVars {
		original, exists := os.LookupEnv(v)
		os.Unsetenv(v)
		if exists {
			t.Cleanup(func() { os.Setenv(v, original) })
		}
	}
}

// captureUserAgent makes an HTTP request through the SDK and returns the
// captured User-Agent header string.
func captureUserAgent(t *testing.T) string {
	t.Helper()

	var capturedUA string
	cfg := &config.Config{
		Host:  "https://test.databricks.com",
		Token: "test-token",
		HTTPTransport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			capturedUA = r.Header.Get("User-Agent")
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		}),
	}

	clientCfg, err := config.HTTPClientConfigFromConfig(cfg)
	require.NoError(t, err)
	client := httpclient.NewApiClient(clientCfg)

	_ = client.Do(t.Context(), "GET", "/api/2.0/clusters/list")
	return capturedUA
}

// TestSDKAgentDetection verifies the SDK adds agent/<name> to the User-Agent
// header when exactly one agent env var is set.
func TestSDKAgentDetection(t *testing.T) {
	unsetAgentEnv(t)
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)

	t.Setenv("CLAUDECODE", "1")

	ua := captureUserAgent(t)
	assert.Contains(t, ua, "agent/claude-code")
	assert.Equal(t, 1, strings.Count(ua, "agent/"), "expected exactly one agent/ segment")
}

// TestSDKNoAgentDetected verifies no agent/ segment is added when no agent
// env vars are set.
func TestSDKNoAgentDetected(t *testing.T) {
	unsetAgentEnv(t)
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)

	ua := captureUserAgent(t)
	assert.NotContains(t, ua, "agent/")
}

// TestSDKMultipleAgentsSuppressed verifies no agent/ segment is added when
// multiple agent env vars are set (ambiguity guard).
func TestSDKMultipleAgentsSuppressed(t *testing.T) {
	unsetAgentEnv(t)
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)

	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CURSOR_AGENT", "1")

	ua := captureUserAgent(t)
	assert.NotContains(t, ua, "agent/")
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
