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
