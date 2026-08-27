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

	fileChange := "seed.txt"
	emptyFile := ""
	whitespaceFile := "  \t"

	valueChange := "${resources.jobs.foo.id}"

	tests := []struct {
		name           string
		triggers       []resources.JobRunTrigger
		preventDestroy bool
		summary        string
	}{
		{
			name: "on_bundle_deploy true",
			triggers: []resources.JobRunTrigger{
				{OnBundleDeploy: &trueVal},
			},
		},
		{
			name: "on_file_change set",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &fileChange},
			},
		},
		{
			name: "both triggers as separate entries",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &fileChange},
				{OnBundleDeploy: &trueVal},
				{OnValueChange: &valueChange},
			},
		},
		{
			name: "empty entry",
			triggers: []resources.JobRunTrigger{
				{},
			},
			summary: "lifecycle.triggers entry must set on_bundle_deploy, on_file_change, or on_value_change",
		},
		{
			name: "both keys on one entry",
			triggers: []resources.JobRunTrigger{
				{OnBundleDeploy: &trueVal, OnFileChange: &fileChange},
			},
			summary: "lifecycle.triggers entry must set only one of on_bundle_deploy, on_file_change, or on_value_change",
		},
		{
			name: "on_bundle_deploy false",
			triggers: []resources.JobRunTrigger{
				{OnBundleDeploy: &falseVal},
			},
			summary: "lifecycle.triggers.on_bundle_deploy must be true when set",
		},
		{
			name: "on_file_change empty",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &emptyFile},
			},
			summary: "lifecycle.triggers.on_file_change must be non-empty when set",
		},
		{
			name: "on_file_change whitespace",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &whitespaceFile},
			},
			summary: "lifecycle.triggers.on_file_change must be non-empty when set",
		},
		{
			name: "on_bundle_deploy with prevent_destroy",
			triggers: []resources.JobRunTrigger{
				{OnBundleDeploy: &trueVal},
			},
			preventDestroy: true,
			summary:        "lifecycle.triggers.on_bundle_deploy is incompatible with lifecycle.prevent_destroy",
		},
		{
			name: "on_file_change with prevent_destroy",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &fileChange},
			},
			preventDestroy: true,
			summary:        "lifecycle.triggers.on_file_change is incompatible with lifecycle.prevent_destroy",
		},
		{
			name: "both triggers with prevent_destroy",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &fileChange},
				{OnBundleDeploy: &trueVal},
			},
			preventDestroy: true,
			summary:        "lifecycle.triggers.on_bundle_deploy and on_file_change are incompatible with lifecycle.prevent_destroy",
		},
		{
			name: "all triggers with prevent_destroy",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &fileChange},
				{OnBundleDeploy: &trueVal},
				{OnValueChange: &valueChange},
			},
			preventDestroy: true,
			summary:        "lifecycle.triggers.on_bundle_deploy, on_file_change and on_value_change are incompatible with lifecycle.prevent_destroy",
		},
		{
			name:           "prevent_destroy alone",
			preventDestroy: true,
		},
		{
			name: "on_value_change set",
			triggers: []resources.JobRunTrigger{
				{OnValueChange: &valueChange},
			},
		},
		{
			name: "on_value_change with prevent_destroy",
			triggers: []resources.JobRunTrigger{
				{OnValueChange: &valueChange},
			},
			preventDestroy: true,
			summary:        "lifecycle.triggers.on_value_change is incompatible with lifecycle.prevent_destroy",
		},
		{
			name: "on_value_change and on_file_change on one entry",
			triggers: []resources.JobRunTrigger{
				{OnFileChange: &fileChange, OnValueChange: &valueChange},
			},
			summary: "lifecycle.triggers entry must set only one of on_bundle_deploy, on_file_change, or on_value_change",
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
									Lifecycle: resources.Lifecycle{PreventDestroy: tt.preventDestroy},
									Triggers:  tt.triggers,
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
