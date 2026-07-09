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

func jobRunBundle(token string) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				JobRuns: map[string]*resources.JobRun{
					"my_run": {RunNow: jobs.RunNow{JobId: 1, IdempotencyToken: token}},
				},
			},
		},
	}
}

func TestValidateJobRunIdempotencyTokenRejectsUserValue(t *testing.T) {
	diags := ValidateJobRunIdempotencyToken().Apply(t.Context(), jobRunBundle("x"))
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, "idempotency_token is computed automatically and must not be set in bundle configuration", diags[0].Summary)
	assert.Equal(t, "resources.job_runs.my_run.idempotency_token", diags[0].Paths[0].String())
}

func TestValidateJobRunIdempotencyTokenAllowsUnset(t *testing.T) {
	diags := ValidateJobRunIdempotencyToken().Apply(t.Context(), jobRunBundle(""))
	require.Empty(t, diags)
}

func TestValidateJobRunIdempotencyTokenReportsAllSorted(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				JobRuns: map[string]*resources.JobRun{
					"b_run": {RunNow: jobs.RunNow{JobId: 1, IdempotencyToken: "x"}},
					"a_run": {RunNow: jobs.RunNow{JobId: 2, IdempotencyToken: "y"}},
				},
			},
		},
	}

	diags := ValidateJobRunIdempotencyToken().Apply(t.Context(), b)
	require.Len(t, diags, 2)
	// Sorted by path, so a_run comes before b_run regardless of map order.
	assert.Equal(t, "resources.job_runs.a_run.idempotency_token", diags[0].Paths[0].String())
	assert.Equal(t, "resources.job_runs.b_run.idempotency_token", diags[1].Paths[0].String())
}
