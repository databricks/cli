package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyTokenIsStableHex(t *testing.T) {
	run := jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "prod"}}

	got, err := idempotencyToken(run)
	require.NoError(t, err)

	// hex SHA-256 is always 64 lowercase hex chars (the Jobs API maximum).
	assert.Regexp(t, "^[0-9a-f]{64}$", got)

	// Deterministic: the same config yields the same token, so a retry dedupes.
	again, err := idempotencyToken(run)
	require.NoError(t, err)
	assert.Equal(t, got, again)
}

func TestIdempotencyTokenIgnoresPresetToken(t *testing.T) {
	a, err := idempotencyToken(jobs.RunNow{JobId: 123})
	require.NoError(t, err)
	b, err := idempotencyToken(jobs.RunNow{JobId: 123, IdempotencyToken: "user-supplied"})
	require.NoError(t, err)

	// The token is cleared before hashing, so a preset value cannot change it.
	assert.Equal(t, a, b)
}

func TestIdempotencyTokenChangesWithConfig(t *testing.T) {
	dev, err := idempotencyToken(jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "dev"}})
	require.NoError(t, err)
	prod, err := idempotencyToken(jobs.RunNow{JobId: 123, JobParameters: map[string]string{"env": "prod"}})
	require.NoError(t, err)
	otherJob, err := idempotencyToken(jobs.RunNow{JobId: 456})
	require.NoError(t, err)

	assert.NotEqual(t, dev, prod)     // different params --> different token
	assert.NotEqual(t, dev, otherJob) // different job_id --> different token
}

func TestJobRunIdempotentCreate(t *testing.T) {
	_, client := setupTestServerClient(t)
	ctx := t.Context()

	// A run can only target an existing job, so create one first.
	job, err := client.Jobs.Create(ctx, jobs.CreateJob{
		Name: "idempotency-job",
		Tasks: []jobs.Task{{
			TaskKey:      "t",
			NotebookTask: &jobs.NotebookTask{NotebookPath: "/Workspace/Users/user@example.com/notebook"},
		}},
	})
	require.NoError(t, err)

	r := (&ResourceJobRun{}).New(client)
	config := &JobRunState{RunNow: jobs.RunNow{JobId: job.JobId}}

	// Same config twice: the derived token is identical, so the backend returns
	// the existing run instead of creating a duplicate.
	id1, _, err := r.DoCreate(ctx, config)
	require.NoError(t, err)
	id2, _, err := r.DoCreate(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "same config must dedupe to the same run")

	// Different config: different token, so a genuinely new run is created.
	other := &JobRunState{RunNow: jobs.RunNow{JobId: job.JobId, JobParameters: map[string]string{"env": "prod"}}}
	id3, _, err := r.DoCreate(ctx, other)
	require.NoError(t, err)
	assert.NotEqual(t, id1, id3, "different config must create a new run")
}
