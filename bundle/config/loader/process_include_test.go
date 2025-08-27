package loader_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessInclude(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "testdata/basic",
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
		},
	}

	m := loader.ProcessInclude(filepath.Join(b.BundleRootPath, "host.yml"), "host.yml")
	assert.Equal(t, "ProcessInclude(host.yml)", m.Name())

	// Assert the host value prior to applying the mutator
	assert.Equal(t, "foo", b.Config.Workspace.Host)

	// Apply the mutator and assert that the host value has been updated
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

func TestProcessIncludeFormatMatch(t *testing.T) {
	for _, fileName := range []string{
		"one_job.job.yml",
		"one_pipeline.pipeline.yaml",
		"two_job.yml",
		"job_and_pipeline.yml",
		"multiple_resources.yml",
	} {
		t.Run(fileName, func(t *testing.T) {
			b := &bundle.Bundle{
				BundleRootPath: "testdata/format_match",
				Config: config.Root{
					Bundle: config.Bundle{
						Name: "format_test",
					},
				},
			}

			m := loader.ProcessInclude(filepath.Join(b.BundleRootPath, fileName), fileName)
			diags := bundle.Apply(context.Background(), b, m)
			assert.Empty(t, diags)
		})
	}
}

func TestProcessIncludeFormatNotMatch(t *testing.T) {
	for fileName, expectedDiags := range map[string]diag.Diagnostics{
		"single_job.pipeline.yaml": {
			{
				Severity: diag.Recommendation,
				Summary:  "define a single pipeline in a file with the .pipeline.yaml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n",
				Locations: []dyn.Location{
					{File: "testdata/format_not_match/single_job.pipeline.yaml", Line: 11, Column: 11},
					{File: "testdata/format_not_match/single_job.pipeline.yaml", Line: 4, Column: 7},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.jobs.job1"),
					dyn.MustPathFromString("targets.target1.resources.jobs.job1"),
				},
			},
		},
		"job_and_pipeline.job.yml": {
			{
				Severity: diag.Recommendation,
				Summary:  "define a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - pipeline1 (pipeline)\n",
				Locations: []dyn.Location{
					{File: "testdata/format_not_match/job_and_pipeline.job.yml", Line: 11, Column: 11},
					{File: "testdata/format_not_match/job_and_pipeline.job.yml", Line: 4, Column: 7},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.pipelines.pipeline1"),
					dyn.MustPathFromString("targets.target1.resources.jobs.job1"),
				},
			},
		},
		"job_and_pipeline.experiment.yml": {
			{
				Severity: diag.Recommendation,
				Summary:  "define a single experiment in a file with the .experiment.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - pipeline1 (pipeline)\n",
				Locations: []dyn.Location{
					{File: "testdata/format_not_match/job_and_pipeline.experiment.yml", Line: 11, Column: 11},
					{File: "testdata/format_not_match/job_and_pipeline.experiment.yml", Line: 4, Column: 7},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.pipelines.pipeline1"),
					dyn.MustPathFromString("targets.target1.resources.jobs.job1"),
				},
			},
		},
		"two_jobs.job.yml": {
			{
				Severity: diag.Recommendation,
				Summary:  "define a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
				Locations: []dyn.Location{
					{File: "testdata/format_not_match/two_jobs.job.yml", Line: 4, Column: 7},
					{File: "testdata/format_not_match/two_jobs.job.yml", Line: 7, Column: 7},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.jobs.job1"),
					dyn.MustPathFromString("resources.jobs.job2"),
				},
			},
		},
		"second_job_in_target.job.yml": {
			{
				Severity: diag.Recommendation,
				Summary:  "define a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
				Locations: []dyn.Location{
					{File: "testdata/format_not_match/second_job_in_target.job.yml", Line: 11, Column: 11},
					{File: "testdata/format_not_match/second_job_in_target.job.yml", Line: 4, Column: 7},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.jobs.job1"),
					dyn.MustPathFromString("targets.target1.resources.jobs.job2"),
				},
			},
		},
		"two_jobs_in_target.job.yml": {
			{
				Severity: diag.Recommendation,
				Summary:  "define a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
				Locations: []dyn.Location{
					{File: "testdata/format_not_match/two_jobs_in_target.job.yml", Line: 6, Column: 11},
					{File: "testdata/format_not_match/two_jobs_in_target.job.yml", Line: 8, Column: 11},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("targets.target1.resources.jobs.job1"),
					dyn.MustPathFromString("targets.target1.resources.jobs.job2"),
				},
			},
		},
		"multiple_resources.model_serving_endpoint.yml": {
			{
				Severity: diag.Recommendation,
				Summary:  "define a single model serving endpoint in a file with the .model_serving_endpoint.yml extension.",
				Detail: `The following resources are defined or configured in this file:
  - experiment1 (experiment)
  - job1 (job)
  - job2 (job)
  - job3 (job)
  - model1 (model)
  - model_serving_endpoint1 (model_serving_endpoint)
  - pipeline1 (pipeline)
  - pipeline2 (pipeline)
  - quality_monitor1 (quality_monitor)
  - registered_model1 (registered_model)
  - schema1 (schema)
`,
				Locations: []dyn.Location{
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 12, Column: 7},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 14, Column: 7},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 18, Column: 7},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 22, Column: 7},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 24, Column: 7},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 28, Column: 7},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 35, Column: 11},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 39, Column: 11},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 43, Column: 11},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 4, Column: 7},
					{File: "testdata/format_not_match/multiple_resources.model_serving_endpoint.yml", Line: 8, Column: 7},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("resources.experiments.experiment1"),
					dyn.MustPathFromString("resources.jobs.job1"),
					dyn.MustPathFromString("resources.jobs.job2"),
					dyn.MustPathFromString("resources.model_serving_endpoints.model_serving_endpoint1"),
					dyn.MustPathFromString("resources.models.model1"),
					dyn.MustPathFromString("resources.pipelines.pipeline1"),
					dyn.MustPathFromString("resources.pipelines.pipeline2"),
					dyn.MustPathFromString("resources.schemas.schema1"),
					dyn.MustPathFromString("targets.target1.resources.jobs.job3"),
					dyn.MustPathFromString("targets.target1.resources.quality_monitors.quality_monitor1"),
					dyn.MustPathFromString("targets.target1.resources.registered_models.registered_model1"),
				},
			},
		},
	} {
		t.Run(fileName, func(t *testing.T) {
			b := &bundle.Bundle{
				BundleRootPath: "testdata/format_not_match",
				Config: config.Root{
					Bundle: config.Bundle{
						Name: "format_test",
					},
				},
			}

			m := loader.ProcessInclude(filepath.Join(b.BundleRootPath, fileName), fileName)
			diags := bundle.Apply(context.Background(), b, m)
			require.Len(t, diags, 1)
			assert.Equal(t, expectedDiags, diags)
		})
	}
}
