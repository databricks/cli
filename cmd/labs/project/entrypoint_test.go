package project

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clusterTransport answers the cluster call in place of a server, so the test needs no listener. It records the
// User-Agent of the cluster call.
type clusterTransport struct {
	userAgent string
}

func (c *clusterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The SDK first asks for the host's well-known config; only the cluster call is checked.
	status, body := http.StatusNotFound, ""
	if strings.HasSuffix(req.URL.Path, "/clusters/get") {
		c.userAgent = req.Header.Get("User-Agent")
		status, body = http.StatusOK, `{"cluster_id":"c1","state":"RUNNING"}`
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

// TestEnsureRunningClusterNotAnAgentDropsAgentHeader checks that the labs entrypoint's cluster client leaves the agent
// token out of User-Agent under DATABRICKS_NOT_AN_AGENT, and keeps it without the variable.
func TestEnsureRunningClusterNotAnAgentDropsAgentHeader(t *testing.T) {
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")

	cases := []struct {
		name    string
		noAgent bool
	}{
		{name: "variable set", noAgent: true},
		{name: "variable unset", noAgent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transport := &clusterTransport{}
			ctx := cmdio.MockDiscard(t.Context())
			if tc.noAgent {
				t.Setenv("DATABRICKS_NOT_AN_AGENT", "1")
			}
			cfg := &config.Config{Host: "https://workspace.test", Token: "dapi-test", ClusterID: "c1", HTTPTransport: transport}
			require.NoError(t, (&Entrypoint{}).ensureRunningCluster(ctx, cfg))

			if tc.noAgent {
				assert.NotContains(t, transport.userAgent, "agent/")
			} else {
				assert.Contains(t, transport.userAgent, "agent/claude-code")
			}
		})
	}
}
