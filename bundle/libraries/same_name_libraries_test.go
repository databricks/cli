package libraries

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
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

	diags := bundle.Apply(context.Background(), b, CheckForSameNameLibraries())
	require.Len(t, diags, 1)
	require.Equal(t, diag.Error, diags[0].Severity)
	require.Equal(t, "Duplicate local library name", diags[0].Summary)
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
