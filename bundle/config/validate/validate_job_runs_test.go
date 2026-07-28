package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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

func TestValidateJobRunsAllowsValidConfig(t *testing.T) {
	b := jobRunsBundle(map[string]*resources.JobRun{
		// An empty `job_runs.<name>:` entry unmarshals to a nil pointer, which the
		// validator skips.
		"empty":   nil,
		"minimal": {RunNow: jobs.RunNow{JobId: 1}},
		"rerun":   {RunNow: jobs.RunNow{JobId: 2}, RerunToken: "v2"},
	})

	require.Empty(t, ValidateJobRuns().Apply(t.Context(), b))
}

func TestValidateJobRunsRejectsIdempotencyToken(t *testing.T) {
	b := jobRunsBundle(map[string]*resources.JobRun{
		"b_run": {RunNow: jobs.RunNow{JobId: 1, IdempotencyToken: "x"}},
		"a_run": {RunNow: jobs.RunNow{JobId: 2, IdempotencyToken: "y"}},
		"ok":    {RunNow: jobs.RunNow{JobId: 3}},
	})

	diags := ValidateJobRuns().Apply(t.Context(), b)

	require.Len(t, diags, 2)
	// Sorted by name, so a_run comes before b_run regardless of map order.
	assert.Equal(t, "resources.job_runs.a_run.idempotency_token", diags[0].Paths[0].String())
	assert.Equal(t, "resources.job_runs.b_run.idempotency_token", diags[1].Paths[0].String())
	assert.Contains(t, diags[0].Summary, "idempotency_token is computed automatically")
}
