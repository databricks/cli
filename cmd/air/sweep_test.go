package aircmd

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindForEachTask(t *testing.T) {
	// No tasks at all.
	assert.Nil(t, findForEachTask(&jobs.GetRunResponse{}))

	// A task that is not a foreach.
	assert.Nil(t, findForEachTask(&jobs.GetRunResponse{Tasks: []jobs.RunTask{{TaskKey: new("a")}}}))

	// The foreach task is found even when it isn't first.
	run := &jobs.GetRunResponse{Tasks: []jobs.RunTask{
		{TaskKey: new("a")},
		{TaskKey: new("sweep"), ForEachTask: &jobs.ForEachTask{}},
	}}
	got := findForEachTask(run)
	require.NotNil(t, got)
	assert.Equal(t, "sweep", utils.Deref(got.TaskKey))
}

func sweepTaskFixture() *jobs.RunTask {
	return &jobs.RunTask{
		TaskKey:     new("sweep"),
		RunId:       new(int64(99)),
		ForEachTask: &jobs.ForEachTask{},
	}
}

// sweepServer serves runs/get for the parent run 1 (with the foreach task's
// stats) and for the foreach task run 99 (with iterationsBody, or an error when
// it is empty).
func sweepServer(t *testing.T, iterationsBody string) *sdkclients.Clients {
	t.Helper()
	return fakeWorkspace(t, map[string]http.HandlerFunc{
		"GET /api/2.2/jobs/runs/get": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("run_id") {
			case "1":
				_, _ = w.Write([]byte(`{"run_id":1,"tasks":[{"task_key":"sweep","run_id":99,"for_each_task":{"stats":{"task_run_stats":{
					"total_iterations":4,"succeeded_iterations":1,"failed_iterations":1,"active_iterations":2,"completed_iterations":2}}}}]}`))
			case "99":
				if iterationsBody == "" {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"Run 99 does not exist."}`))
					return
				}
				_, _ = w.Write([]byte(iterationsBody))
			default:
				t.Errorf("unexpected runs/get for run %s", r.URL.Query().Get("run_id"))
			}
		},
	})
}

func TestBuildSweepInfo(t *testing.T) {
	ctx := t.Context()

	t.Run("counts and iteration rows", func(t *testing.T) {
		w := sweepServer(t, `{"iterations":[{"task_key":"iter_0","run_id":100,"state":{"result_state":"SUCCESS"},
			"gen_ai_compute_task":{"mlflow_experiment_name":"/Users/me@example.com/exp"}}]}`)

		info := buildSweepInfo(ctx, w, 1, sweepTaskFixture())
		assert.Equal(t, 4, info.Total)
		assert.Equal(t, 2, info.Completed)
		assert.Equal(t, 1, info.Succeeded)
		assert.Equal(t, 1, info.Failed)
		assert.Equal(t, 2, info.Active)
		require.Len(t, info.Tasks, 1)
		assert.Equal(t, "iter_0", info.Tasks[0].TaskKey)
		assert.Equal(t, "100", info.Tasks[0].RunID)
		assert.Equal(t, "SUCCESS", info.Tasks[0].Status)
		assert.Equal(t, "exp", info.Tasks[0].Experiment)
	})

	t.Run("iteration lookup failure still returns counts", func(t *testing.T) {
		w := sweepServer(t, "")

		info := buildSweepInfo(ctx, w, 1, sweepTaskFixture())
		assert.Equal(t, 4, info.Total)
		assert.Empty(t, info.Tasks)
	})
}
