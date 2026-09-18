package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJobRunDependencies(t *testing.T) {
	tests := []struct {
		name       string
		dependency string
		summary    string
	}{
		{
			name:       "job run id reference",
			dependency: "${resources.job_runs.prepare.id}",
		},
		{
			name:       "literal name",
			dependency: "prepare",
			summary:    "depends_on entries must be job run ID references, for example ${resources.job_runs.prepare.id}",
		},
		{
			name:       "job definition reference",
			dependency: "${resources.jobs.prepare.id}",
			summary:    "depends_on entries must be job run ID references, for example ${resources.job_runs.prepare.id}",
		},
		{
			name:       "job run outcome reference",
			dependency: "${resources.job_runs.prepare.state.result_state}",
			summary:    "depends_on entries must be job run ID references, for example ${resources.job_runs.prepare.id}",
		},
		{
			name:       "reference with extra text",
			dependency: "run ${resources.job_runs.prepare.id}",
			summary:    "depends_on entries must be job run ID references, for example ${resources.job_runs.prepare.id}",
		},
		{
			name:       "undefined job run",
			dependency: "${resources.job_runs.missing.id}",
			summary:    `depends_on references undefined job run "missing"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						JobRuns: map[string]*resources.JobRun{
							"prepare": {},
							"publish": {DependsOn: []string{tt.dependency}},
						},
					},
				},
			}

			diags := ValidateJobRunDependencies().Apply(t.Context(), b)
			if tt.summary == "" {
				assert.Empty(t, diags)
				return
			}

			require.Len(t, diags, 1)
			assert.Equal(t, tt.summary, diags[0].Summary)
			require.Len(t, diags[0].Paths, 1)
			assert.Equal(t, "resources.job_runs.publish.depends_on[0]", diags[0].Paths[0].String())
		})
	}
}
