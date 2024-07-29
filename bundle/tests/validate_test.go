package config_tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUniqueResourceIdentifiers(t *testing.T) {
	tcases := []struct {
		name        string
		diagnostics diag.Diagnostics
	}{
		{
			name: "duplicate_resource_names_in_root_job_and_pipeline",
			diagnostics: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "multiple resources have been defined with the same key: foo",
					Locations: []dyn.Location{
						{File: filepath.FromSlash("validate/duplicate_resource_names_in_root_job_and_pipeline/databricks.yml"), Line: 10, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_names_in_root_job_and_pipeline/databricks.yml"), Line: 13, Column: 7},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("jobs.foo"),
						dyn.MustPathFromString("pipelines.foo"),
					},
				},
			},
		},
		{
			name: "duplicate_resource_names_in_root_job_and_experiment",
			diagnostics: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "multiple resources have been defined with the same key: foo",
					Locations: []dyn.Location{
						{File: filepath.FromSlash("validate/duplicate_resource_names_in_root_job_and_experiment/databricks.yml"), Line: 10, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_names_in_root_job_and_experiment/databricks.yml"), Line: 18, Column: 7},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("experiments.foo"),
						dyn.MustPathFromString("jobs.foo"),
					},
				},
			},
		},
		{
			name: "duplicate_resource_name_in_subconfiguration",
			diagnostics: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "multiple resources have been defined with the same key: foo",
					Locations: []dyn.Location{
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_subconfiguration/databricks.yml"), Line: 13, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_subconfiguration/resources.yml"), Line: 4, Column: 7},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("jobs.foo"),
						dyn.MustPathFromString("pipelines.foo"),
					},
				},
			},
		},
		{
			name: "duplicate_resource_name_in_subconfiguration_job_and_job",
			diagnostics: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "multiple resources have been defined with the same key: foo",
					Locations: []dyn.Location{
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_subconfiguration_job_and_job/databricks.yml"), Line: 13, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_subconfiguration_job_and_job/resources.yml"), Line: 4, Column: 7},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("jobs.foo"),
					},
				},
			},
		},
		{
			name: "duplicate_resource_names_in_different_subconfiguations",
			diagnostics: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "multiple resources have been defined with the same key: foo",
					Locations: []dyn.Location{
						{File: filepath.FromSlash("validate/duplicate_resource_names_in_different_subconfiguations/resources1.yml"), Line: 4, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_names_in_different_subconfiguations/resources2.yml"), Line: 4, Column: 7},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("jobs.foo"),
						dyn.MustPathFromString("pipelines.foo"),
					},
				},
			},
		},
		{
			name: "duplicate_resource_name_in_multiple_locations",
			diagnostics: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "multiple resources have been defined with the same key: foo",
					Locations: []dyn.Location{
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_multiple_locations/databricks.yml"), Line: 13, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_multiple_locations/resources1.yml"), Line: 4, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_multiple_locations/resources1.yml"), Line: 8, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_multiple_locations/resources2.yml"), Line: 4, Column: 7},
						{File: filepath.FromSlash("validate/duplicate_resource_name_in_multiple_locations/resources2.yml"), Line: 8, Column: 7},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("experiments.foo"),
						dyn.MustPathFromString("jobs.foo"),
						dyn.MustPathFromString("pipelines.foo"),
					},
				},
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			b, err := bundle.Load(ctx, "./validate/"+tc.name)
			require.NoError(t, err)

			// The UniqueResourceKeys mutator is run as part of the Load phase.
			diags := bundle.Apply(ctx, b, phases.Load())
			assert.Equal(t, tc.diagnostics, diags)
		})
	}
}
