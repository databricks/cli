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
			// Readers take the run narrowed to its AI Runtime task.
			shown := defaultTaskRun(run)
			assert.Equal(t, tt.experiment, jobExperiment(shown), "jobExperiment")
			assert.Equal(t, tt.sweep, isSweep(shown), "isSweep")
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

	first := defaultTaskRun(run)
	assert.Equal(t, "a", first.Tasks[0].TaskKey, "the default is the first AI Runtime task, not the notebook")
	assert.Equal(t, "a", airTaskKey(first))
	notebookOnly := &jobs.Run{RunId: 2, Tasks: []jobs.RunTask{notebookTask("prep", 21)}}
	assert.Same(t, notebookOnly, defaultTaskRun(notebookOnly), "a run without an AI Runtime task is not narrowed")
	assert.Empty(t, airTaskKey(notebookOnly))
}

func TestJobTimingUsesTheAirTask(t *testing.T) {
	prep := notebookTask("prep", 11)
	prep.StartTime, prep.EndTime = 100, 150
	train := aiRuntimeTask("train", 12, "exp", "GPU_1xA10", 1)
	train.StartTime, train.EndTime = 200, 260
	run := &jobs.Run{RunId: 1, StartTime: 90, EndTime: 300, Tasks: []jobs.RunTask{prep, train}}

	start, end := jobTiming(defaultTaskRun(run))
	assert.Equal(t, int64(200), start)
	assert.Equal(t, int64(260), end)
}

// finished returns a task run whose latest attempt ended in result.
func finished(task jobs.RunTask, life jobs.RunLifeCycleState, result jobs.RunResultState) jobs.RunTask {
	task.State = &jobs.RunState{LifeCycleState: life, ResultState: result}
	return task
}

func TestTaskRunReportsTheTasksOwnState(t *testing.T) {
	failedRun := &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateFailed}
	tests := []struct {
		name     string
		runState *jobs.RunState
		tasks    []jobs.RunTask
		key      string
		want     string
	}{
		{
			name:     "succeeded task next to a failed one",
			runState: failedRun,
			tasks: []jobs.RunTask{
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess),
				finished(aiRuntimeTask("eval", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed),
			},
			key:  "train",
			want: "SUCCESS",
		},
		{
			name:     "the failed task",
			runState: failedRun,
			tasks: []jobs.RunTask{
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess),
				finished(aiRuntimeTask("eval", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed),
			},
			key:  "eval",
			want: "FAILED",
		},
		{
			name:     "skipped because the prep notebook failed",
			runState: failedRun,
			tasks: []jobs.RunTask{
				finished(notebookTask("prep", 11), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed),
				finished(aiRuntimeTask("train", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateSkipped, jobs.RunResultStateUpstreamFailed),
			},
			key:  "train",
			want: "UPSTREAM_FAILED",
		},
		{
			name:     "latest attempt is the one reported",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateSuccess},
			tasks: []jobs.RunTask{
				finished(notebookTask("prep", 10), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess),
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed),
				func() jobs.RunTask {
					t := finished(aiRuntimeTask("train", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess)
					t.AttemptNumber = 1
					return t
				}(),
			},
			key:  "train",
			want: "SUCCESS",
		},
		{
			name:     "a failed attempt in an active run may be retried, so the run's state stands",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			tasks: []jobs.RunTask{
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed),
				finished(notebookTask("report", 12), jobs.RunLifeCycleStateRunning, ""),
			},
			key:  "train",
			want: "RUNNING",
		},
		{
			name:     "a pending task in a running run appears pending",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			tasks: []jobs.RunTask{
				finished(notebookTask("prep", 11), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess),
				finished(aiRuntimeTask("train", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStatePending, ""),
			},
			key:  "train",
			want: "PENDING",
		},
		{
			name:     "a queued task in a running run appears pending",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			tasks: []jobs.RunTask{
				finished(notebookTask("prep", 11), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess),
				finished(aiRuntimeTask("train", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateQueued, ""),
			},
			key:  "train",
			want: "PENDING",
		},
		{
			name:     "a task blocked on its upstream task is reported as blocked",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			tasks: []jobs.RunTask{
				finished(notebookTask("prep", 11), jobs.RunLifeCycleStateRunning, ""),
				finished(aiRuntimeTask("train", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateBlocked, ""),
			},
			key:  "train",
			want: "BLOCKED",
		},
		{
			name:     "a skipped task keeps its state while another branch still runs",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			tasks: []jobs.RunTask{
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateRunning, ""),
				finished(notebookTask("prep_eval", 12), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed),
				finished(aiRuntimeTask("eval", 13, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateSkipped, jobs.RunResultStateUpstreamFailed),
			},
			key:  "eval",
			want: "UPSTREAM_FAILED",
		},
		{
			name:     "a cancelled task keeps its state while another task still runs",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			tasks: []jobs.RunTask{
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateCanceled),
				finished(notebookTask("report", 12), jobs.RunLifeCycleStateRunning, ""),
			},
			key:  "train",
			want: "CANCELED",
		},
		{
			name:     "a timed-out attempt in an active run may be retried, so the run's state stands",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
			tasks: []jobs.RunTask{
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateTimedout),
				finished(notebookTask("report", 12), jobs.RunLifeCycleStateRunning, ""),
			},
			key:  "train",
			want: "RUNNING",
		},
		{
			name:     "the run's only task keeps the run's state",
			runState: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateTimedout},
			tasks: []jobs.RunTask{
				finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess),
			},
			key:  "train",
			want: "TIMEDOUT",
		},
		{
			name:     "a task without a state falls back to the run",
			runState: failedRun,
			tasks: []jobs.RunTask{
				finished(notebookTask("prep", 10), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed),
				aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1),
			},
			key:  "train",
			want: "FAILED",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &jobs.Run{RunId: 1, State: tt.runState, Tasks: tt.tasks}
			narrowed := taskRun(run, tt.key)
			require.NotNil(t, narrowed)
			assert.Equal(t, tt.want, displayRunStatus(narrowed))
			assert.Equal(t, tt.want, buildListRow(narrowed, "https://example.test", 0).Status)
			assert.Same(t, tt.runState, run.State, "the source run keeps its state")
		})
	}
}

func TestTaskRunOfASkippedTaskHasNoTiming(t *testing.T) {
	train := finished(aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateFailed)
	train.StartTime, train.EndTime = 200_000, 260_000
	eval := finished(aiRuntimeTask("eval", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateSkipped, jobs.RunResultStateUpstreamFailed)
	run := &jobs.Run{
		RunId:     1,
		StartTime: 100_000,
		EndTime:   300_000,
		State:     &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateFailed},
		Tasks:     []jobs.RunTask{train, eval},
	}

	skipped := taskRun(run, "eval")
	assert.Nil(t, startedAt(skipped), "a task that never started reports no start, not the run's")
	assert.Nil(t, durationSeconds(skipped))

	ran := taskRun(run, "train")
	require.NotNil(t, durationSeconds(ran))
	assert.Equal(t, int64(60), *durationSeconds(ran), "train reports its own 60s, not the run's 200s")
}

func TestRunLevelWaitingStatesAreReportedAsSuch(t *testing.T) {
	for _, life := range []jobs.RunLifeCycleState{
		jobs.RunLifeCycleStateQueued, jobs.RunLifeCycleStateBlocked, jobs.RunLifeCycleStateWaitingForRetry,
	} {
		t.Run(string(life), func(t *testing.T) {
			run := &jobs.Run{RunId: 1, State: &jobs.RunState{LifeCycleState: life}, Tasks: []jobs.RunTask{aiRuntimeTask("train", 11, "e", "GPU_1xA10", 1)}}
			assert.False(t, runAppearsPending(run))
			assert.Equal(t, string(life), displayRunStatus(run))
			assert.Equal(t, string(life), detailedDisplayRunStatus(run, ""))
		})
	}

	pending := &jobs.Run{RunId: 1, State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStatePending}}
	assert.Equal(t, waitingForWorkloadStatus, detailedDisplayRunStatus(pending, ""))

	// The notebook-first DAB shape: train waits on prep, not on compute.
	blocked := &jobs.Run{
		RunId: 1,
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{
			finished(notebookTask("prep", 11), jobs.RunLifeCycleStateRunning, ""),
			finished(aiRuntimeTask("train", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateBlocked, ""),
		},
	}
	assert.Equal(t, "BLOCKED", detailedDisplayRunStatus(taskRun(blocked, "train"), ""))

	// The same task queued for compute reads like a single-task run's wait.
	queued := &jobs.Run{
		RunId: 1,
		State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		Tasks: []jobs.RunTask{
			finished(notebookTask("prep", 11), jobs.RunLifeCycleStateTerminated, jobs.RunResultStateSuccess),
			finished(aiRuntimeTask("train", 12, "e", "GPU_1xA10", 1), jobs.RunLifeCycleStateQueued, ""),
		},
	}
	assert.Equal(t, waitingForWorkloadStatus, detailedDisplayRunStatus(taskRun(queued, "train"), ""))
}
