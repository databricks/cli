package structaccess

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestGet_ConfigRoot_JobTagsAccess(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": {
					BaseResource: resources.BaseResource{ID: "jobid", URL: "joburl"},
					JobSettings: jobs.JobSettings{
						Name: "example",
						Tasks: []jobs.Task{
							{
								TaskKey:      "t1",
								NotebookTask: &jobs.NotebookTask{NotebookPath: "/Workspace/Users/user@example.com/nb"},
							},
						},
						Tags: map[string]string{
							"env":  "dev",
							"team": "platform",
						},
					},
				},
			},
			Apps: map[string]*resources.App{
				"my_app": {
					BaseResource: resources.BaseResource{URL: "app_outer_url"},
					App: apps.App{
						Url: "app_inner_url",
					},
				},
			},
		},
	}

	// Access a value inside the tags map
	v, err := GetByString(root, "resources.jobs.my_job.tags.env")
	require.NoError(t, err)
	require.Equal(t, "dev", v)
	require.NoError(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tags.env"))
	require.NoError(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tags.anything"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tags.env.inner"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tags1"))

	// Array indexing test (1)
	v, err = GetByString(root, "resources.jobs.my_job.tasks[0].task_key")
	require.NoError(t, err)
	require.Equal(t, "t1", v)
	require.NoError(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tasks[0].task_key"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tasks[0].task_key.inner"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tasks[0].task_key1"))

	// Array indexing test (2)
	v, err = GetByString(root, "resources.jobs.my_job.tasks[0].notebook_task.notebook_path")
	require.NoError(t, err)
	require.Equal(t, "/Workspace/Users/user@example.com/nb", v)
	require.NoError(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tasks[0].notebook_task.notebook_path"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tasks[0].notebook_task.notebook_path.inner"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tasks[0].notebook_task.notebook_path1"))

	// Test ambiguous field access: outer is ignored because it has bundle tag
	v, err = GetByString(root, "resources.apps.my_app.url")
	require.NoError(t, err)
	require.Equal(t, "app_inner_url", v)
	require.NoError(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.apps.my_app.url"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.apps.my_app.url.inner"))
	require.Error(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.apps.my_app.url1"))
}

// TestGet_NewClusterArraySyntax verifies that [0] on a struct field is a no-op,
// enabling terraform-style references like tasks[0].new_cluster[0].spark_version
// where new_cluster is a plain struct in the Go SDK but a single-block attribute
// in terraform (represented as an array of length 1).
func TestGet_NewClusterArraySyntax(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": {
					JobSettings: jobs.JobSettings{
						Tasks: []jobs.Task{
							{
								TaskKey: "t1",
								NewCluster: &compute.ClusterSpec{
									SparkVersion: "15.0.x-scala2.12",
									NodeTypeId:   "Standard_DS3_v2",
								},
							},
						},
					},
				},
			},
		},
	}

	// [0] on new_cluster struct is a no-op (terraform-style single-block access).
	v, err := GetByString(root, "resources.jobs.my_job.tasks[0].new_cluster[0].spark_version")
	require.NoError(t, err)
	require.Equal(t, "15.0.x-scala2.12", v)
	require.NoError(t, ValidateByString(reflect.TypeFor[config.Root](), "resources.jobs.my_job.tasks[0].new_cluster[0].spark_version"))

	// Without [0]: same result.
	v, err = GetByString(root, "resources.jobs.my_job.tasks[0].new_cluster.spark_version")
	require.NoError(t, err)
	require.Equal(t, "15.0.x-scala2.12", v)

	// [1] on a struct is still an error.
	_, err = GetByString(root, "resources.jobs.my_job.tasks[0].new_cluster[1].spark_version")
	require.Error(t, err)
}
