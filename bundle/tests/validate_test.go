package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/stretchr/testify/assert"
)

func TestValidateUniqueResourceIdentifiers(t *testing.T) {
	tcases := []struct {
		name     string
		errorMsg string
	}{
		// {
		// 	name:     "duplicate_resource_names_in_root_job_and_pipeline",
		// 	errorMsg: "multiple resources named foo (jobs.foo at validate/duplicate_resource_names_in_root_job_and_pipeline/databricks.yml:10:7, pipelines.foo at validate/duplicate_resource_names_in_root_job_and_pipeline/databricks.yml:13:7)",
		// },
		// {
		// 	name:     "duplicate_resource_names_in_root_job_and_experiment",
		// 	errorMsg: "multiple resources named foo (experiments.foo at validate/duplicate_resource_names_in_root_job_and_experiment/databricks.yml:18:7, jobs.foo at validate/duplicate_resource_names_in_root_job_and_experiment/databricks.yml:10:7)",
		// },
		// {
		// 	name:     "duplicate_resource_name_in_subconfiguration",
		// 	errorMsg: "multiple resources named foo (jobs.foo at validate/duplicate_resource_name_in_subconfiguration/databricks.yml:13:7, pipelines.foo at validate/duplicate_resource_name_in_subconfiguration/resources.yml:4:7)",
		// },
		{
			name:     "duplicate_resource_name_in_subconfiguration_job_and_job",
			errorMsg: "resource jobs.foo has been defined at multiple locations: (validate/duplicate_resource_name_in_subconfiguration_job_and_job/databricks.yml:13:13, validate/duplicate_resource_name_in_subconfiguration_job_and_job/resources.yml:4:13)",
		},
		// {
		// 	name:     "duplicate_resource_names_in_different_subconfiguations",
		// 	errorMsg: "multiple resources named foo (jobs.foo at validate/duplicate_resource_names_in_different_subconfiguations/resources1.yml:4:7, pipelines.foo at validate/duplicate_resource_names_in_different_subconfiguations/resources2.yml:4:7)",
		// },
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			b := load(t, "./validate/"+tc.name)
			diags := bundle.ApplyReadOnly(context.Background(), bundle.ReadOnly(b), validate.UniqueResourceKeys())
			assert.ErrorContains(t, diags.Error(), tc.errorMsg)
		})
	}
}
