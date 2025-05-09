package mutator

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

func TestNoWorkspacePrefixUsed(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/test",
				ArtifactPath: "/Workspace/Users/test/artifacts",
				FilePath:     "/Workspace/Users/test/files",
				StatePath:    "/Workspace/Users/test/state",
			},

			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"test_job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "/Workspace/${workspace.root_path}/file1.py",
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "/Workspace${workspace.file_path}/notebook1",
									},
									Libraries: []compute.Library{
										{
											Jar: "/Workspace/${workspace.artifact_path}/jar1.jar",
										},
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "${workspace.file_path}/notebook2",
									},
									Libraries: []compute.Library{
										{
											Jar: "${workspace.artifact_path}/jar2.jar",
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

	diags := bundle.Apply(context.Background(), b, RewriteWorkspacePrefix())
	require.Len(t, diags, 3)

	expectedErrors := map[string]bool{
		`substring "/Workspace/${workspace.root_path}" found in "/Workspace/${workspace.root_path}/file1.py". Please update this to "${workspace.root_path}/file1.py".`:             true,
		`substring "/Workspace${workspace.file_path}" found in "/Workspace${workspace.file_path}/notebook1". Please update this to "${workspace.file_path}/notebook1".`:             true,
		`substring "/Workspace/${workspace.artifact_path}" found in "/Workspace/${workspace.artifact_path}/jar1.jar". Please update this to "${workspace.artifact_path}/jar1.jar".`: true,
	}

	for _, d := range diags {
		require.Equal(t, diag.Warning, d.Severity)
		require.Contains(t, expectedErrors, d.Summary)
		delete(expectedErrors, d.Summary)
	}

	require.Equal(t, "${workspace.root_path}/file1.py", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[0].SparkPythonTask.PythonFile)
	require.Equal(t, "${workspace.file_path}/notebook1", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[1].NotebookTask.NotebookPath)
	require.Equal(t, "${workspace.artifact_path}/jar1.jar", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[1].Libraries[0].Jar)
	require.Equal(t, "${workspace.file_path}/notebook2", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[2].NotebookTask.NotebookPath)
	require.Equal(t, "${workspace.artifact_path}/jar2.jar", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[2].Libraries[0].Jar)
}
