package loader

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
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

	m := ProcessInclude(filepath.Join(b.RootPath, "host.yml"), "host.yml")
	assert.Equal(t, "ProcessInclude(host.yml)", m.Name())

	// Assert the host value prior to applying the mutator
	assert.Equal(t, "foo", b.Config.Workspace.Host)

	// Apply the mutator and assert that the host value has been updated
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

func TestProcessIncludeValidatesFileFormat(t *testing.T) {
	b := &bundle.Bundle{
		RootPath: "testdata/format",
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "format_test",
			},
		},
	}

	m := ProcessInclude(filepath.Join(b.RootPath, "foo.job.yml"), "foo.job.yml")
	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	// Assert that the diagnostics contain the expected information
	assert.Len(t, diags, 1)
	assert.Equal(t, diag.Diagnostics{
		{
			Severity: diag.Info,
			Summary:  "We recommend only defining a single job in a file with the .job.yml extension.\nThe following resources are defined or configured in this file:\n  - bar (job)\n  - foo (job)\n",
			Locations: []dyn.Location{
				{File: filepath.FromSlash("testdata/format/foo.job.yml"), Line: 4, Column: 7},
				{File: filepath.FromSlash("testdata/format/foo.job.yml"), Line: 7, Column: 7},
			},
			Paths: []dyn.Path{
				dyn.MustPathFromString("resources.jobs.bar"),
				dyn.MustPathFromString("resources.jobs.foo"),
			},
		},
	}, diags)
}

func TestResourceNames(t *testing.T) {
	names := []string{}
	typ := reflect.TypeOf(config.Resources{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTags := strings.Split(field.Tag.Get("json"), ",")
		singularName := strings.TrimSuffix(jsonTags[0], "s")
		names = append(names, singularName)
	}

	// Assert the contents of the two lists are equal. Please add the singular
	// name of your resource to resourceNames global if you are adding a new
	// resource.
	assert.Equal(t, len(resourceTypes), len(names))
	for _, name := range names {
		assert.Contains(t, resourceTypes, name)
	}
}

func TestValidateFileFormat(t *testing.T) {
	onlyJob := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job1": {},
			},
		},
		Targets: map[string]*config.Target{
			"target1": {
				Resources: &config.Resources{
					Jobs: map[string]*resources.Job{
						"job1": {},
					},
				},
			},
		},
	}
	onlyJobBundle := bundle.Bundle{Config: onlyJob}

	onlyPipeline := config.Root{
		Resources: config.Resources{
			Pipelines: map[string]*resources.Pipeline{
				"pipeline1": {},
			},
		},
	}
	onlyPipelineBundle := bundle.Bundle{Config: onlyPipeline}

	bothJobAndPipeline := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job1": {},
			},
		},
		Targets: map[string]*config.Target{
			"target1": {
				Resources: &config.Resources{
					Pipelines: map[string]*resources.Pipeline{
						"pipeline1": {},
					},
				},
			},
		},
	}
	bothJobAndPipelineBundle := bundle.Bundle{Config: bothJobAndPipeline}

	twoJobs := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job1": {},
				"job2": {},
			},
		},
	}
	twoJobsBundle := bundle.Bundle{Config: twoJobs}

	twoJobsTopLevelAndTarget := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job1": {},
			},
		},
		Targets: map[string]*config.Target{
			"target1": {
				Resources: &config.Resources{
					Jobs: map[string]*resources.Job{
						"job2": {},
					},
				},
			},
		},
	}
	twoJobsTopLevelAndTargetBundle := bundle.Bundle{Config: twoJobsTopLevelAndTarget}

	twoJobsInTarget := config.Root{
		Targets: map[string]*config.Target{
			"target1": {
				Resources: &config.Resources{
					Jobs: map[string]*resources.Job{
						"job1": {},
						"job2": {},
					},
				},
			},
		},
	}
	twoJobsInTargetBundle := bundle.Bundle{Config: twoJobsInTarget}

	tcases := []struct {
		name      string
		bundle    *bundle.Bundle
		expected  diag.Diagnostics
		fileName  string
		locations map[string]dyn.Location
	}{
		{
			name:     "single job",
			bundle:   &onlyJobBundle,
			expected: nil,
			fileName: "foo.job.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1": {File: "foo.job.yml", Line: 1, Column: 1},
			},
		},
		{
			name:     "single pipeline",
			bundle:   &onlyPipelineBundle,
			expected: nil,
			fileName: "foo.pipeline.yml",
			locations: map[string]dyn.Location{
				"resources.pipelines.pipeline1": {File: "foo.pipeline.yaml", Line: 1, Column: 1},
			},
		},
		{
			name:   "single job but extension is pipeline",
			bundle: &onlyJobBundle,
			expected: diag.Diagnostics{
				{
					Severity: diag.Info,
					Summary:  "We recommend only defining a single pipeline in a file with the .pipeline.yml extension.\nThe following resources are defined or configured in this file:\n  - job1 (job)\n",
					Locations: []dyn.Location{
						{File: "foo.pipeline.yml", Line: 1, Column: 1},
						{File: "foo.pipeline.yml", Line: 2, Column: 2},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("resources.jobs.job1"),
						dyn.MustPathFromString("targets.target1.resources.jobs.job1"),
					},
				},
			},
			fileName: "foo.pipeline.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1":                 {File: "foo.pipeline.yml", Line: 1, Column: 1},
				"targets.target1.resources.jobs.job1": {File: "foo.pipeline.yml", Line: 2, Column: 2},
			},
		},
		{
			name:     "job and pipeline",
			bundle:   &bothJobAndPipelineBundle,
			expected: nil,
			fileName: "foo.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1":                           {File: "foo.yml", Line: 1, Column: 1},
				"targets.target1.resources.pipelines.pipeline1": {File: "foo.yml", Line: 2, Column: 2},
			},
		},
		{
			name:   "job and pipeline but extension is job",
			bundle: &bothJobAndPipelineBundle,
			expected: diag.Diagnostics{
				{
					Severity: diag.Info,
					Summary:  "We recommend only defining a single job in a file with the .job.yml extension.\nThe following resources are defined or configured in this file:\n  - job1 (job)\n  - pipeline1 (pipeline)\n",
					Locations: []dyn.Location{
						{File: "foo.job.yml", Line: 1, Column: 1},
						{File: "foo.job.yml", Line: 2, Column: 2},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("resources.jobs.job1"),
						dyn.MustPathFromString("targets.target1.resources.pipelines.pipeline1"),
					},
				},
			},
			fileName: "foo.job.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1":                           {File: "foo.job.yml", Line: 1, Column: 1},
				"targets.target1.resources.pipelines.pipeline1": {File: "foo.job.yml", Line: 2, Column: 2},
			},
		},
		{
			name:   "job and pipeline but extension is experiment",
			bundle: &bothJobAndPipelineBundle,
			expected: diag.Diagnostics{
				{
					Severity: diag.Info,
					Summary:  "We recommend only defining a single experiment in a file with the .experiment.yml extension.\nThe following resources are defined or configured in this file:\n  - job1 (job)\n  - pipeline1 (pipeline)\n",
					Locations: []dyn.Location{
						{File: "foo.experiment.yml", Line: 1, Column: 1},
						{File: "foo.experiment.yml", Line: 2, Column: 2},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("resources.jobs.job1"),
						dyn.MustPathFromString("targets.target1.resources.pipelines.pipeline1"),
					},
				},
			},
			fileName: "foo.experiment.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1":                           {File: "foo.experiment.yml", Line: 1, Column: 1},
				"targets.target1.resources.pipelines.pipeline1": {File: "foo.experiment.yml", Line: 2, Column: 2},
			},
		},
		{
			name:   "two jobs",
			bundle: &twoJobsBundle,
			expected: diag.Diagnostics{
				{
					Severity: diag.Info,
					Summary:  "We recommend only defining a single job in a file with the .job.yml extension.\nThe following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
					Locations: []dyn.Location{
						{File: "foo.job.yml", Line: 1, Column: 1},
						{File: "foo.job.yml", Line: 2, Column: 2},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("resources.jobs.job1"),
						dyn.MustPathFromString("resources.jobs.job2"),
					},
				},
			},
			fileName: "foo.job.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1": {File: "foo.job.yml", Line: 1, Column: 1},
				"resources.jobs.job2": {File: "foo.job.yml", Line: 2, Column: 2},
			},
		},
		{
			name:     "two jobs but extension is simple yaml",
			bundle:   &twoJobsBundle,
			expected: nil,
			fileName: "foo.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1": {File: "foo.yml", Line: 1, Column: 1},
				"resources.jobs.job2": {File: "foo.yml", Line: 2, Column: 2},
			},
		},
		{
			name:   "two jobs in top level and target",
			bundle: &twoJobsTopLevelAndTargetBundle,
			expected: diag.Diagnostics{
				{
					Severity: diag.Info,
					Summary:  "We recommend only defining a single job in a file with the .job.yml extension.\nThe following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
					Locations: []dyn.Location{
						{File: "foo.job.yml", Line: 1, Column: 1},
						{File: "foo.job.yml", Line: 2, Column: 2},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString("resources.jobs.job1"),
						dyn.MustPathFromString("targets.target1.resources.jobs.job2"),
					},
				},
			},
			fileName: "foo.job.yml",
			locations: map[string]dyn.Location{
				"resources.jobs.job1":                 {File: "foo.job.yml", Line: 1, Column: 1},
				"targets.target1.resources.jobs.job2": {File: "foo.job.yml", Line: 2, Column: 2},
			},
		},
		{
			name:   "two jobs in target",
			bundle: &twoJobsInTargetBundle,
			expected: diag.Diagnostics{
				{
					Severity: diag.Info,
					Summary:  "We recommend only defining a single job in a file with the .job.yml extension.\nThe following resources are defined or configured in this file:\n  - job1 (job)\n  - job2 (job)\n",
					Locations: []dyn.Location{
						{File: "foo.job.yml", Line: 1, Column: 1},
						{File: "foo.job.yml", Line: 2, Column: 2},
					},
					Paths: []dyn.Path{
						dyn.MustPathFromString(("targets.target1.resources.jobs.job1")),
						dyn.MustPathFromString("targets.target1.resources.jobs.job2"),
					},
				},
			},
			fileName: "foo.job.yml",
			locations: map[string]dyn.Location{
				"targets.target1.resources.jobs.job1": {File: "foo.job.yml", Line: 1, Column: 1},
				"targets.target1.resources.jobs.job2": {File: "foo.job.yml", Line: 2, Column: 2},
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.locations {
				bundletest.SetLocation(tc.bundle, k, []dyn.Location{v})
			}

			diags := validateFileFormat(&tc.bundle.Config, tc.fileName)
			assert.Equal(t, tc.expected, diags)
		})
	}
}
