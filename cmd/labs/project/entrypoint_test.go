package project

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureRunningClusterNotAnAgentDropsAgentHeader checks that the labs entrypoint's cluster client leaves the agent
// token out of User-Agent under DATABRICKS_NOT_AN_AGENT, and keeps it without the variable.
func TestEnsureRunningClusterNotAnAgentDropsAgentHeader(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)

	cases := []struct {
		name    string
		noAgent bool
	}{
		{name: "variable set", noAgent: true},
		{name: "variable unset", noAgent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var userAgent string
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				// The SDK first asks for the host's well-known config; only the cluster call is checked.
				if !strings.HasSuffix(r.URL.Path, "/clusters/get") {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				userAgent = r.Header.Get("User-Agent")
				rw.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(rw, `{"cluster_id":"c1","state":"RUNNING"}`)
			}))
			t.Cleanup(server.Close)

			ctx := cmdio.MockDiscard(t.Context())
			if tc.noAgent {
				t.Setenv("DATABRICKS_NOT_AN_AGENT", "1")
			}
			cfg := &config.Config{Host: server.URL, Token: "dapi-test", ClusterID: "c1"}
			require.NoError(t, (&Entrypoint{}).ensureRunningCluster(ctx, cfg))

			if tc.noAgent {
				assert.NotContains(t, userAgent, "agent/")
			} else {
				assert.Contains(t, userAgent, "agent/claude-code")
			}
		})
	}
}
