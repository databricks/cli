package dresources

import (
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyTokenIsStableHex(t *testing.T) {
	ctx := t.Context()
	run := jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "prod"}}

	got, err := idempotencyToken(ctx, &JobRunState{RunNow: run})
	require.NoError(t, err)

	// hex SHA-256 is always 64 lowercase hex chars (the Jobs API maximum).
	assert.Regexp(t, "^[0-9a-f]{64}$", got)

	// Deterministic: the same config yields the same token, so a retry dedupes.
	again, err := idempotencyToken(ctx, &JobRunState{RunNow: run})
	require.NoError(t, err)
	assert.Equal(t, got, again)
}

func TestIdempotencyTokenIgnoresPresetToken(t *testing.T) {
	ctx := t.Context()
	a, err := idempotencyToken(ctx, &JobRunState{RunNow: jobs.RunNow{JobId: 123}})
	require.NoError(t, err)
	b, err := idempotencyToken(ctx, &JobRunState{RunNow: jobs.RunNow{JobId: 123, IdempotencyToken: "user-supplied"}})
	require.NoError(t, err)

	// The token is cleared before hashing, so a preset value cannot change it.
	assert.Equal(t, a, b)
}

func TestIdempotencyTokenChangesWithConfig(t *testing.T) {
	ctx := t.Context()
	dev, err := idempotencyToken(ctx, &JobRunState{RunNow: jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "dev"}}})
	require.NoError(t, err)
	prod, err := idempotencyToken(ctx, &JobRunState{RunNow: jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "prod"}}})
	require.NoError(t, err)
	otherJob, err := idempotencyToken(ctx, &JobRunState{RunNow: jobs.RunNow{JobId: 456}})
	require.NoError(t, err)

	assert.NotEqual(t, dev, prod)     // different params --> different token
	assert.NotEqual(t, dev, otherJob) // different job_id --> different token
}

func TestIdempotencyTokenChangesWithResourceIdentity(t *testing.T) {
	run := jobs.RunNow{JobId: 123}
	a, err := idempotencyToken(WithResourceIdentity(t.Context(), "resources.job_runs.a"), &JobRunState{RunNow: run})
	require.NoError(t, err)
	b, err := idempotencyToken(WithResourceIdentity(t.Context(), "resources.job_runs.b"), &JobRunState{RunNow: run})
	require.NoError(t, err)

	// Identical config under different resource keys must not collapse onto one run.
	assert.NotEqual(t, a, b)
}

// jobRunClient returns a client whose GetRun always reports the given terminal
// run state, so WaitAfterCreate can be exercised without a real run.
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

func TestJobRunWaitAfterCreateFailsOnFailedResult(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateFailed,
		StateMessage:   "task failed",
	})

	r := (&ResourceJobRun{}).New(client)
	_, err := r.WaitAfterCreate(t.Context(), "123", &JobRunState{})

	// Only SUCCESS completes the deploy; a FAILED result fails it.
	require.ErrorContains(t, err, "did not succeed: FAILED: task failed")
}

func TestJobRunWaitAfterCreateFailsOnSkipped(t *testing.T) {
	// A skipped run has no result_state, so the lifecycle state is reported.
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateSkipped,
	})

	r := (&ResourceJobRun{}).New(client)
	_, err := r.WaitAfterCreate(t.Context(), "123", &JobRunState{})

	require.ErrorContains(t, err, "did not succeed: SKIPPED")
}

func TestJobRunWaitAfterCreateSucceeds(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateTerminated,
		ResultState:    jobs.RunResultStateSuccess,
	})

	r := (&ResourceJobRun{}).New(client)
	remote, err := r.WaitAfterCreate(t.Context(), "123", &JobRunState{})

	require.NoError(t, err)
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
}

func TestJobRunWaitAfterCreateFailsOnInternalError(t *testing.T) {
	client := jobRunClient(t, &jobs.RunState{
		LifeCycleState: jobs.RunLifeCycleStateInternalError,
	})

	r := (&ResourceJobRun{}).New(client)
	_, err := r.WaitAfterCreate(t.Context(), "123", &JobRunState{})

	// INTERNAL_ERROR is the one terminal state that fails the deploy.
	require.Error(t, err)
}

// Reports RUNNING before TERMINATED, so this is the only test that exercises the
// poll loop (the others stub an already-terminal state).
func TestJobRunWaitAfterCreatePollsUntilTerminal(t *testing.T) {
	server := testserver.New(t)

	// logRunPageURL does one GET before the loop, so RUNNING must cover it plus
	// the waiter's first poll.
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
	remote, err := r.WaitAfterCreate(t.Context(), "123", &JobRunState{})
	require.NoError(t, err)

	// SUCCESS is only reachable by polling past the RUNNING reads.
	require.NotNil(t, remote.State)
	assert.Equal(t, jobs.RunResultStateSuccess, remote.State.ResultState)
	assert.GreaterOrEqual(t, gets.Load(), int32(2), "expected WaitAfterCreate to poll more than once")
}
