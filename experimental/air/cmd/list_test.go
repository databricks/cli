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
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// airBaseRun builds a single-task AIR run (ai_runtime_task), as runs/list
// returns it with expand_tasks.
func airBaseRun(id int64, user, accelType string, count int, experiment string) jobs.BaseRun {
	return jobs.BaseRun{
		RunId:           id,
		RunName:         "run-" + strconv.FormatInt(id, 10),
		CreatorUserName: user,
		State:           &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{{AiRuntimeTask: &jobs.AiRuntimeTask{
			Experiment: experiment,
			Deployments: []jobs.DeploymentSpec{{
				Compute: jobs.ComputeSpec{AcceleratorType: jobs.ComputeSpecAcceleratorType(accelType), AcceleratorCount: count},
			}},
		}}},
	}
}

// airRun is the runs/get equivalent of airBaseRun, for the helpers that operate
// on a *jobs.Run.
func airRun(id int64, user, accelType string, count int, experiment string) jobs.Run {
	return *baseRunToRun(airBaseRun(id, user, accelType, count, experiment))
}

// runsListBody marshals one runs/list response page.
func runsListBody(t *testing.T, nextToken string, runs ...jobs.BaseRun) string {
	t.Helper()
	b, err := json.Marshal(jobs.ListRunsResponse{Runs: runs, NextPageToken: nextToken, HasMore: nextToken != ""})
	require.NoError(t, err)
	return string(b)
}

// runsServer serves one runs/list response body per call, repeating the last
// once exhausted, and a stub for any other request (the SDK config probe).
func runsServer(t *testing.T, bodies ...string) *httptest.Server {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
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
	runs := []jobs.BaseRun{
		airBaseRun(1, "me@example.com", "GPU_8xH100", 8, "/Users/me@example.com/exp-a"),
		{RunId: 2, CreatorUserName: "me@example.com", Tasks: []jobs.RunTask{{}}}, // not an AIR run
		airBaseRun(3, "other@example.com", "GPU_1xA10", 1, "/Users/other/exp-b"), // wrong user
		airBaseRun(5, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/exp-c"),
	}
	srv := runsServer(t, runsListBody(t, "", runs...))

	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		activeOnly: true,
		userFilter: "me@example.com",
	}).next(10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "5", rows[1].RunID)
}

func TestListAirRunsExperimentFilter(t *testing.T) {
	runs := []jobs.BaseRun{
		airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/qwen-train"),
		airBaseRun(2, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/llama-train"),
	}
	srv := runsServer(t, runsListBody(t, "", runs...))

	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		activeOnly: true,
		filters:    listFilters{Experiment: "qwen*"},
	}).next(10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "1", rows[0].RunID)
}

func TestListAirRunsLimitTruncates(t *testing.T) {
	runs := []jobs.BaseRun{
		airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "exp-a"),
		airBaseRun(2, "me@example.com", "GPU_1xH100", 1, "exp-b"),
		airBaseRun(3, "me@example.com", "GPU_1xH100", 1, "exp-c"),
	}
	srv := runsServer(t, runsListBody(t, "", runs...))

	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{activeOnly: true}).next(2)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "2", rows[1].RunID)
}

func TestListAirRunsPaginates(t *testing.T) {
	page1 := runsListBody(t, "tok", airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "exp-a"))
	page2 := runsListBody(t, "", airBaseRun(2, "me@example.com", "GPU_1xH100", 1, "exp-b"))
	srv := runsServer(t, page1, page2)

	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{activeOnly: true}).next(10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "2", rows[1].RunID)
}

// TestRunFetcherResumesAcrossCalls covers the lazy paging the interactive table
// relies on: a next() that stops mid-page must resume on the following call, then
// report exhaustion.
func TestRunFetcherResumesAcrossCalls(t *testing.T) {
	runs := []jobs.BaseRun{
		airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "exp-a"),
		airBaseRun(2, "me@example.com", "GPU_1xH100", 1, "exp-b"),
		airBaseRun(3, "me@example.com", "GPU_1xH100", 1, "exp-c"),
	}
	srv := runsServer(t, runsListBody(t, "", runs...))
	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{activeOnly: true})

	first, err := f.next(2)
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, "1", first[0].RunID)
	assert.Equal(t, "2", first[1].RunID)

	second, err := f.next(2)
	require.NoError(t, err)
	require.Len(t, second, 1) // only the leftover remains
	assert.Equal(t, "3", second[0].RunID)
	assert.True(t, f.exhausted)

	third, err := f.next(2)
	require.NoError(t, err)
	assert.Empty(t, third)
}

func TestBuildListRowFromRun(t *testing.T) {
	run := airRun(842552489592352, "me@example.com", "GPU_1xA10", 1, "my-first-air-run")
	run.StartTime = 1700000000000
	run.EndTime = 1700000012000
	run.State = &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateSuccess}

	assert.True(t, isAirRun(&run))
	assert.Equal(t, "my-first-air-run", jobExperiment(&run))
	gpu, count := jobCompute(&run)
	assert.Equal(t, "GPU_1xA10", gpu)
	assert.Equal(t, 1, count)

	row := buildListRow(&run, "https://example.test", 0)
	assert.Equal(t, "842552489592352", row.RunID)
	assert.Equal(t, "SUCCESS", row.Status)
	assert.Equal(t, "my-first-air-run", row.Experiment)
	assert.Equal(t, "1x A10", row.Accelerators)
	assert.Equal(t, "12s", row.Duration)
}

func TestBuildListRow(t *testing.T) {
	run := airRun(123, "me@example.com", "GPU_8xH100", 8, "/Users/me@example.com/exp")
	run.StartTime = 1700000000000
	run.EndTime = 1700000012000
	run.State = &jobs.RunState{ResultState: jobs.RunResultStateSuccess}

	row := buildListRow(&run, "https://example.test", 0)
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
	row := buildListRow(&jobs.Run{RunId: 7}, "https://example.test", 0)
	assert.Equal(t, "-", row.Experiment)
	assert.Equal(t, "-", row.Duration)
	assert.Equal(t, "-", row.Accelerators)
	assert.Equal(t, "-", row.MLflowURL)
	assert.Equal(t, "UNKNOWN", row.Status)
	assert.Nil(t, row.StartedAt)
}

func TestBuildListRowSweep(t *testing.T) {
	run := jobs.Run{RunId: 9, Tasks: []jobs.RunTask{{
		ForEachTask: &jobs.RunForEachTask{Task: jobs.Task{AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "sweep"}}},
	}}}
	assert.True(t, buildListRow(&run, "https://example.test", 0).IsSweep)
	assert.Equal(t, "sweep", buildListRow(&run, "https://example.test", 0).Experiment)
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
