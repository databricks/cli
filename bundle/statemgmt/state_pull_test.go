package statemgmt_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	legacyDMSState = `{"state_version":2,"lineage":"old-dms","features":{"deployment_history":{}},"state":{}}`
	ordinaryState  = `{"state_version":2,"lineage":"ordinary","serial":9,"state":{}}`
	stateWALSuffix = ".wal"
)

type statePullTransport struct {
	state  string
	status int
	reads  int
}

func (s *statePullTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	status := http.StatusNotFound
	body := `{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`
	if strings.HasSuffix(r.URL.Query().Get("path"), "/resources.json") {
		if s.status != 0 {
			status = s.status
			body = `{"error_code":"PERMISSION_DENIED","message":"remote state denied"}`
		} else if s.state != "" {
			status = http.StatusOK
			body = `{"object_type":"FILE","path":"/Workspace/test/state/resources.json"}`
			if r.URL.Path == "/api/2.0/workspace/export" {
				body = s.state
				s.reads++
			}
		}
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func TestPullResourcesStateAfterDeploymentHistory(t *testing.T) {
	for _, tt := range []struct {
		name       string
		local      string
		remote     string
		status     int
		wal        bool
		wantErr    string
		wantAbsent bool
	}{
		{name: "destroyed deployment discards legacy cache", local: legacyDMSState, wantAbsent: true},
		{name: "ordinary remote state replaces legacy cache", local: legacyDMSState, remote: ordinaryState},
		{name: "live DMS deployment remains protected", local: legacyDMSState, remote: legacyDMSState, wantErr: "unsetting experimental.deployment_history"},
		{name: "higher serial local state cannot override live DMS", local: ordinaryState, remote: legacyDMSState, wantErr: "unsetting experimental.deployment_history"},
		{name: "remote access error preserves cache", local: legacyDMSState, status: http.StatusForbidden, wantErr: "remote state denied"},
		{name: "invalid remote state preserves cache", local: legacyDMSState, remote: "{", wantErr: "parsing state"},
		{name: "invalid local state preserves cache", local: `{"serial":"invalid","features":{"deployment_history":{}}}`, wantErr: "parsing"},
		{name: "legacy marker and WAL require reconciliation", local: legacyDMSState, wal: true, wantErr: "while its WAL exists"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logdiag.InitContext(t.Context())
			logdiag.SetCollect(ctx, true)
			b := &bundle.Bundle{BundleRootPath: t.TempDir()}
			b.Config.Bundle.Target = "default"
			b.Config.Workspace.StatePath = "/Workspace/test/state"
			transport := &statePullTransport{state: tt.remote, status: tt.status}
			w := databricks.Must(databricks.NewWorkspaceClient(&databricks.Config{
				Host: "https://workspace.test", Token: "token", HTTPTransport: transport,
			}))
			b.SetWorkpaceClient(w)
			_, localPath := b.StateFilenameDirect(ctx)
			require.NoError(t, os.MkdirAll(filepath.Dir(localPath), 0o700))
			require.NoError(t, os.WriteFile(localPath, []byte(tt.local), 0o600))
			const wal = "{\"state_version\":2,\"lineage\":\"stale\",\"serial\":1}\n{\"k\":\"resources.jobs.old\",\"v\":{\"__id__\":\"stale-id\"}}\n"
			if tt.wal {
				require.NoError(t, os.WriteFile(localPath+stateWALSuffix, []byte(wal), 0o600))
			}

			// A high-serial ordinary cache is valid for read-only commands; deploy always pulls.
			alwaysPull := statemgmt.AlwaysPull(tt.local == ordinaryState)
			_, winner := statemgmt.PullResourcesState(ctx, b, alwaysPull, engine.EngineSetting{Type: engine.EngineDirect})
			if tt.wantErr != "" {
				assert.True(t, logdiag.HasError(ctx))
				assert.Contains(t, logdiag.GetFirstErrorSummary(ctx), tt.wantErr)
				local, err := os.ReadFile(localPath)
				require.NoError(t, err)
				assert.Equal(t, tt.local, string(local))
			} else {
				require.False(t, logdiag.HasError(ctx), logdiag.GetFirstErrorSummary(ctx))
				require.NotNil(t, winner)
				if tt.wantAbsent {
					assert.NoFileExists(t, localPath)
				} else {
					assert.Positive(t, transport.reads)
					assert.False(t, winner.IsLocal)
					local, err := os.ReadFile(localPath)
					require.NoError(t, err)
					assert.Equal(t, tt.remote, string(local))
				}
			}
			if tt.wal {
				contents, err := os.ReadFile(localPath + stateWALSuffix)
				require.NoError(t, err)
				assert.Equal(t, wal, string(contents))
			}
		})
	}
}
