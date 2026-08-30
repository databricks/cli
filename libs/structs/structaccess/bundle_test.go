package structaccess

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/apps"
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

// A bundle resource embeds a config struct that embeds the SDK request struct, so its
// fields sit two levels down. Get, Set and ValidatePath all have to reach them, and
// ForceSendFields belongs to the struct that declares the field -- not to the outer one
// that shadows the name.
func TestGetSet_DoublyEmbeddedField(t *testing.T) {
	project := &resources.PostgresProject{} //exhaustruct:ignore
	project.ProjectId = "p"

	require.NoError(t, ValidateByString(reflect.TypeOf(project), "budget_policy_id"))

	require.NoError(t, SetByString(project, "budget_policy_id", "abc"))
	require.Equal(t, "abc", project.BudgetPolicyId)

	value, err := GetByString(project, "budget_policy_id")
	require.NoError(t, err)
	require.Equal(t, "abc", value)

	// An explicit empty value is recorded on ProjectSpec, which declares the field.
	require.NoError(t, SetByString(project, "budget_policy_id", ""))
	require.Contains(t, project.ProjectSpec.ForceSendFields, "BudgetPolicyId")
	require.NotContains(t, project.PostgresProjectConfig.ForceSendFields, "BudgetPolicyId")

	value, err = GetByString(project, "budget_policy_id")
	require.NoError(t, err)
	require.Equal(t, "", value)

	// And dropping it again leaves the field absent.
	require.NoError(t, SetByString(project, "budget_policy_id", nil))
	require.NotContains(t, project.ProjectSpec.ForceSendFields, "BudgetPolicyId")
	value, err = GetByString(project, "budget_policy_id")
	require.NoError(t, err)
	require.Nil(t, value)
}
