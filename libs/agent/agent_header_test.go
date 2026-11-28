package agent

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithoutAgentToken(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		dropped bool
	}{
		{in: "cli/1.0 databricks-sdk-go/0.204.0 go/1.25 os/linux agent/claude-code auth/pat", want: "cli/1.0 databricks-sdk-go/0.204.0 go/1.25 os/linux auth/pat", dropped: true},
		{in: "cli/1.0 agent/multiple", want: "cli/1.0", dropped: true},
		{in: "cli/1.0 auth/pat", want: "cli/1.0 auth/pat"},
		{in: "", want: ""},
	}
	for _, tc := range cases {
		got, dropped := withoutAgentToken(tc.in)
		assert.Equal(t, tc.want, got)
		assert.Equal(t, tc.dropped, dropped)
	}
}

// recordingTransport is the base transport of a test; it keeps the User-Agent of each request it gets.
type recordingTransport struct {
	userAgents []string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.userAgents = append(r.userAgents, req.Header.Get("User-Agent"))
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
}

func TestNoAgentHeaderTransportDropsAgentAndKeepsRequest(t *testing.T) {
	base := &recordingTransport{}
	cfg := &config.Config{HTTPTransport: base}
	ctx := env.Set(t.Context(), "DATABRICKS_NOT_AN_AGENT", "1")
	ApplyNotAnAgentHeader(ctx, cfg)

	req, err := http.NewRequest(http.MethodGet, "https://workspace.test/api", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "cli/1.0 agent/claude-code auth/pat")
	resp, err := cfg.HTTPTransport.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.Equal(t, []string{"cli/1.0 auth/pat"}, base.userAgents)
	// The caller's request keeps its header.
	assert.Equal(t, "cli/1.0 agent/claude-code auth/pat", req.Header.Get("User-Agent"))
}

// skippingTransport is a fixture transport that asks the SDK not to retry on I/O errors.
type skippingTransport struct {
	recordingTransport
}

func (*skippingTransport) SkipRetryOnIO() bool { return true }

func TestNoAgentHeaderTransportForwardsSkipRetryOnIO(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_NOT_AN_AGENT", "1")
	cases := []struct {
		name string
		base http.RoundTripper
		want bool
	}{
		{name: "fixture", base: &skippingTransport{}, want: true},
		{name: "plain", base: &recordingTransport{}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{HTTPTransport: tc.base}
			ApplyNotAnAgentHeader(ctx, cfg)
			skippable, ok := cfg.HTTPTransport.(interface{ SkipRetryOnIO() bool })
			require.True(t, ok)
			assert.Equal(t, tc.want, skippable.SkipRetryOnIO())
		})
	}
}

func TestApplyNotAnAgentHeaderUnsetLeavesConfig(t *testing.T) {
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")
	base := &recordingTransport{}
	cfg := &config.Config{HTTPTransport: base}
	ApplyNotAnAgentHeader(t.Context(), cfg)
	assert.Same(t, base, cfg.HTTPTransport)
}

// meTransport stands in for the workspace: it answers the SDK's requests without a listener, which the
// Windows test runners refuse, and keeps the User-Agent of each Me call.
type meTransport struct {
	userAgents chan string
}

func (m *meTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}
	// The SDK first asks for the host's well-known config; only the Me call is checked.
	if req.URL.Path == "/api/2.0/preview/scim/v2/Me" {
		m.userAgents <- req.Header.Get("User-Agent")
		resp.StatusCode = http.StatusOK
		resp.Header.Set("Content-Type", "application/json")
		resp.Body = io.NopCloser(strings.NewReader(`{"id":"1","userName":"user@example.com"}`))
	}
	return resp, nil
}

func TestNotAnAgentHeaderOnSDKRequests(t *testing.T) {
	cases := []struct {
		name    string
		noAgent bool
	}{
		{name: "variable set", noAgent: true},
		{name: "variable unset", noAgent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.SetAgentEnv(t, "CLAUDECODE", "1")
			transport := &meTransport{userAgents: make(chan string, 1)}

			ctx := t.Context()
			if tc.noAgent {
				ctx = env.Set(ctx, "DATABRICKS_NOT_AN_AGENT", "1")
			}
			cfg := &config.Config{Host: "https://workspace.test", Token: "token", HTTPTransport: transport}
			ApplyNotAnAgentHeader(ctx, cfg)
			w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
			require.NoError(t, err)
			_, err = w.CurrentUser.Me(ctx, iam.MeRequest{})
			require.NoError(t, err)

			ua := <-transport.userAgents
			if tc.noAgent {
				assert.NotContains(t, ua, "agent/")
			} else {
				assert.Contains(t, ua, "agent/claude-code")
			}
		})
	}
}
