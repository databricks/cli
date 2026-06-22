package aircmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWorkflow builds a single-task AIR workflow for a given user and compute,
// as ListTrainingWorkflows would return it.
func testWorkflow(id int64, user, gpuType string, count int, experiment string) trainingWorkflow {
	var w trainingWorkflow
	w.JobRunID = strconv.FormatInt(id, 10)
	w.Metadata.CreatorName = user
	w.Spec.Compute.HardwareAcceleratorType = gpuType
	w.Spec.Compute.AcceleratorCount = count
	w.Status.State = "TRAINING_WORKFLOW_STATE_RUNNING"
	w.Status.Job.Name = "run-" + strconv.FormatInt(id, 10)
	w.Status.Mlflow.Experiment = experiment
	return w
}

// workflowsBody marshals a ListTrainingWorkflows response page.
func workflowsBody(t *testing.T, nextToken string, wfs ...trainingWorkflow) string {
	t.Helper()
	b, err := json.Marshal(trainingWorkflowsResponse{TrainingWorkflows: wfs, NextPageToken: nextToken})
	require.NoError(t, err)
	return string(b)
}

// workflowsServer serves one response body per call to the workflows endpoint,
// repeating the last body once exhausted, and a stub for any other request (the
// SDK's well-known config discovery).
func workflowsServer(t *testing.T, bodies ...string) *httptest.Server {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == listWorkflowsPath {
			body := bodies[min(call, len(bodies)-1)]
			call++
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListAirRunsAppliesClientFilters(t *testing.T) {
	qwen := testWorkflow(1, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/qwen-train")
	llama := testWorkflow(2, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/llama-train")
	srv := workflowsServer(t, workflowsBody(t, "", qwen, llama))

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		limit:   10,
		filters: listFilters{Experiment: "qwen*"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "1", rows[0].RunID)
}

func TestListAirRunsLimitTruncates(t *testing.T) {
	wfs := []trainingWorkflow{
		testWorkflow(1, "me@example.com", "GPU_1xH100", 1, "exp-a"),
		testWorkflow(2, "me@example.com", "GPU_1xH100", 1, "exp-b"),
		testWorkflow(3, "me@example.com", "GPU_1xH100", 1, "exp-c"),
	}
	srv := workflowsServer(t, workflowsBody(t, "", wfs...))

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{limit: 2})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "2", rows[1].RunID)
}

func TestListAirRunsNoLimitFetchesAll(t *testing.T) {
	// limit 0 means "all": follow the page token to the end with no early stop.
	page1 := workflowsBody(t, "tok", testWorkflow(1, "me@example.com", "GPU_1xH100", 1, "exp-a"))
	page2 := workflowsBody(t, "", testWorkflow(2, "me@example.com", "GPU_1xH100", 1, "exp-b"))
	srv := workflowsServer(t, page1, page2)

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{limit: 0})
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestListAirRunsPaginates(t *testing.T) {
	// First page returns one run and a continuation token; the loop must follow it
	// and stop when the token is empty.
	page1 := workflowsBody(t, "tok", testWorkflow(1, "me@example.com", "GPU_1xH100", 1, "exp-a"))
	page2 := workflowsBody(t, "", testWorkflow(2, "me@example.com", "GPU_1xH100", 1, "exp-b"))
	srv := workflowsServer(t, page1, page2)

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "2", rows[1].RunID)
}

func TestBuildListRow(t *testing.T) {
	var w trainingWorkflow
	w.JobRunID = "123"
	w.Metadata.CreatorName = "me@example.com"
	w.Spec.Compute.HardwareAcceleratorType = "GPU_8xH100"
	w.Spec.Compute.AcceleratorCount = 8
	w.Status.State = "TRAINING_WORKFLOW_STATE_TERMINATED_COMPLETED"
	w.Status.StartTime = "2026-06-05T17:32:39.000Z"
	w.Status.EndTime = "2026-06-05T17:32:51.000Z"
	w.Status.Job.Name = "my-run"
	w.Status.Mlflow.Experiment = "/Users/me@example.com/exp"
	w.Status.Mlflow.ExperimentID = "E1"
	w.Status.Mlflow.RunID = "R1"

	row := buildListRow(&w, "https://h.test")
	assert.Equal(t, "123", row.RunID)
	assert.Equal(t, "my-run", row.RunName)
	assert.Equal(t, "me@example.com", row.User)
	assert.Equal(t, "SUCCESS", row.Status)
	assert.Equal(t, "exp", row.Experiment)
	assert.Equal(t, "12s", row.Duration)
	assert.Equal(t, "8x H100", row.Accelerators)
	assert.Equal(t, "https://h.test/ml/experiments/E1/runs/R1/artifacts/logs/node_0", row.MLflowURL)
	assert.False(t, row.IsSweep)
	require.NotNil(t, row.StartedAt)
	assert.Equal(t, "2026-06-05T17:32:39+00:00", *row.StartedAt)
}

func TestBuildListRowDashFallbacks(t *testing.T) {
	// A workflow with no experiment, compute, MLflow IDs, or start time falls back
	// to dashes for the optional columns and UNKNOWN for the unset state.
	row := buildListRow(&trainingWorkflow{}, "https://h.test")
	assert.Equal(t, "-", row.Experiment)
	assert.Equal(t, "-", row.Duration)
	assert.Equal(t, "-", row.Accelerators)
	assert.Equal(t, "-", row.MLflowURL)
	assert.Equal(t, "UNKNOWN", row.Status)
	assert.Nil(t, row.StartedAt)
}

func TestBuildListRowSweep(t *testing.T) {
	// task_run_id is set only for sweeps, so its presence marks the row.
	w := trainingWorkflow{TaskRunID: "456"}
	assert.True(t, buildListRow(&w, "https://h.test").IsSweep)
}

func TestListInvalidLimit(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := newListCommand()
	cmd.SetContext(ctx)
	require.NoError(t, cmd.Flags().Set("limit", "0"))

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --limit")
}

func TestListInvalidFilter(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)
	cmd := newListCommand()
	cmd.SetContext(ctx)
	require.NoError(t, cmd.Flags().Set("filter", "bogus=1"))

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported --filter key")
}
