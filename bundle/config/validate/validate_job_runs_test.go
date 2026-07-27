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
	wait := false
	b := jobRunsBundle(map[string]*resources.JobRun{
		// An empty `job_runs.<name>:` entry unmarshals to a nil pointer; the
		// validator must skip it rather than panic dereferencing.
		"empty":   nil,
		"minimal": {RunNow: jobs.RunNow{JobId: 1}},
		"tuned":   {RunNow: jobs.RunNow{JobId: 2}, RerunToken: "v2", WaitForCompletion: &wait, Timeout: "90m"},
	})

	require.Empty(t, ValidateJobRuns().Apply(t.Context(), b))
}

func TestValidateJobRunsRejectsBadFields(t *testing.T) {
	b := jobRunsBundle(map[string]*resources.JobRun{
		"b_run": {RunNow: jobs.RunNow{JobId: 1, IdempotencyToken: "x"}},
		"a_run": {RunNow: jobs.RunNow{JobId: 2}, Timeout: "soon"},
	})

	diags := ValidateJobRuns().Apply(t.Context(), b)

	require.Len(t, diags, 2)
	// Sorted by path, so a_run comes before b_run regardless of map order.
	assert.Equal(t, "resources.job_runs.a_run.timeout", diags[0].Paths[0].String())
	assert.Contains(t, diags[0].Summary, `timeout must be a duration such as "30m" or "2h"`)
	assert.Equal(t, "resources.job_runs.b_run.idempotency_token", diags[1].Paths[0].String())
	assert.Contains(t, diags[1].Summary, "idempotency_token is computed automatically")
}
