package libraries

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestSameNameLibraries(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									Libraries: []compute.Library{
										{
											Whl: "full/path/test.whl",
										},
									},
								},
								{
									Libraries: []compute.Library{
										{
											Whl: "other/path/test.whl",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.test.tasks[0]", []dyn.Location{
		{File: "databricks.yml", Line: 10, Column: 1},
	})
	bundletest.SetLocation(b, "resources.jobs.test.tasks[1]", []dyn.Location{
		{File: "databricks.yml", Line: 20, Column: 1},
	})

	diags := bundle.Apply(context.Background(), b, CheckForSameNameLibraries())
	require.Len(t, diags, 1)
	require.Equal(t, diag.Error, diags[0].Severity)
	require.Equal(t, "Duplicate local library name test.whl", diags[0].Summary)
	require.Equal(t, []dyn.Location{
		{File: "databricks.yml", Line: 10, Column: 1},
		{File: "databricks.yml", Line: 20, Column: 1},
	}, diags[0].Locations)

	paths := make([]string, 0)
	for _, p := range diags[0].Paths {
		paths = append(paths, p.String())
	}
	require.Equal(t, []string{
		"resources.jobs.test.tasks[0].libraries[0].whl",
		"resources.jobs.test.tasks[1].libraries[0].whl",
	}, paths)
}

func TestSameNameLibrariesWithUniqueLibraries(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									Libraries: []compute.Library{
										{
											Whl: "full/path/test-0.1.1.whl",
										},

										{
											Whl: "cowsay",
										},
									},
								},
								{
									Libraries: []compute.Library{
										{
											Whl: "other/path/test-0.1.0.whl",
										},

										{
											Whl: "cowsay",
										},
									},
								},
								{
									Libraries: []compute.Library{
										{
											Whl: "full/path/test-0.1.1.whl", // Use the same library as the first task
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, CheckForSameNameLibraries())
	require.Empty(t, diags)
}
