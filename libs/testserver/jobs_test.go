package testserver

import (
	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func submitRun(t *testing.T, workspace *FakeWorkspace, request jobs.SubmitRunRequest) jobs.SubmitRunResponse {
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

	submitResponse := submitRun(t, workspace, jobs.SubmitRunRequest{
		RunName: "ssh-tunnel",
		Tasks: []jobs.RunTaskSettings{
			{TaskKey: "main", EnvironmentKey: "default"},
		},
	})
	require.NotZero(t, submitResponse.RunId)

	run := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, "ssh-tunnel", run.RunName)
	assert.Equal(t, jobs.RunType_SubmitRun, run.RunType)

	require.Len(t, run.Tasks, 1)
	task := run.Tasks[0]
	assert.Equal(t, "main", task.TaskKey)
	// ssh connect's waitForJobToStart polls the V2 per-task status.
	require.NotNil(t, task.Status)
	assert.Equal(t, jobs.RunLifecycleStateV2_State_Running, task.Status.State)
}

func TestJobsSubmit_DefaultsRunNameToUntitled(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	submitResponse := submitRun(t, workspace, jobs.SubmitRunRequest{
		Tasks: []jobs.RunTaskSettings{{TaskKey: "main"}},
	})

	run := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, "Untitled", run.RunName)
}

func TestJobsSubmit_RunReachesTerminalStateOnPoll(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	submitResponse := submitRun(t, workspace, jobs.SubmitRunRequest{
		Tasks: []jobs.RunTaskSettings{{TaskKey: "main"}},
	})

	// The generic `jobs submit` waiter polls the V1 run-level state: RUNNING first,
	// then TERMINATED/SUCCESS.
	first := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, jobs.RunLifeCycleState_RunLifeCycleState_Running, first.State.LifeCycleState)

	second := getRun(t, workspace, submitResponse.RunId)
	assert.Equal(t, jobs.RunLifeCycleState_RunLifeCycleState_Terminated, second.State.LifeCycleState)
	assert.Equal(t, jobs.RunResultState_RunResultState_Success, second.State.ResultState)
}

func createJob(t *testing.T, workspace *FakeWorkspace, tasks ...jobs.TaskSettings) int64 {
	t.Helper()
	body, err := json.Marshal(jobs.CreateJobRequest{Name: "my-job", Tasks: tasks})
	require.NoError(t, err)

	response := workspace.JobsCreate(Request{Body: body})
	require.Equal(t, 0, response.StatusCode)
	return response.Body.(jobs.CreateJobResponse).JobId
}

func runNow(t *testing.T, workspace *FakeWorkspace, request jobs.RunNowRequest) Response {
	t.Helper()
	body, err := json.Marshal(request)
	require.NoError(t, err)
	return workspace.JobsRunNow(Request{Body: body})
}

func terminatedTask(taskKey string, result jobs.RunResultState_RunResultState) jobs.RunTask {
	return jobs.RunTask{
		TaskKey: taskKey,
		State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleState_RunLifeCycleState_Terminated,
			ResultState:    result,
		},
	}
}

func TestTerminateRun_FailedTaskFailsTheRun(t *testing.T) {
	run := jobs.Run{Tasks: []jobs.RunTask{
		terminatedTask("first", jobs.RunResultState_RunResultState_Success),
		terminatedTask("second", jobs.RunResultState_RunResultState_Failed),
	}}

	terminateRun(&run)

	// A failed run reports INTERNAL_ERROR, not TERMINATED, in life_cycle_state.
	assert.Equal(t, jobs.RunLifeCycleState_RunLifeCycleState_InternalError, run.State.LifeCycleState)
	assert.Equal(t, jobs.RunResultState_RunResultState_Failed, run.State.ResultState)
	assert.Equal(t, "Task second failed with message: Workload failed, see run output for details.", run.State.StateMessage)
}

func TestNewTaskFailure_ReportsTheExceptionAndTheTraceback(t *testing.T) {
	output := "Traceback (most recent call last):\n  File \"fail.py\", line 1\nRuntimeError: intentional failure\n"

	failure := newTaskFailure(errors.New("exit status 1"), output)

	assert.Equal(t, "RuntimeError: intentional failure", failure.Error())
	assert.Equal(t, strings.TrimRight(output, "\n"), failure.trace)
}

func TestNewTaskFailure_SingleLineOutputIsTheException(t *testing.T) {
	failure := newTaskFailure(errors.New("exit status 1"), "RuntimeError: intentional failure\n")

	assert.Equal(t, "RuntimeError: intentional failure", failure.Error())
}

// A task can exit non-zero without writing anything, e.g. sys.exit(1).
func TestNewTaskFailure_SilentFailureIsReportedAsTheExitError(t *testing.T) {
	failure := newTaskFailure(errors.New("exit status 1"), "")

	assert.Equal(t, "exit status 1", failure.Error())
}

func TestTerminateRun_CompletesTasksThatAreStillRunning(t *testing.T) {
	// jobs/runs/submit records its tasks as running: they are never executed.
	run := jobs.Run{Tasks: []jobs.RunTask{
		{TaskKey: "main", State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleState_RunLifeCycleState_Running}},
	}}

	terminateRun(&run)

	assert.Equal(t, jobs.RunResultState_RunResultState_Success, run.State.ResultState)
	assert.Empty(t, run.State.StateMessage)
	assert.Equal(t, jobs.RunResultState_RunResultState_Success, run.Tasks[0].State.ResultState)
}

// See errNoCodeInWorkspace: a missing notebook is this server's gap, not a
// failure of the job.
func TestJobsGetRun_TaskWithoutCodeDoesNotFailTheRun(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	jobID := createJob(t, workspace, jobs.TaskSettings{
		TaskKey:      "main",
		NotebookTask: &jobs.NotebookTask{NotebookPath: "/missing-notebook"},
	})

	response := runNow(t, workspace, jobs.RunNowRequest{JobId: jobID})
	require.Equal(t, 0, response.StatusCode)
	runID := response.Body.(jobs.RunNowResponse).RunId

	// The first poll reports RUNNING, the second the terminal state.
	require.Equal(t, jobs.RunLifeCycleState_RunLifeCycleState_Running, getRun(t, workspace, runID).State.LifeCycleState)
	assert.Equal(t, jobs.RunResultState_RunResultState_Success, getRun(t, workspace, runID).State.ResultState)
}

func TestJobsRunNow_RepeatedTokenReturnsTheSameRun(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	jobID := createJob(t, workspace, jobs.TaskSettings{TaskKey: "main"})
	request := jobs.RunNowRequest{JobId: jobID, IdempotencyToken: "token"}

	first := runNow(t, workspace, request).Body.(jobs.RunNowResponse).RunId
	second := runNow(t, workspace, request).Body.(jobs.RunNowResponse).RunId

	assert.Equal(t, first, second)
	assert.Len(t, workspace.JobRuns, 1)
}

// Requests without a token are never deduped (`bundle run`, `jobs run-now`).
func TestJobsRunNow_WithoutATokenIsNeverDeduped(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	jobID := createJob(t, workspace, jobs.TaskSettings{TaskKey: "main"})

	first := runNow(t, workspace, jobs.RunNowRequest{JobId: jobID}).Body.(jobs.RunNowResponse).RunId
	second := runNow(t, workspace, jobs.RunNowRequest{JobId: jobID}).Body.(jobs.RunNowResponse).RunId

	assert.NotEqual(t, first, second)
	assert.Len(t, workspace.JobRuns, 2)
}

// An interrupted deploy leaves the run going, and the CLI cancels it before
// deleting it, since jobs/runs/delete rejects an active run.
func TestJobsCancelRun_SettlesAnActiveRun(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	jobID := createJob(t, workspace, jobs.TaskSettings{
		TaskKey:      "main",
		NotebookTask: &jobs.NotebookTask{NotebookPath: "/missing-notebook"},
	})
	runID := runNow(t, workspace, jobs.RunNowRequest{JobId: jobID}).Body.(jobs.RunNowResponse).RunId

	body, err := json.Marshal(jobs.CancelRunRequest{RunId: runID})
	require.NoError(t, err)
	require.Equal(t, 0, workspace.JobsCancelRun(Request{Body: body}).StatusCode)

	run := getRun(t, workspace, runID)
	assert.Equal(t, jobs.RunLifeCycleState_RunLifeCycleState_Terminated, run.State.LifeCycleState)
	assert.Equal(t, jobs.RunResultState_RunResultState_Canceled, run.State.ResultState)
}

func TestJobsCancelRun_LeavesAFinishedRunAlone(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	jobID := createJob(t, workspace, jobs.TaskSettings{
		TaskKey:      "main",
		NotebookTask: &jobs.NotebookTask{NotebookPath: "/missing-notebook"},
	})
	runID := runNow(t, workspace, jobs.RunNowRequest{JobId: jobID}).Body.(jobs.RunNowResponse).RunId

	// The first poll settles the run, the way a completed deploy leaves it.
	getRun(t, workspace, runID)
	require.Equal(t, jobs.RunResultState_RunResultState_Success, getRun(t, workspace, runID).State.ResultState)

	body, err := json.Marshal(jobs.CancelRunRequest{RunId: runID})
	require.NoError(t, err)
	require.Equal(t, 0, workspace.JobsCancelRun(Request{Body: body}).StatusCode)

	assert.Equal(t, jobs.RunResultState_RunResultState_Success, getRun(t, workspace, runID).State.ResultState)
}

func TestJobsSubmit_RejectsInvalidGitProvider(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	body, err := json.Marshal(jobs.SubmitRunRequest{
		GitSource: &jobs.GitSource{GitUrl: "https://example.com/repo"},
		Tasks:     []jobs.RunTaskSettings{{TaskKey: "main"}},
	})
	require.NoError(t, err)

	response := workspace.JobsSubmit(Request{Body: body})
	assert.Equal(t, 400, response.StatusCode)
	assert.Equal(t, missingJobGitProviderMessage, response.Body.(map[string]string)["message"])
}
