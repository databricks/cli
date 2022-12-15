package terraform

import (
	"testing"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/libraries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertJob(t *testing.T) {
	var src = resources.Job{
		JobSettings: &jobs.JobSettings{
			Name: "my job",
			JobClusters: []jobs.JobCluster{
				{
					JobClusterKey: "key",
					NewCluster: &clusters.CreateCluster{
						SparkVersion: "10.4.x-scala2.12",
					},
				},
			},
			GitSource: &jobs.GitSource{
				GitProvider: jobs.GitSourceGitProviderGithub,
				GitUrl:      "https://github.com/foo/bar",
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	assert.Equal(t, "my job", out.Resource.Job["my_job"].Name)
	assert.Len(t, out.Resource.Job["my_job"].JobCluster, 1)
	assert.Equal(t, "https://github.com/foo/bar", out.Resource.Job["my_job"].GitSource.Url)
	assert.Nil(t, out.Data)
}

func TestConvertJobTaskLibraries(t *testing.T) {
	var src = resources.Job{
		JobSettings: &jobs.JobSettings{
			Name: "my job",
			Tasks: []jobs.JobTaskSettings{
				{
					TaskKey: "key",
					Libraries: []libraries.Library{
						{
							Pypi: &libraries.PythonPyPiLibrary{
								Package: "mlflow",
							},
						},
					},
				},
			},
		},
	}

	var config = config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": &src,
			},
		},
	}

	out := BundleToTerraform(&config)
	assert.Equal(t, "my job", out.Resource.Job["my_job"].Name)
	require.Len(t, out.Resource.Job["my_job"].Task, 1)
	require.Len(t, out.Resource.Job["my_job"].Task[0].Library, 1)
	assert.Equal(t, "mlflow", out.Resource.Job["my_job"].Task[0].Library[0].Pypi.Package)
}
