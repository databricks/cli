package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestValidateJobRunTriggers(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		triggers []resources.JobRunTrigger
		summary  string
	}{
		{
			name: "on_bundle_deploy true",
			triggers: []resources.JobRunTrigger{
				{OnBundleDeploy: &trueVal},
			},
		},
		{
			name: "empty entry",
			triggers: []resources.JobRunTrigger{
				{},
			},
			summary: "lifecycle.triggers entry must set exactly one trigger mode",
		},
		{
			name: "two modes",
			triggers: []resources.JobRunTrigger{
				{OnBundleDeploy: &trueVal, OnFileChange: "src/**/*.py"},
			},
			summary: "lifecycle.triggers entry must set exactly one trigger mode",
		},
		{
			name: "on_bundle_deploy false",
			triggers: []resources.JobRunTrigger{
				{OnBundleDeploy: &falseVal},
			},
			summary: "lifecycle.triggers.on_bundle_deploy must be true when set",
		},
		{
			name: "on_file_change unsupported",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: "src/**/*.py"},
			},
			summary: "lifecycle.triggers.on_file_change is not supported yet",
		},
		{
			name: "on_value_change unsupported",
			triggers: []resources.JobRunTrigger{
				{OnValueChange: "${resources.jobs.foo.id}"},
			},
			summary: "lifecycle.triggers.on_value_change is not supported yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						JobRuns: map[string]*resources.JobRun{
							"my_run": {
								Lifecycle: &resources.JobRunLifecycle{
									Triggers: tt.triggers,
								},
							},
						},
					},
				},
			}
			diags := bundle.Apply(t.Context(), b, mutator.ValidateJobRunTriggers())
			if tt.summary == "" {
				assert.Empty(t, diags)
				return
			}
			assert.True(t, diags.HasError())
			assert.Equal(t, tt.summary, diags[0].Summary)
		})
	}
}
