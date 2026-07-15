package aircmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// runCancelAll runs `cancel --all` against w with the given output mode and
// stdin, capturing output into buf.
func runCancelAll(t *testing.T, w *databricks.WorkspaceClient, out flags.Output, in io.Reader, buf *bytes.Buffer) error {
	t.Helper()
	cmd := withOutput(newCancelCommand(), out)
	require.NoError(t, cmd.Flags().Set("all", "true"))
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), out, in, buf, buf, "", ""))
	cmd.SetContext(cmdctx.SetWorkspaceClient(ctx, w))
	return cmd.RunE(cmd, nil)
}

// cancelEnvelope decodes the air JSON envelope with the cancel payload.
type cancelEnvelope struct {
	V    int        `json:"v"`
	Data cancelData `json:"data"`
}

// runCancel runs the cancel command against w with the given output mode and
// stdin, capturing stdout/stderr into buf.
func runCancel(t *testing.T, w *databricks.WorkspaceClient, out flags.Output, in string, buf *bytes.Buffer, args ...string) (*cobra.Command, error) {
	t.Helper()
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), out, strings.NewReader(in), buf, buf, "", ""))
	ctx = cmdctx.SetWorkspaceClient(ctx, w)
	cmd := withOutput(newCancelCommand(), out)
	cmd.SetContext(ctx)
	return cmd, cmd.RunE(cmd, args)
}

func TestCancelArgs(t *testing.T) {
	tests := []struct {
		name    string
		all     bool
		args    []string
		wantErr string
	}{
		{name: "one id", args: []string{"123"}},
		{name: "many ids", args: []string{"123", "456"}},
		{name: "all", all: true},
		{name: "no input", wantErr: "provide at least one JOB_RUN_ID, or use --all"},
		{name: "ids with all", all: true, args: []string{"123"}, wantErr: "cannot combine JOB_RUN_ID arguments with --all"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCancelCommand()
			if tc.all {
				require.NoError(t, cmd.Flags().Set("all", "true"))
			}
			err := cmd.Args(cmd, tc.args)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestCancelRunInvalidID(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	for _, id := range []string{"abc", "0", "-1"} {
		err := cancelRun(t.Context(), m.WorkspaceClient, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid run ID")
	}
}

func TestCancelByIDSuccess(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 123}).Return(nil, nil)
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 456}).Return(nil, nil)

	var buf bytes.Buffer
	_, err := runCancel(t, m.WorkspaceClient, flags.OutputText, "", &buf, "123", "456")
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Successfully requested cancellation for run 123")
	assert.Contains(t, out, "Successfully requested cancellation for run 456")
	// More than one run cancelled prints the count summary.
	assert.Contains(t, out, "Successfully requested cancellation for 2 run(s).")
}

func TestCancelByIDNotFound(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 5}).Return(nil, apierr.ErrResourceDoesNotExist)

	var buf bytes.Buffer
	_, err := runCancel(t, m.WorkspaceClient, flags.OutputText, "", &buf, "5")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	out := buf.String()
	assert.Contains(t, out, "Run 5 not found")
	assert.Contains(t, out, "1 run(s) failed to cancel.")
}

func TestCancelByIDNotFoundInvalidParam(t *testing.T) {
	// The cancel endpoint reports an unknown run as 400 INVALID_PARAMETER_VALUE,
	// which the SDK does not remap to ErrResourceDoesNotExist for this path.
	m := mocks.NewMockWorkspaceClient(t)
	apiErr := &apierr.APIError{StatusCode: http.StatusBadRequest, ErrorCode: "INVALID_PARAMETER_VALUE", Message: "Run 5 does not exist."}
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 5}).Return(nil, apiErr)

	var buf bytes.Buffer
	_, err := runCancel(t, m.WorkspaceClient, flags.OutputText, "", &buf, "5")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Contains(t, buf.String(), "Run 5 not found")
}

func TestCancelPartialFailureJSON(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 123}).Return(nil, nil)
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 5}).Return(nil, apierr.ErrResourceDoesNotExist)

	var buf bytes.Buffer
	_, err := runCancel(t, m.WorkspaceClient, flags.OutputJSON, "", &buf, "123", "5")
	// The envelope is printed, but a failure still exits non-zero.
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	var got cancelEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, []string{"123"}, got.Data.Cancelled)
	require.Len(t, got.Data.Failed, 1)
	assert.Equal(t, "5", got.Data.Failed[0].RunID)
	assert.False(t, got.Data.All)
}

func TestCancelByIDSuccessJSON(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 123}).Return(nil, nil)

	var buf bytes.Buffer
	_, err := runCancel(t, m.WorkspaceClient, flags.OutputJSON, "", &buf, "123")
	require.NoError(t, err)

	var got cancelEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, []string{"123"}, got.Data.Cancelled)
	assert.Empty(t, got.Data.Failed)
}

func TestCancelByIDGenericFailure(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().CancelRun(mock.Anything, jobs.CancelRun{RunId: 7}).Return(nil, errors.New("boom"))

	var buf bytes.Buffer
	_, err := runCancel(t, m.WorkspaceClient, flags.OutputText, "", &buf, "7")
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Contains(t, buf.String(), "Failed to cancel run 7: boom")
}

func TestCancelAllNoActiveRuns(t *testing.T) {
	w := newTestWorkspaceClient(t, runsServer(t, runsListBody(t, "")).URL)
	var buf bytes.Buffer
	require.NoError(t, runCancelAll(t, w, flags.OutputText, nil, &buf))
	assert.Contains(t, buf.String(), "No active runs found.")
}

func TestCancelAllNoActiveRunsJSON(t *testing.T) {
	srv := runsServer(t, runsListBody(t, ""))
	w := newTestWorkspaceClient(t, srv.URL)

	var buf bytes.Buffer
	require.NoError(t, runCancelAll(t, w, flags.OutputJSON, nil, &buf))

	var got cancelEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Empty(t, got.Data.Cancelled)
	assert.True(t, got.Data.All)
	assert.Equal(t, srv.URL, got.Data.Workspace)
}

func TestCancelAllConfirmYes(t *testing.T) {
	srv := runsServer(t, runsListBody(t, "",
		airBaseRun(111, "me@example.com", "GPU_1xA10", 1, "/Users/me@example.com/exp-a"),
		airBaseRun(222, "me@example.com", "GPU_1xA10", 1, "/Users/me@example.com/exp-b"),
	))
	w := newTestWorkspaceClient(t, srv.URL)

	var buf bytes.Buffer
	require.NoError(t, runCancelAll(t, w, flags.OutputText, strings.NewReader("y\n"), &buf))
	out := buf.String()
	assert.Contains(t, out, "active run(s) to cancel")
	assert.Contains(t, out, "Successfully requested cancellation for run 111")
	assert.Contains(t, out, "Successfully requested cancellation for run 222")
}

func TestCancelAllAbort(t *testing.T) {
	srv := runsServer(t, runsListBody(t, "",
		airBaseRun(111, "me@example.com", "GPU_1xA10", 1, "/Users/me@example.com/exp-a"),
	))
	w := newTestWorkspaceClient(t, srv.URL)

	var buf bytes.Buffer
	err := runCancelAll(t, w, flags.OutputText, strings.NewReader("n\n"), &buf)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Contains(t, buf.String(), "Cancellation aborted.")
}

func TestCancelAllConfirmReadError(t *testing.T) {
	srv := runsServer(t, runsListBody(t, "",
		airBaseRun(111, "me@example.com", "GPU_1xA10", 1, "/Users/me@example.com/exp-a"),
	))
	w := newTestWorkspaceClient(t, srv.URL)

	var buf bytes.Buffer
	err := runCancelAll(t, w, flags.OutputText, iotest.ErrReader(errors.New("read failed")), &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read failed")
}

func TestCancelAllMeError(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything, iam.MeRequest{}).Return(nil, errors.New("nope"))

	var buf bytes.Buffer
	err := runCancelAll(t, m.WorkspaceClient, flags.OutputText, nil, &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve current user")
}

func TestCancelAllListError(t *testing.T) {
	// Me succeeds (default empty user), but listing active runs fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error_code":"INTERNAL","message":"boom"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	w := newTestWorkspaceClient(t, srv.URL)

	var buf bytes.Buffer
	err := runCancelAll(t, w, flags.OutputText, nil, &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list active runs")
}

func TestDisplayCancelPreview(t *testing.T) {
	var buf bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, nil, &buf, &buf, "", ""))

	started := "2026-06-05 17:32 UTC"
	rows := []listRow{
		{RunID: "111", Experiment: "exp-a", StartedAt: &started},
		{RunID: "222"}, // no experiment or start time -> N/A
	}
	displayCancelPreview(ctx, rows, "https://my-workspace.cloud.databricks.test")

	out := buf.String()
	assert.Contains(t, out, "Workspace: https://my-workspace.cloud.databricks.test")
	assert.Contains(t, out, "Found 2 active run(s) to cancel:")
	assert.Contains(t, out, "Run ID")
	assert.Contains(t, out, "111")
	assert.Contains(t, out, "exp-a")
	assert.Contains(t, out, "222")
	assert.Contains(t, out, na)
}
