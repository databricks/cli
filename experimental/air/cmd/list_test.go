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

// airJobRun builds a single-task AIR run (ai_runtime_task), as runs/list returns
// it with expand_tasks.
func airJobRun(id int64, user, accelType string, count int, experiment string) jobRun {
	return jobRun{
		RunID:           id,
		RunName:         "run-" + strconv.FormatInt(id, 10),
		CreatorUserName: user,
		State:           jobState{LifeCycleState: "RUNNING"},
		Tasks: []jobTask{{AiRuntimeTask: &jobAiRuntimeTask{
			Experiment: experiment,
			Deployments: []aiRuntimeDeploy{{
				Compute: airCompute{AcceleratorType: accelType, AcceleratorCount: count},
			}},
		}}},
	}
}

// runsListBody marshals one runs/list response page.
func runsListBody(t *testing.T, nextToken string, runs ...jobRun) string {
	t.Helper()
	b, err := json.Marshal(jobsRunsListResponse{Runs: runs, NextPageToken: nextToken})
	require.NoError(t, err)
	return string(b)
}

// runsServer serves one runs/list response body per call, repeating the last
// once exhausted, and a stub for any other request (the SDK config probe).
func runsServer(t *testing.T, bodies ...string) *httptest.Server {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == jobsRunsListPath {
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

func TestListAirRunsFiltersUserAndType(t *testing.T) {
	runs := []jobRun{
		airJobRun(1, "me@example.com", "GPU_8xH100", 8, "/Users/me@example.com/exp-a"),
		{RunID: 2, CreatorUserName: "me@example.com", Tasks: []jobTask{{}}},     // not an AIR run
		airJobRun(3, "other@example.com", "GPU_1xA10", 1, "/Users/other/exp-b"), // wrong user
		airJobRun(5, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/exp-c"),
	}
	srv := runsServer(t, runsListBody(t, "", runs...))

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		userFilter: "me@example.com",
		limit:      10,
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "5", rows[1].RunID)
}

func TestListAirRunsExperimentFilter(t *testing.T) {
	runs := []jobRun{
		airJobRun(1, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/qwen-train"),
		airJobRun(2, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/llama-train"),
	}
	srv := runsServer(t, runsListBody(t, "", runs...))

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		limit:   10,
		filters: listFilters{Experiment: "qwen*"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "1", rows[0].RunID)
}

func TestListAirRunsLimitTruncates(t *testing.T) {
	runs := []jobRun{
		airJobRun(1, "me@example.com", "GPU_1xH100", 1, "exp-a"),
		airJobRun(2, "me@example.com", "GPU_1xH100", 1, "exp-b"),
		airJobRun(3, "me@example.com", "GPU_1xH100", 1, "exp-c"),
	}
	srv := runsServer(t, runsListBody(t, "", runs...))

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{limit: 2})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "2", rows[1].RunID)
}

func TestListAirRunsPaginates(t *testing.T) {
	page1 := runsListBody(t, "tok", airJobRun(1, "me@example.com", "GPU_1xH100", 1, "exp-a"))
	page2 := runsListBody(t, "", airJobRun(2, "me@example.com", "GPU_1xH100", 1, "exp-b"))
	srv := runsServer(t, page1, page2)

	rows, err := listAirRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "2", rows[1].RunID)
}

// TestFetchJobRunsParsesAiRuntimeTask pins the raw parse against the real
// runs/get shape, since the typed SDK omits ai_runtime_task.
func TestFetchJobRunsParsesAiRuntimeTask(t *testing.T) {
	body := `{"runs":[{
		"run_id": 842552489592352,
		"run_name": "my-first-air-run",
		"creator_user_name": "me@example.com",
		"start_time": 1700000000000,
		"end_time": 1700000012000,
		"state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"},
		"tasks": [{
			"ai_runtime_task": {
				"experiment": "my-first-air-run",
				"deployments": [{"compute": {"accelerator_count": 1, "accelerator_type": "GPU_1xA10"}}]
			}
		}]
	}]}`
	srv := runsServer(t, body)

	resp, err := fetchJobRunsPage(t.Context(), newTestWorkspaceClient(t, srv.URL), map[string]any{})
	require.NoError(t, err)
	require.Len(t, resp.Runs, 1)
	run := &resp.Runs[0]
	assert.True(t, isAirRun(run))
	assert.Equal(t, "my-first-air-run", jobExperiment(run))
	gpu, count := jobCompute(run)
	assert.Equal(t, "GPU_1xA10", gpu)
	assert.Equal(t, 1, count)

	row := buildListRow(run)
	assert.Equal(t, "842552489592352", row.RunID)
	assert.Equal(t, "SUCCESS", row.Status)
	assert.Equal(t, "my-first-air-run", row.Experiment)
	assert.Equal(t, "1x A10", row.Accelerators)
	assert.Equal(t, "12s", row.Duration)
}

func TestBuildListRow(t *testing.T) {
	run := airJobRun(123, "me@example.com", "GPU_8xH100", 8, "/Users/me@example.com/exp")
	run.StartTime = 1700000000000
	run.EndTime = 1700000012000
	run.State = jobState{ResultState: "SUCCESS"}

	row := buildListRow(&run)
	assert.Equal(t, "123", row.RunID)
	assert.Equal(t, "me@example.com", row.User)
	assert.Equal(t, "SUCCESS", row.Status)
	assert.Equal(t, "exp", row.Experiment)
	assert.Equal(t, "12s", row.Duration)
	assert.Equal(t, "8x H100", row.Accelerators)
	assert.Equal(t, "-", row.MLflowURL)
	assert.False(t, row.IsSweep)
	require.NotNil(t, row.StartedAt)
}

func TestBuildListRowDashFallbacks(t *testing.T) {
	// A run with no task, compute, or start time falls back to dashes and UNKNOWN.
	row := buildListRow(&jobRun{RunID: 7})
	assert.Equal(t, "-", row.Experiment)
	assert.Equal(t, "-", row.Duration)
	assert.Equal(t, "-", row.Accelerators)
	assert.Equal(t, "-", row.MLflowURL)
	assert.Equal(t, "UNKNOWN", row.Status)
	assert.Nil(t, row.StartedAt)
}

func TestBuildListRowSweep(t *testing.T) {
	run := jobRun{RunID: 9, Tasks: []jobTask{{
		ForEachTask: &forEachTask{Task: jobTask{AiRuntimeTask: &jobAiRuntimeTask{Experiment: "sweep"}}},
	}}}
	assert.True(t, buildListRow(&run).IsSweep)
	assert.Equal(t, "sweep", buildListRow(&run).Experiment)
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
