package validate

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
						JobSettings: &jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "/Workspace/${workspace.root_path}/file1.py",
									},
								},
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "/Workspace/${workspace.file_path}/notebook1",
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

	diags := bundle.Apply(context.Background(), b, NoWorkspacePrefixUsed())
	require.Len(t, diags, 3)

	expectedErrors := map[string]bool{
		"/Workspace/${workspace.root_path} used in the remote path /Workspace/${workspace.root_path}/file1.py. Please change to use ${workspace.root_path}/file1.py instead":             true,
		"/Workspace/${workspace.file_path} used in the remote path /Workspace/${workspace.file_path}/notebook1. Please change to use ${workspace.file_path}/notebook1 instead":           true,
		"/Workspace/${workspace.artifact_path} used in the remote path /Workspace/${workspace.artifact_path}/jar1.jar. Please change to use ${workspace.artifact_path}/jar1.jar instead": true,
	}

	for _, d := range diags {
		require.Equal(t, d.Severity, diag.Error)
		require.Contains(t, expectedErrors, d.Summary)
		delete(expectedErrors, d.Summary)
	}
}
