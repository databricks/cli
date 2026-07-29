package dresources

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jobRunClientFor returns a client talking to server. Call it after the test
// registers its own handlers: first registration wins, so the defaults added here
// only fill the gaps.
func jobRunClientFor(t *testing.T, server *testserver.Server) *databricks.WorkspaceClient {
	t.Helper()
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return client
}

// jobRunServer returns a client whose runs/get is the given handler, so a wait can
// be driven without a real run.
func jobRunServer(t *testing.T, getRun testserver.HandlerFunc) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", getRun)
	return jobRunClientFor(t, server)
}

// The Jobs API reports the run page in the legacy fragment form; errors and
// progress lines carry the path form it converts to.
const (
	testRunPageURL  = "https://myworkspace.databricks.test/?o=900800700600#job/456/run/123"
	testRunPageLink = "run page: https://myworkspace.databricks.test/jobs/456/runs/123?o=900800700600"
)

// jobRunClient returns a client whose GetRun always reports the given run state.
func jobRunClient(t *testing.T, state *jobs.RunState) *databricks.WorkspaceClient {
	t.Helper()
	return jobRunServer(t, func(req testserver.Request) any {
		return jobs.Run{RunId: 123, JobId: 456, State: state, RunPageUrl: testRunPageURL}
	})
}

// waitForTestRun drives the framework hook, so it covers parsing the id the
// framework hands back from DoCreate along with the wait itself.
func waitForTestRun(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient) (*JobRunRemote, error) {
	t.Helper()
	r := (&ResourceJobRun{}).New(client)
	return r.WaitAfterCreate(ctx, "123", &JobRunState{})
}

func TestJobRunWaitSucceeds(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	})

	remote, err := waitForTestRun(t, t.Context(), client)

	require.NoError(t, err)
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
}

func TestJobRunWaitFailsOnFailedResult(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateFailed,
		StateMessage:   "task failed",
	})

	_, err := waitForTestRun(t, t.Context(), client)

	// Only SUCCESS completes the deploy; a FAILED result fails it.
	require.ErrorContains(t, err, "did not succeed: FAILED: task failed")
}

func TestJobRunWaitReportsFailedTask(t *testing.T) {
	failed := &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateFailed,
	}
	server := testserver.New(t)
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{
			RunId: 123,
			JobId: 456,
			State: failed,
			Tasks: []jobs.RunTask{
				{TaskKey: "ok", RunId: 998, State: &jobs.RunState{
					LifeCycleState: jobs.RunLifeCycleStateTerminated,
					ResultState:    jobs.RunResultStateSuccess,
				}},
				{TaskKey: "main", RunId: 999, State: failed},
			},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return jobs.RunOutput{Error: "notebook not found"}
	})

	_, err := waitForTestRun(t, t.Context(), jobRunClientFor(t, server))

	// The error names the failing task and the message it reported, and leaves out
	// the tasks that did not fail.
	require.ErrorContains(t, err, `task "main": notebook not found`)
	assert.NotContains(t, err.Error(), `task "ok"`)
}

func TestJobRunWaitFailsOnSkipped(t *testing.T) {
	// A skipped run has no result_state, so the lifecycle state is reported.
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateSkipped,
	})

	_, err := waitForTestRun(t, t.Context(), client)

	require.ErrorContains(t, err, "did not succeed: SKIPPED")
}

func TestJobRunWaitFailsOnInternalError(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateInternalError,
	})

	_, err := waitForTestRun(t, t.Context(), client)

	// The SDK waiter errors on INTERNAL_ERROR, ahead of the result check.
	require.ErrorContains(t, err, "INTERNAL_ERROR")
	require.ErrorContains(t, err, testRunPageLink)
}

func TestJobRunWaitAbandonedLinksTheRun(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning})

	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()

	_, err := waitForTestRun(t, ctx, client)

	// Giving up on the wait does not stop the run, so the error links to it.
	require.Error(t, err)
	require.ErrorContains(t, err, testRunPageLink)
}

// Reporting RUNNING for the first two polls exercises the poll loop; the other
// tests stub an already-terminal state.
func TestJobRunWaitPollsUntilTerminal(t *testing.T) {
	var gets atomic.Int32
	client := jobRunServer(t, func(req testserver.Request) any {
		if gets.Add(1) <= 2 {
			return jobs.Run{RunId: 123, JobId: 456, State: &jobs.RunState{
				LifeCycleState: jobs.RunLifeCycleStateRunning,
			}}
		}
		return jobs.Run{RunId: 123, JobId: 456, State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateTerminated,
			ResultState:    jobs.RunResultStateSuccess,
		}}
	})

	remote, err := waitForTestRun(t, t.Context(), client)
	require.NoError(t, err)

	// SUCCESS is only reachable by polling past the RUNNING reads.
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
	assert.Equal(t, int32(3), gets.Load(), "expected the wait to poll past both RUNNING reads")
}
