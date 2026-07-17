package aircmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsCommandShape(t *testing.T) {
	cmd := newLogsCommand()
	assert.Equal(t, "logs JOB_RUN_ID", cmd.Use)
	assert.Empty(t, cmd.Commands(), "logs must not register subcommands")
	assert.NoError(t, cmd.Args(cmd, []string{"123"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"1", "2"}))

	// --review stays hidden to match the Python CLI.
	review := cmd.Flags().Lookup("review")
	require.NotNil(t, review)
	assert.True(t, review.Hidden)
}

// runLogsCmd invokes the logs command's RunE with the given flags against a mock
// (no-HTTP) workspace client. Used for input validation that fails before any
// API call.
func runLogsCmd(t *testing.T, args []string, flagsToSet map[string]string) error {
	t.Helper()
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := withOutput(newLogsCommand(), flags.OutputText)
	for k, v := range flagsToSet {
		require.NoError(t, cmd.Flags().Set(k, v))
	}
	cmd.SetContext(ctx)
	return cmd.RunE(cmd, args)
}

func TestLogsFlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		flags   map[string]string
		wantMsg string
	}{
		{
			name:    "lines and minutes are mutually exclusive",
			args:    []string{"5"},
			flags:   map[string]string{"lines": "100", "minutes": "10"},
			wantMsg: "cannot combine --lines with --minutes",
		},
		{
			name:    "negative lines rejected",
			args:    []string{"5"},
			flags:   map[string]string{"lines": "-1"},
			wantMsg: "invalid --lines",
		},
		{
			name:    "negative minutes rejected",
			args:    []string{"5"},
			flags:   map[string]string{"minutes": "-1"},
			wantMsg: "invalid --minutes",
		},
		{
			name:    "download-to not implemented",
			args:    []string{"5"},
			flags:   map[string]string{"download-to": "/tmp/logs"},
			wantMsg: "--download-to is not implemented yet",
		},
		{
			name:    "review not implemented",
			args:    []string{"5"},
			flags:   map[string]string{"review": "true"},
			wantMsg: "--review is not implemented yet",
		},
		{
			name:    "invalid run id",
			args:    []string{"abc"},
			wantMsg: "invalid JOB_RUN_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runLogsCmd(t, tt.args, tt.flags)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

// completedRunLogsServer serves the auth probe, a terminal runs/get, and a
// single page of Bricklens logs (newest-first, as the tail fetch requests).
func completedRunLogsServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{
				"run_id": 5,
				"start_time": 1000,
				"end_time": 2000,
				"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
				"tasks": [{"attempt_number": 0}]
			}`))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			// Newest-first, as drainTail requests; reversed to oldest-first on print.
			_, _ = w.Write([]byte(`{"log_records": [
				{"time_unix_nano": 2000000000, "body": "line two", "node_index": 0},
				{"time_unix_nano": 1000000000, "body": "line one", "node_index": 0}
			]}`))
		default:
			// Me() probe and SDK config discovery.
			_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLogsCompletedRunTail(t *testing.T) {
	srv := completedRunLogsServer(t)
	var buf bytes.Buffer
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(&cobra.Command{}, flags.OutputText)
	cmd.SetContext(ctx)
	cmd.SetOut(&buf)

	// Drive runLogs directly (bypassing PreRunE auth wiring) with a resolved request.
	err := runLogs(ctx, cmd, logRequest{runID: 5, node: 0, attempt: -1})
	require.NoError(t, err)

	// Records print oldest-first regardless of the newest-first fetch order.
	assert.Equal(t, "line one\nline two\n", buf.String())
}

// featureDisabledLogsServer serves a terminal runs/get and a Bricklens endpoint
// gated off by the backend SAFE flag (FEATURE_DISABLED), so the command must
// fall back to the MLflow path.
func featureDisabledLogsServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{
				"run_id": 5,
				"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
				"tasks": [{"attempt_number": 0}]
			}`))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/ai-training/workflows/by-run-id/"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code": "FEATURE_DISABLED", "message": "bricklens logs gated off"}`))
		default:
			_, _ = w.Write([]byte(`{"userName":"u@example.com"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLogsFallsBackToMLflow(t *testing.T) {
	srv := featureDisabledLogsServer(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), newTestWorkspaceClient(t, srv.URL))
	cmd := withOutput(&cobra.Command{}, flags.OutputText)
	cmd.SetContext(ctx)

	// The Bricklens endpoint is gated off, so fetchLogs routes to the MLflow
	// fallback. Until that path is implemented (air-logs-m3) it reports the
	// explicit not-yet-available error, which proves the try/catch wiring fires.
	err := runLogs(ctx, cmd, logRequest{runID: 5, node: 0, attempt: -1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MLflow fallback is not yet implemented")
}
