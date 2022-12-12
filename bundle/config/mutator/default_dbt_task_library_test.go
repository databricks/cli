package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/libraries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dbtTaskWithLibraries(lib ...libraries.Library) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.JobTaskSettings{
								{
									TaskKey: "key",
									DbtTask: &jobs.DbtTask{
										Commands: []string{
											"dbt run",
										},
									},
									Libraries: lib,
								},
							},
						},
					},
				},
			},
		},
	}

}

func TestDefaultDbtTaskLibraryWithoutLibraries(t *testing.T) {
	bundle := dbtTaskWithLibraries()
	_, err := mutator.DefaultDbtTaskLibrary().Apply(context.Background(), bundle)
	require.NoError(t, err)

	job := bundle.Config.Resources.Jobs["job"]
	libraries := job.Tasks[0].Libraries
	assert.Len(t, libraries, 1)
	assert.Equal(t, libraries[0].Pypi.Package, "dbt-databricks>=1.0.0,<2.0.0")
}

func TestDefaultDbtTaskLibraryWithUnrelatedLibraries(t *testing.T) {
	bundle := dbtTaskWithLibraries(
		libraries.Library{
			Egg: "dbfs:/my_egg",
		},
		libraries.Library{
			Pypi: &libraries.PythonPyPiLibrary{
				Package: "databricks",
			},
		},
	)
	_, err := mutator.DefaultDbtTaskLibrary().Apply(context.Background(), bundle)
	require.NoError(t, err)

	job := bundle.Config.Resources.Jobs["job"]
	libraries := job.Tasks[0].Libraries
	assert.Len(t, libraries, 3)
	assert.Equal(t, libraries[len(libraries)-1].Pypi.Package, "dbt-databricks>=1.0.0,<2.0.0")
}

func TestDefaultDbtTaskLibraryWithDbtDatabricks(t *testing.T) {
	bundle := dbtTaskWithLibraries(
		libraries.Library{
			Pypi: &libraries.PythonPyPiLibrary{
				Package: "dbt-databricks>=2.0.0,<3.0.0",
			},
		},
	)
	_, err := mutator.DefaultDbtTaskLibrary().Apply(context.Background(), bundle)
	require.NoError(t, err)

	job := bundle.Config.Resources.Jobs["job"]
	libraries := job.Tasks[0].Libraries
	assert.Len(t, libraries, 1)
	assert.Equal(t, libraries[0].Pypi.Package, "dbt-databricks>=2.0.0,<3.0.0")
}
