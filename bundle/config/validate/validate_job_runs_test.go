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
		// An empty `job_runs.<name>:` entry unmarshals to a nil pointer, which the
		// validator skips.
		"empty":     nil,
		"minimal":   {RunNow: jobs.RunNow{JobId: 1}},
		"bounded":   {RunNow: jobs.RunNow{JobId: 2}, RerunToken: "v2", Timeout: "90m"},
		"unwatched": {RunNow: jobs.RunNow{JobId: 3}, WaitForCompletion: &wait},
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

func TestValidateJobRunsRejectsUnusableTimeout(t *testing.T) {
	tests := []struct {
		name    string
		run     resources.JobRun
		summary string
	}{
		// The SDK waiter reads a zero timeout as its own 20 minute default and a
		// negative one as "poll forever", so neither may reach it.
		{"zero", resources.JobRun{Timeout: "0s"}, `timeout must be positive, got "0s"`},
		{"negative", resources.JobRun{Timeout: "-5m"}, `timeout must be positive, got "-5m"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := jobRunsBundle(map[string]*resources.JobRun{"my_run": &tt.run})

			diags := ValidateJobRuns().Apply(t.Context(), b)

			require.Len(t, diags, 1)
			assert.Equal(t, "resources.job_runs.my_run.timeout", diags[0].Paths[0].String())
			assert.Contains(t, diags[0].Summary, tt.summary)
		})
	}
}

func TestJobRunStateReference(t *testing.T) {
	tests := map[string]string{
		// The whole state object and any field within it are equally unknowable
		// while the run is in flight.
		"resources.job_runs.nightly.state":                  "nightly",
		"resources.job_runs.nightly.state.result_state":     "nightly",
		"resources.job_runs.nightly.state.life_cycle_state": "nightly",
		"resources.job_runs.nightly.run_page_url":           "",
		"resources.job_runs.nightly.id":                     "",
		"resources.jobs.nightly.state":                      "",
		"resources.job_runs.nightly":                        "",
		"var.state":                                         "",
		"workspace.root_path":                               "",
	}

	for ref, expected := range tests {
		t.Run(ref, func(t *testing.T) {
			name, ok := jobRunStateReference(ref)
			assert.Equal(t, expected != "", ok)
			assert.Equal(t, expected, name)
		})
	}
}
