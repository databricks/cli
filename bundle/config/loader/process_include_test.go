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
		RootPath: "testdata/basic",
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
		},
	}

	m := loader.ProcessInclude(filepath.Join(b.RootPath, "host.yml"), "host.yml")
	assert.Equal(t, "ProcessInclude(host.yml)", m.Name())

	// Assert the host value prior to applying the mutator
	assert.Equal(t, "foo", b.Config.Workspace.Host)

	// Apply the mutator and assert that the host value has been updated
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

func TestProcessIncludeFormatPass(t *testing.T) {
	for _, fileName := range []string{
		"one_job.job.yml",
		"one_pipeline.pipeline.yaml",
		"two_job.yml",
		"job_and_pipeline.yml",
	} {
		t.Run(fileName, func(t *testing.T) {
			b := &bundle.Bundle{
				RootPath: "testdata/format_pass",
				Config: config.Root{
					Bundle: config.Bundle{
						Name: "format_test",
					},
				},
			}

			m := loader.ProcessInclude(filepath.Join(b.RootPath, fileName), fileName)
			diags := bundle.Apply(context.Background(), b, m)
			assert.Empty(t, diags)
		})
	}
}

func TestProcessIncludeFormatFail(t *testing.T) {
	for fileName, expectedDiags := range map[string]diag.Diagnostics{
		"single_job.pipeline.yaml": {
			{
				Severity: diag.Recommendation,
				Summary:  "We recommend only defining a single pipeline in a file with the .pipeline.yaml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n",
				Locations: []dyn.Location{
					{File: filepath.FromSlash("testdata/format_fail/single_job.pipeline.yaml"), Line: 11, Column: 11},
					{File: filepath.FromSlash("testdata/format_fail/single_job.pipeline.yaml"), Line: 4, Column: 7},
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
				Summary:  "We recommend only defining a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - pipeline1 (pipeline)\n",
				Locations: []dyn.Location{
					{File: filepath.FromSlash("testdata/format_fail/job_and_pipeline.job.yml"), Line: 11, Column: 11},
					{File: filepath.FromSlash("testdata/format_fail/job_and_pipeline.job.yml"), Line: 4, Column: 7},
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
				Summary:  "We recommend only defining a single experiment in a file with the .experiment.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - pipeline1 (pipeline)\n",
				Locations: []dyn.Location{
					{File: filepath.FromSlash("testdata/format_fail/job_and_pipeline.experiment.yml"), Line: 11, Column: 11},
					{File: filepath.FromSlash("testdata/format_fail/job_and_pipeline.experiment.yml"), Line: 4, Column: 7},
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
				Summary:  "We recommend only defining a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
				Locations: []dyn.Location{
					{File: filepath.FromSlash("testdata/format_fail/two_jobs.job.yml"), Line: 4, Column: 7},
					{File: filepath.FromSlash("testdata/format_fail/two_jobs.job.yml"), Line: 7, Column: 7},
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
				Summary:  "We recommend only defining a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
				Locations: []dyn.Location{
					{File: filepath.FromSlash("testdata/format_fail/second_job_in_target.job.yml"), Line: 11, Column: 11},
					{File: filepath.FromSlash("testdata/format_fail/second_job_in_target.job.yml"), Line: 4, Column: 7},
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
				Summary:  "We recommend only defining a single job in a file with the .job.yml extension.",
				Detail:   "The following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
				Locations: []dyn.Location{
					{File: filepath.FromSlash("testdata/format_fail/two_jobs_in_target.job.yml"), Line: 6, Column: 11},
					{File: filepath.FromSlash("testdata/format_fail/two_jobs_in_target.job.yml"), Line: 8, Column: 11},
				},
				Paths: []dyn.Path{
					dyn.MustPathFromString("targets.target1.resources.jobs.job1"),
					dyn.MustPathFromString("targets.target1.resources.jobs.job2"),
				},
			},
		},
	} {
		t.Run(fileName, func(t *testing.T) {
			b := &bundle.Bundle{
				RootPath: "testdata/format_fail",
				Config: config.Root{
					Bundle: config.Bundle{
						Name: "format_test",
					},
				},
			}

			m := loader.ProcessInclude(filepath.Join(b.RootPath, fileName), fileName)
			diags := bundle.Apply(context.Background(), b, m)
			require.Len(t, diags, 1)
			assert.Equal(t, expectedDiags, diags)
		})
	}
}
