package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGetServer serves one runs/get body and a stub for the SDK config probe.
func runGetServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/get" {
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchJobRunParsesAiRuntimeTask(t *testing.T) {
	body := `{
		"run_id": 5,
		"tasks": [{
			"ai_runtime_task": {
				"experiment": "my-exp",
				"deployments": [{
					"command_path": "/Workspace/Users/me/.air/cli_launch/my-exp/my-exp_abc/command.sh",
					"compute": {"accelerator_count": 1, "accelerator_type": "GPU_1xA10"}
				}]
			}
		}]
	}`
	srv := runGetServer(t, body)

	run, err := fetchJobRun(t.Context(), newTestWorkspaceClient(t, srv.URL), 5)
	require.NoError(t, err)
	assert.Equal(t, "my-exp", jobExperiment(run))
	gpuType, count := jobCompute(run)
	assert.Equal(t, "GPU_1xA10", gpuType)
	assert.Equal(t, 1, count)
}

type runGetResponse struct {
	status int
	body   string
}

// runGetByIDServer serves runs/get per run_id and a stub for the SDK config
// probe; ids missing from responses return a run with that id.
func runGetByIDServer(t *testing.T, responses map[string]runGetResponse) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.2/jobs/runs/get" {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		id := r.URL.Query().Get("run_id")
		resp, ok := responses[id]
		if !ok {
			resp = runGetResponse{status: http.StatusOK, body: `{"run_id": ` + id + `}`}
		}
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runIDs(runs []*jobs.Run) []int64 {
	ids := make([]int64, len(runs))
	for i, r := range runs {
		ids[i] = r.RunId
	}
	return ids
}

func TestHydrateJobRunsDropsUnviewableAndPurgedRuns(t *testing.T) {
	srv := runGetByIDServer(t, map[string]runGetResponse{
		"2": {http.StatusBadRequest, `{"error_code": "INVALID_PARAMETER_VALUE", "message": "Run 2 does not exist."}`},
		"4": {http.StatusForbidden, `{"error_code": "PERMISSION_DENIED", "message": "denied"}`},
		"5": {http.StatusNotFound, `{"error_code": "NOT_FOUND", "message": "not found"}`},
	})

	runs, err := hydrateJobRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), []int64{1, 2, 3, 4, 5, 6})
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 3, 6}, runIDs(runs))
}

func TestHydrateJobRunsFailsOnOtherErrors(t *testing.T) {
	tests := []struct {
		name    string
		resp    runGetResponse
		wantErr string
	}{
		{
			name:    "internal error",
			resp:    runGetResponse{http.StatusInternalServerError, `{"error_code": "INTERNAL_ERROR", "message": "boom"}`},
			wantErr: "failed to get run 2: boom",
		},
		{
			name:    "other invalid parameter",
			resp:    runGetResponse{http.StatusBadRequest, `{"error_code": "INVALID_PARAMETER_VALUE", "message": "run_id must be positive"}`},
			wantErr: "failed to get run 2: run_id must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := runGetByIDServer(t, map[string]runGetResponse{"2": tt.resp})

			_, err := hydrateJobRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), []int64{1, 2, 3})
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

// notebookTask builds a non-AIR task run, like the prep step that precedes the
// ai_runtime_task in the official multi-task DAB example.
func notebookTask(key string, runID int64) jobs.RunTask {
	return jobs.RunTask{TaskKey: key, RunId: runID, NotebookTask: &jobs.NotebookTask{NotebookPath: "/prep"}}
}

// aiRuntimeTask builds a direct ai_runtime_task task run.
func aiRuntimeTask(key string, runID int64, experiment, accelType string, count int) jobs.RunTask {
	return jobs.RunTask{TaskKey: key, RunId: runID, AiRuntimeTask: &jobs.AiRuntimeTask{
		Experiment: experiment,
		Deployments: []jobs.DeploymentSpec{{
			Compute: jobs.ComputeSpec{AcceleratorType: jobs.ComputeSpecAcceleratorType(accelType), AcceleratorCount: count},
		}},
	}}
}

// forEachAiRuntimeTask builds a foreach (sweep) task whose iterated task is an
// ai_runtime_task.
func forEachAiRuntimeTask(key string, runID int64, experiment string) jobs.RunTask {
	return jobs.RunTask{TaskKey: key, RunId: runID, ForEachTask: &jobs.RunForEachTask{
		Task: jobs.Task{AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: experiment}},
	}}
}

func TestAirTaskSelection(t *testing.T) {
	tests := []struct {
		name       string
		tasks      []jobs.RunTask
		air        bool
		keys       []string
		experiment string
		sweep      bool
	}{
		{
			name:       "AIR task first",
			tasks:      []jobs.RunTask{aiRuntimeTask("train", 11, "exp-train", "GPU_1xA10", 1), notebookTask("report", 12)},
			air:        true,
			keys:       []string{"train"},
			experiment: "exp-train",
		},
		{
			name:       "AIR task second",
			tasks:      []jobs.RunTask{notebookTask("prep", 11), aiRuntimeTask("train", 12, "exp-train", "GPU_1xA10", 1)},
			air:        true,
			keys:       []string{"train"},
			experiment: "exp-train",
		},
		{
			name:       "two AIR tasks",
			tasks:      []jobs.RunTask{aiRuntimeTask("a", 11, "exp-a", "GPU_1xA10", 1), aiRuntimeTask("b", 12, "exp-b", "GPU_8xH100", 8)},
			air:        true,
			keys:       []string{"a", "b"},
			experiment: "exp-a",
		},
		{
			name:       "foreach AIR task",
			tasks:      []jobs.RunTask{forEachAiRuntimeTask("sweep", 11, "exp-sweep")},
			air:        true,
			keys:       []string{"sweep"},
			experiment: "exp-sweep",
			sweep:      true,
		},
		{
			name:       "foreach AIR task after a notebook",
			tasks:      []jobs.RunTask{notebookTask("prep", 11), forEachAiRuntimeTask("sweep", 12, "exp-sweep")},
			air:        true,
			keys:       []string{"sweep"},
			experiment: "exp-sweep",
			sweep:      true,
		},
		{
			name:  "retries share one key",
			tasks: []jobs.RunTask{notebookTask("prep", 11), aiRuntimeTask("train", 12, "exp-train", "GPU_1xA10", 1), aiRuntimeTask("train", 13, "exp-train", "GPU_1xA10", 1)},
			air:   true, keys: []string{"train"}, experiment: "exp-train",
		},
		{
			name:  "no AIR task",
			tasks: []jobs.RunTask{notebookTask("prep", 11)},
		},
		{
			name: "no tasks",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &jobs.Run{RunId: 1, Tasks: tt.tasks}
			assert.Equal(t, tt.air, isAirRun(run), "isAirRun")
			assert.Equal(t, tt.keys, airTaskKeys(run), "airTaskKeys")
			assert.Equal(t, tt.experiment, jobExperiment(run), "jobExperiment")
			assert.Equal(t, tt.sweep, isSweep(run), "isSweep")
			if len(tt.tasks) == 0 {
				assert.Nil(t, firstTask(run), "firstTask")
			}
		})
	}
}

func TestTaskRunNarrowsToOneAirTask(t *testing.T) {
	run := &jobs.Run{
		RunId: 1,
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{
			notebookTask("prep", 11),
			aiRuntimeTask("a", 12, "exp-a", "GPU_1xA10", 1),
			{TaskKey: "a", RunId: 13, AttemptNumber: 1, AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp-a"}},
			aiRuntimeTask("b", 14, "exp-b", "GPU_8xH100", 8),
		},
	}

	a := taskRun(run, "a")
	require.NotNil(t, a)
	assert.Equal(t, int64(1), a.RunId)
	assert.Equal(t, run.State, a.State)
	require.Len(t, a.Tasks, 2, "both attempts of task a, in order")
	assert.Equal(t, int64(12), a.Tasks[0].RunId)
	assert.Equal(t, int64(13), a.Tasks[1].RunId)
	assert.Equal(t, "exp-a", jobExperiment(a))

	b := taskRun(run, "b")
	require.NotNil(t, b)
	assert.Equal(t, "exp-b", jobExperiment(b))
	gpu, n := jobCompute(b)
	assert.Equal(t, "GPU_8xH100", gpu)
	assert.Equal(t, 8, n)

	assert.Len(t, run.Tasks, 4, "the source run is left intact")
	assert.Nil(t, taskRun(run, "prep"), "a non-AIR task is not selectable")
	assert.Nil(t, taskRun(run, "missing"))
}

func TestJobTimingUsesTheAirTask(t *testing.T) {
	prep := notebookTask("prep", 11)
	prep.StartTime, prep.EndTime = 100, 150
	train := aiRuntimeTask("train", 12, "exp", "GPU_1xA10", 1)
	train.StartTime, train.EndTime = 200, 260
	run := &jobs.Run{RunId: 1, StartTime: 90, EndTime: 300, Tasks: []jobs.RunTask{prep, train}}

	start, end := jobTiming(run)
	assert.Equal(t, int64(200), start)
	assert.Equal(t, int64(260), end)
}
