//go:build unix

// The tests interrupt the shim by sending a signal to their own process, which os/signal cannot
// deliver to self on Windows. The shim itself only runs on the Linux driver.
package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAgentShimServer serves the gateway probe with modelServices and records the telemetry
// upload into body.
func newAgentShimServer(t *testing.T, modelServices http.HandlerFunc, body *telemetry.RequestBody) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/2.1/unity-catalog/model-services"):
			modelServices(w, r)
		case r.URL.Path == "/telemetry-ext":
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(b, body))
			_ = json.NewEncoder(w).Encode(telemetry.ResponseBody{NumProtoSuccess: 1})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("{}"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newAgentShimContext returns a context with the telemetry and auth setup cmd/root provides,
// and an isolated home so the shim never touches the real one.
func newAgentShimContext(t *testing.T, host string) context.Context {
	ctx := cmdctx.GenerateExecId(t.Context())
	ctx = dbr.MockRuntime(ctx, dbr.Environment{})
	ctx = telemetry.WithNewLogger(ctx)
	ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: host, Token: "test-token"})
	return env.WithUserHomeDir(ctx, t.TempDir())
}

// assertUserAbortedEvent uploads the buffered telemetry as cmd/root does, on the command's own
// context (which the signal leaves intact), and checks it holds one USER_ABORTED failure.
func assertUserAbortedEvent(t *testing.T, ctx context.Context, body *telemetry.RequestBody) {
	t.Helper()
	require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))
	require.Len(t, body.ProtoLogs, 1)
	var logged protos.FrontendLog
	require.NoError(t, json.Unmarshal([]byte(body.ProtoLogs[0]), &logged))
	event := logged.Entry.DatabricksCliLog.SshAgentShimEvent
	require.NotNil(t, event)
	assert.False(t, event.IsSuccess)
	assert.Equal(t, protos.SshAgentShimErrorCategoryUserAborted, event.ErrorCategory)
}

// TestRunAgentShimSignalReportsUserAborted interrupts a launch mid-setup and checks the
// failure event still reaches the telemetry upload as USER_ABORTED. Without a signal handler
// in RunAgentShim the signal would terminate the test process instead.
func TestRunAgentShimSignalReportsUserAborted(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			var body telemetry.RequestBody
			srv := newAgentShimServer(t, func(w http.ResponseWriter, r *http.Request) {
				// The gateway probe is in flight, so RunAgentShim's handler is installed:
				// interrupt it and hold the request until the cancellation aborts it.
				assert.NoError(t, syscall.Kill(os.Getpid(), sig))
				<-r.Context().Done()
			}, &body)
			ctx := newAgentShimContext(t, srv.URL)

			err := RunAgentShim(ctx, newProbeClient(t, srv.URL), "claude", nil)
			require.ErrorIs(t, err, context.Canceled)
			assertUserAbortedEvent(t, ctx, &body)
		})
	}
}

// TestRunAgentShimSignalDuringIgnoredStepSkipsLaunch interrupts the launch during a step that
// swallows the cancellation: the warm-path `npm config set` run by bootstrapToolchain. The
// probe has already succeeded, so only the check before the exec can catch the signal, and
// the agent must not launch.
func TestRunAgentShimSignalDuringIgnoredStepSkipsLaunch(t *testing.T) {
	const modelServicesBody = `{"model_services":[{"name":"model-services/system.ai.gpt-5"}]}`

	var body telemetry.RequestBody
	srv := newAgentShimServer(t, jsonHandler(http.StatusOK, modelServicesBody), &body)
	ctx := newAgentShimContext(t, srv.URL)

	// Stub the toolchain so bootstrap takes the warm path. npm interrupts the shim and waits to
	// be killed by the cancellation, so ctx is cancelled by the time bootstrap returns; the
	// sleep is bounded so a regression there can't hang the test. ucode stands in for the
	// agent: if the shim execs it, it leaves the marker and fails the run. PATH holds only the
	// stubs, so they use shell builtins and an absolute path to sleep.
	sleep, err := exec.LookPath("sleep")
	require.NoError(t, err)
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "launched")
	writeFakeExec(t, binDir, "npm", "kill -INT $PPID\nexec \""+sleep+"\" 30\n")
	writeFakeExec(t, binDir, "ucode", ": > \""+marker+"\"\nexit 1\n")
	t.Setenv("PATH", binDir)

	err = RunAgentShim(ctx, newProbeClient(t, srv.URL), "claude", nil)
	require.ErrorIs(t, err, context.Canceled)
	assert.NoFileExists(t, marker, "the agent must not launch after a signal")
	assertUserAbortedEvent(t, ctx, &body)
}
