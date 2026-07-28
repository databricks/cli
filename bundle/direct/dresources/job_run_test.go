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

// testIdentity is what the framework attaches on a first create.
var testIdentity = CreateIdentity{
	Deployment:  "/Workspace/Users/me/.bundle/mybundle/default",
	ResourceKey: "resources.job_runs.nightly",
}

func tokenFor(t *testing.T, identity CreateIdentity, state *JobRunState) string {
	t.Helper()
	token, err := idempotencyToken(identity, state)
	require.NoError(t, err)
	return token
}

func TestIdempotencyTokenIsStableHex(t *testing.T) {
	run := jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "prod"}}

	got := tokenFor(t, testIdentity, &JobRunState{RunNow: run})

	// hex SHA-256 is always 64 lowercase hex chars (the Jobs API maximum).
	assert.Regexp(t, "^[0-9a-f]{64}$", got)

	// Deterministic: the same config yields the same token, so a retry dedupes.
	assert.Equal(t, got, tokenFor(t, testIdentity, &JobRunState{RunNow: run}))
}

func TestIdempotencyTokenIgnoresPresetToken(t *testing.T) {
	base := tokenFor(t, testIdentity, &JobRunState{RunNow: jobs.RunNow{JobId: 123}})
	preset := tokenFor(t, testIdentity, &JobRunState{RunNow: jobs.RunNow{JobId: 123, IdempotencyToken: "user-supplied"}})

	// The token is cleared before hashing, so a preset value cannot change it.
	assert.Equal(t, base, preset)
}

func TestIdempotencyTokenChangesWithConfig(t *testing.T) {
	dev := tokenFor(t, testIdentity, &JobRunState{RunNow: jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "dev"}}})
	prod := tokenFor(t, testIdentity, &JobRunState{RunNow: jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "prod"}}})
	otherJob := tokenFor(t, testIdentity, &JobRunState{RunNow: jobs.RunNow{JobId: 456}})

	assert.NotEqual(t, dev, prod)     // different params --> different token
	assert.NotEqual(t, dev, otherJob) // different job_id --> different token
}

func TestIdempotencyTokenChangesWithRerunToken(t *testing.T) {
	run := jobs.RunNow{JobId: 123}

	base := tokenFor(t, testIdentity, &JobRunState{RunNow: run})
	bumped := tokenFor(t, testIdentity, &JobRunState{RunNow: run, RerunToken: "v2"})
	bumpedAgain := tokenFor(t, testIdentity, &JobRunState{RunNow: run, RerunToken: "v2"})

	assert.NotEqual(t, base, bumped)     // changing rerun_token forces a new run
	assert.Equal(t, bumped, bumpedAgain) // the same rerun_token value stays stable
}

func TestIdempotencyTokenChangesWithIdentity(t *testing.T) {
	state := &JobRunState{RunNow: jobs.RunNow{JobId: 123}}

	base := tokenFor(t, testIdentity, state)

	otherKey := testIdentity
	otherKey.ResourceKey = "resources.job_runs.other"
	otherDeployment := testIdentity
	otherDeployment.Deployment = "/Workspace/Users/me/.bundle/mybundle/prod"

	// The identity keeps identical config apart across resource keys and across
	// deployments sharing a workspace.
	assert.NotEqual(t, base, tokenFor(t, otherKey, state))
	assert.NotEqual(t, base, tokenFor(t, otherDeployment, state))
}

func TestIdempotencyTokenRotatesWithPriorID(t *testing.T) {
	state := &JobRunState{RunNow: jobs.RunNow{JobId: 123}}
	recreate := testIdentity
	recreate.PriorID = "555"

	base := tokenFor(t, testIdentity, state)
	rotated := tokenFor(t, recreate, state)

	// Re-creating a vanished run rotates the token off the tombstoned one...
	assert.NotEqual(t, base, rotated)
	// ...but stays deterministic so a retry dedupes onto the fresh run.
	assert.Equal(t, rotated, tokenFor(t, recreate, state))
}

// jobRunServer returns a test server whose runs/get handler is the given one,
// so a wait can be exercised without a real run.
func jobRunServer(t *testing.T, getRun testserver.HandlerFunc) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	// First registration wins, so this overrides the default runs/get handler.
	server.Handle("GET", "/api/2.2/jobs/runs/get", getRun)
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return client
}

// jobRunClient returns a client whose GetRun always reports the given terminal
// run state.
func jobRunClient(t *testing.T, state *jobs.RunState) *databricks.WorkspaceClient {
	t.Helper()
	return jobRunServer(t, func(req testserver.Request) any {
		return jobs.Run{RunId: 123, JobId: 456, State: state}
	})
}

func waitForTestRun(t *testing.T, client *databricks.WorkspaceClient) (*JobRunRemote, error) {
	t.Helper()
	r := (&ResourceJobRun{}).New(client)
	return r.waitForRun(t.Context(), "resources.job_runs.nightly", 123)
}

func TestJobRunCreateRequiresIdentity(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated})
	r := (&ResourceJobRun{}).New(client)

	// The framework always sets one, so its absence is a wiring bug and errors.
	_, _, err := r.DoCreate(t.Context(), &JobRunState{RunNow: jobs.RunNow{JobId: 123}})
	require.ErrorContains(t, err, "without a create identity")
}

func TestJobRunWaitFailsOnFailedResult(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateFailed,
		StateMessage:   "task failed",
	})

	_, err := waitForTestRun(t, client)

	// Only SUCCESS completes the deploy; a FAILED result fails it.
	require.ErrorContains(t, err, "did not succeed: FAILED: task failed")
	// The state is not saved, so the next deploy re-issues run-now with the same
	// token and lands back on this run. Point at the way out.
	require.ErrorContains(t, err, "set rerun_token to a new value")
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
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	_, err = waitForTestRun(t, client)

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

	_, err := waitForTestRun(t, client)

	require.ErrorContains(t, err, "did not succeed: SKIPPED")
}

func TestJobRunWaitSucceeds(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	})

	remote, err := waitForTestRun(t, client)

	require.NoError(t, err)
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
}

func TestJobRunWaitFailsOnInternalError(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateInternalError,
	})

	_, err := waitForTestRun(t, client)

	// The SDK waiter errors on INTERNAL_ERROR, ahead of the result check, so the
	// wrapping is all that names the run.
	require.ErrorContains(t, err, "waiting for job run 123")
	require.ErrorContains(t, err, "INTERNAL_ERROR")
}

func TestJobRunWaitAbandonedNamesTheRun(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning})

	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	r := (&ResourceJobRun{}).New(client)
	_, err := r.waitForRun(ctx, "resources.job_runs.nightly", 123)

	// Giving up on the wait does not stop the run, so the error has to name it.
	require.ErrorContains(t, err, "waiting for job run 123")
}

// Reports RUNNING before TERMINATED, exercising the poll loop (the other tests
// stub an already-terminal state).
func TestJobRunWaitPollsUntilTerminal(t *testing.T) {
	// Returning RUNNING for the first two polls makes the waiter iterate a few
	// times, calling the progress callback on each, before it sees TERMINATED.
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

	remote, err := waitForTestRun(t, client)
	require.NoError(t, err)

	// SUCCESS is only reachable by polling past the RUNNING reads.
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
	assert.GreaterOrEqual(t, gets.Load(), int32(2), "expected waitForRun to poll more than once")
}
