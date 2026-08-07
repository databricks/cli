package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jobRunsBundle(runs map[string]*resources.JobRun) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{JobRuns: runs},
		},
	}
}

func TestValidateJobRunIdempotencyTokenRejectsAConfiguredToken(t *testing.T) {
	b := jobRunsBundle(map[string]*resources.JobRun{
		"my_run": {RunNow: jobs.RunNow{JobId: 456, IdempotencyToken: "mine"}},
	})

	diags := ValidateJobRunIdempotencyToken().Apply(t.Context(), b)

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, "idempotency_token must not be set in bundle configuration; the CLI sets it on each run-now request", diags[0].Summary)
	assert.Equal(t, "resources.job_runs.my_run.idempotency_token", diags[0].Paths[0].String())
}

func TestValidateJobRunIdempotencyTokenAcceptsRunsWithoutOne(t *testing.T) {
	b := jobRunsBundle(map[string]*resources.JobRun{
		"my_run": {RunNow: jobs.RunNow{JobId: 456}},
		// An empty `job_runs.<name>:` entry loads as a present key with a nil value.
		"empty": nil,
	})

	assert.Empty(t, ValidateJobRunIdempotencyToken().Apply(t.Context(), b))
}
