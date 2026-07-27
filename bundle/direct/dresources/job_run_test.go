package dresources

import (
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
	token, err := idempotencyToken(WithCreateIdentity(t.Context(), identity), state)
	require.NoError(t, err)
	return token
}

func TestIdempotencyTokenRequiresIdentity(t *testing.T) {
	// The framework always sets one, so its absence is a wiring bug, not a case
	// to silently fall back from.
	_, err := idempotencyToken(t.Context(), &JobRunState{RunNow: jobs.RunNow{JobId: 123}})
	require.ErrorContains(t, err, "without a create identity")
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

func TestIdempotencyTokenIgnoresDeployOptions(t *testing.T) {
	run := jobs.RunNow{JobId: 123}
	wait := false

	base := tokenFor(t, testIdentity, &JobRunState{RunNow: run})
	tuned := tokenFor(t, testIdentity, &JobRunState{RunNow: run, WaitForCompletion: &wait, Timeout: "5m"})

	// Neither is run identity, so changing them must not re-trigger the run.
	assert.Equal(t, base, tuned)
}

func TestIdempotencyTokenChangesWithIdentity(t *testing.T) {
	state := &JobRunState{RunNow: jobs.RunNow{JobId: 123}}

	base := tokenFor(t, testIdentity, state)

	otherKey := testIdentity
	otherKey.ResourceKey = "resources.job_runs.other"
	otherDeployment := testIdentity
	otherDeployment.Deployment = "/Workspace/Users/me/.bundle/mybundle/prod"

	// Identical config must not collapse onto one run across resource keys or
	// across deployments sharing a workspace.
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

// jobRunClient returns a client whose GetRun always reports the given terminal
// run state, so waitForRun can be exercised without a real run.
func jobRunClient(t *testing.T, state *jobs.RunState) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	// First registration wins, so this overrides the default runs/get handler.
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return jobs.Run{RunId: 123, JobId: 456, State: state}
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return client
}

func TestJobRunWaitFailsOnFailedResult(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateFailed,
		StateMessage:   "task failed",
	})

	r := (&ResourceJobRun{}).New(client)
	_, err := r.waitForRun(t.Context(), 123, time.Minute)

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
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	r := (&ResourceJobRun{}).New(client)
	_, err = r.waitForRun(t.Context(), 123, time.Minute)

	// The failing task and its error are named; the task that succeeded is not.
	require.ErrorContains(t, err, `task "main": notebook not found`)
	assert.NotContains(t, err.Error(), `task "ok"`)
}

func TestJobRunWaitFailsOnSkipped(t *testing.T) {
	// A skipped run has no result_state, so the lifecycle state is reported.
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateSkipped,
	})

	r := (&ResourceJobRun{}).New(client)
	_, err := r.waitForRun(t.Context(), 123, time.Minute)

	require.ErrorContains(t, err, "did not succeed: SKIPPED")
}

func TestJobRunWaitSucceeds(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	})

	r := (&ResourceJobRun{}).New(client)
	remote, err := r.waitForRun(t.Context(), 123, time.Minute)

	require.NoError(t, err)
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
}

func TestJobRunWaitFailsOnInternalError(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateInternalError,
	})

	r := (&ResourceJobRun{}).New(client)
	_, err := r.waitForRun(t.Context(), 123, time.Minute)

	// INTERNAL_ERROR is the one terminal state that fails the deploy.
	require.Error(t, err)
}

// Reports RUNNING before TERMINATED, so this is the only test that exercises the
// poll loop (the others stub an already-terminal state).
func TestJobRunWaitPollsUntilTerminal(t *testing.T) {
	server := testserver.New(t)

	// waitForRun's progress callback fires on every poll (there is no separate
	// GET before the loop), so returning RUNNING for the first two polls just
	// makes the waiter iterate more than once before it sees TERMINATED.
	var gets atomic.Int32
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
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
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	r := (&ResourceJobRun{}).New(client)
	remote, err := r.waitForRun(t.Context(), 123, time.Minute)
	require.NoError(t, err)

	// SUCCESS is only reachable by polling past the RUNNING reads.
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
	assert.GreaterOrEqual(t, gets.Load(), int32(2), "expected waitForRun to poll more than once")
}
