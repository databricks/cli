package structaccess

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestGet_ConfigRoot_JobTagsAccess(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": {
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
		},
	}

	// Access a value inside the tags map
	v, err := Get(root, "resources.jobs.my_job.tags.env")
	require.NoError(t, err)
	require.Equal(t, "dev", v)

	// Leading dot is allowed
	v, err = Get(root, ".resources.jobs.my_job.tags.team")
	require.NoError(t, err)
	require.Equal(t, "platform", v)

	// Access into first task
	v, err = Get(root, "resources.jobs.my_job.tasks[0].task_key")
	require.NoError(t, err)
	require.Equal(t, "t1", v)

	v, err = Get(root, "resources.jobs.my_job.tasks[0].notebook_task.notebook_path")
	require.NoError(t, err)
	require.Equal(t, "/Workspace/Users/user@example.com/nb", v)
}
