//go:build unix

// The test interrupts the shim by sending a signal to its own process, which os/signal cannot
// deliver to self on Windows. The shim itself only runs on the Linux driver.
package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunAgentShimSignalReportsUserAborted interrupts a launch mid-setup and checks the
// failure event still reaches the telemetry upload as USER_ABORTED. Without a signal handler
// in RunAgentShim the signal would terminate the test process instead.
func TestRunAgentShimSignalReportsUserAborted(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			var body telemetry.RequestBody
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasPrefix(r.URL.Path, "/api/2.1/unity-catalog/model-services"):
					// The gateway probe is in flight, so RunAgentShim's handler is installed:
					// interrupt it and hold the request until the cancellation aborts it.
					assert.NoError(t, syscall.Kill(os.Getpid(), sig))
					<-r.Context().Done()
				case r.URL.Path == "/telemetry-ext":
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					assert.NoError(t, json.Unmarshal(b, &body))
					_ = json.NewEncoder(w).Encode(telemetry.ResponseBody{NumProtoSuccess: 1})
				default:
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte("{}"))
				}
			}))
			defer srv.Close()

			ctx := telemetry.WithNewLogger(t.Context())
			ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: srv.URL, Token: "test-token"})

			err := RunAgentShim(ctx, newProbeClient(t, srv.URL), "claude", nil)
			require.Error(t, err)

			// cmd/root uploads on the command's own context, which the signal leaves intact.
			require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))
			require.Len(t, body.ProtoLogs, 1)
			var logged protos.FrontendLog
			require.NoError(t, json.Unmarshal([]byte(body.ProtoLogs[0]), &logged))
			event := logged.Entry.DatabricksCliLog.SshAgentShimEvent
			require.NotNil(t, event)
			assert.False(t, event.IsSuccess)
			assert.Equal(t, protos.SshAgentShimErrorCategoryUserAborted, event.ErrorCategory)
		})
	}
}
