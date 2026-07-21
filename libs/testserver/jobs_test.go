package testserver

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func submitRun(t *testing.T, workspace *FakeWorkspace, request jobs.SubmitRun) jobs.SubmitRunResponse {
	t.Helper()
	body, err := json.Marshal(request)
	require.NoError(t, err)

	response := workspace.JobsSubmit(Request{Body: body})
	require.Equal(t, 0, response.StatusCode)

	submitResponse, ok := response.Body.(jobs.SubmitRunResponse)
	require.True(t, ok)
	return submitResponse
}

func getRun(t *testing.T, workspace *FakeWorkspace, runID int64) jobs.Run {
	t.Helper()
	response := workspace.JobsGetRun(Request{
		URL: &url.URL{RawQuery: "run_id=" + strconv.FormatInt(runID, 10)},
	})
	require.Equal(t, 0, response.StatusCode)

	run, ok := response.Body.(jobs.Run)
	require.True(t, ok)
	return run
}

func TestJobsSubmit_RecordsRunAndReportsRunningTasks(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	submitResponse := submitRun(t, workspace, jobs.SubmitRun{
		RunName: "ssh-tunnel",
		Tasks: []jobs.SubmitTask{
			{TaskKey: "main", EnvironmentKey: "default"},
		},
	})
	require.NotZero(t, submitResponse.RunId)

	run := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, "ssh-tunnel", run.RunName)
	assert.Equal(t, jobs.RunTypeSubmitRun, run.RunType)

	require.Len(t, run.Tasks, 1)
	task := run.Tasks[0]
	assert.Equal(t, "main", task.TaskKey)
	// ssh connect's waitForJobToStart polls the V2 per-task status.
	require.NotNil(t, task.Status)
	assert.Equal(t, jobs.RunLifecycleStateV2StateRunning, task.Status.State)
}

func TestJobsSubmit_DefaultsRunNameToUntitled(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	submitResponse := submitRun(t, workspace, jobs.SubmitRun{
		Tasks: []jobs.SubmitTask{{TaskKey: "main"}},
	})

	run := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, "Untitled", run.RunName)
}

func TestJobsSubmit_RunReachesTerminalStateOnPoll(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	submitResponse := submitRun(t, workspace, jobs.SubmitRun{
		Tasks: []jobs.SubmitTask{{TaskKey: "main"}},
	})

	// The generic `jobs submit` waiter polls the V1 run-level state: RUNNING first,
	// then TERMINATED/SUCCESS.
	first := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, jobs.RunLifeCycleStateRunning, first.State.LifeCycleState)

	second := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, jobs.RunLifeCycleStateTerminated, second.State.LifeCycleState)
	assert.Equal(t, jobs.RunResultStateSuccess, second.State.ResultState)
}

func createJob(t *testing.T, workspace *FakeWorkspace) int64 {
	t.Helper()
	body, err := json.Marshal(jobs.CreateJob{Name: "my-job"})
	require.NoError(t, err)

	response := workspace.JobsCreate(Request{Body: body})
	require.Equal(t, 0, response.StatusCode)
	return response.Body.(jobs.CreateResponse).JobId
}

func runNow(t *testing.T, workspace *FakeWorkspace, request jobs.RunNow) Response {
	t.Helper()
	body, err := json.Marshal(request)
	require.NoError(t, err)
	return workspace.JobsRunNow(Request{Body: body})
}

// The Jobs API does not free an idempotency_token when its run is deleted, so a
// later reuse errors instead of starting a fresh run. This is exactly why
// job_runs' DoDelete is a noop: deleting a run would tombstone the deterministic
// token and break re-running the same config after destroy or a value flip-back.
func TestJobsRunNow_IdempotencyTokenTombstonedAfterDelete(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	jobID := createJob(t, workspace)

	const token = "stable-token"

	first := runNow(t, workspace, jobs.RunNow{JobId: jobID, IdempotencyToken: token})
	require.Equal(t, 0, first.StatusCode)
	runID := first.Body.(jobs.RunNowResponse).RunId
	require.NotZero(t, runID)

	// A retry with the same token dedupes to the existing run (no duplicate).
	second := runNow(t, workspace, jobs.RunNow{JobId: jobID, IdempotencyToken: token})
	require.Equal(t, 0, second.StatusCode)
	assert.Equal(t, runID, second.Body.(jobs.RunNowResponse).RunId)

	body, err := json.Marshal(jobs.DeleteRun{RunId: runID})
	require.NoError(t, err)
	del := workspace.JobsDeleteRun(Request{Body: body})
	require.Equal(t, 0, del.StatusCode)

	// Reusing the tombstoned token now errors rather than starting a fresh run.
	third := runNow(t, workspace, jobs.RunNow{JobId: jobID, IdempotencyToken: token})
	assert.Equal(t, 400, third.StatusCode)
	assert.Contains(t, third.Body.(string), "has been deleted")
}

func TestJobsSubmit_RejectsInvalidGitProvider(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	body, err := json.Marshal(jobs.SubmitRun{
		GitSource: &jobs.GitSource{GitUrl: "https://example.com/repo"},
		Tasks:     []jobs.SubmitTask{{TaskKey: "main"}},
	})
	require.NoError(t, err)

	response := workspace.JobsSubmit(Request{Body: body})
	assert.Equal(t, 400, response.StatusCode)
	assert.Equal(t, missingJobGitProviderMessage, response.Body.(map[string]string)["message"])
}
