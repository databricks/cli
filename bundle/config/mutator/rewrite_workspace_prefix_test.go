package mutator

import (
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
				ResourcePath: "/Workspace/Users/test/resources",
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
								{
									NotebookTask: &jobs.NotebookTask{
										NotebookPath: "/Workspace/${workspace.resource_path}/notebook3",
									},
									Libraries: []compute.Library{
										{
											Jar: "/Workspace${workspace.resource_path}/jar3.jar",
										},
									},
								},
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "${workspace.file_path}/file2.py",
										Parameters: []string{
											"--input=/Workspace/${workspace.file_path}/in.txt --output=/Workspace/${workspace.file_path}/out.txt",
											"--cp=/Workspace/${workspace.root_path}/lib.jar:/Workspace${workspace.artifact_path}/dep.jar",
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

	diags := bundle.Apply(t.Context(), b, RewriteWorkspacePrefix())
	require.Len(t, diags, 8)

	expectedErrors := map[string]bool{
		`substring "/Workspace/${workspace.root_path}" found in "/Workspace/${workspace.root_path}/file1.py". Please update this to "${workspace.root_path}/file1.py".`:                                                                                                        true,
		`substring "/Workspace${workspace.file_path}" found in "/Workspace${workspace.file_path}/notebook1". Please update this to "${workspace.file_path}/notebook1".`:                                                                                                        true,
		`substring "/Workspace/${workspace.artifact_path}" found in "/Workspace/${workspace.artifact_path}/jar1.jar". Please update this to "${workspace.artifact_path}/jar1.jar".`:                                                                                            true,
		`substring "/Workspace/${workspace.resource_path}" found in "/Workspace/${workspace.resource_path}/notebook3". Please update this to "${workspace.resource_path}/notebook3".`:                                                                                          true,
		`substring "/Workspace${workspace.resource_path}" found in "/Workspace${workspace.resource_path}/jar3.jar". Please update this to "${workspace.resource_path}/jar3.jar".`:                                                                                              true,
		`substring "/Workspace/${workspace.file_path}" found in "--input=/Workspace/${workspace.file_path}/in.txt --output=/Workspace/${workspace.file_path}/out.txt". Please update this to "--input=${workspace.file_path}/in.txt --output=${workspace.file_path}/out.txt".`: true,
		`substring "/Workspace/${workspace.root_path}" found in "--cp=/Workspace/${workspace.root_path}/lib.jar:/Workspace${workspace.artifact_path}/dep.jar". Please update this to "--cp=${workspace.root_path}/lib.jar:/Workspace${workspace.artifact_path}/dep.jar".`:      true,
		`substring "/Workspace${workspace.artifact_path}" found in "--cp=/Workspace/${workspace.root_path}/lib.jar:/Workspace${workspace.artifact_path}/dep.jar". Please update this to "--cp=${workspace.root_path}/lib.jar:${workspace.artifact_path}/dep.jar".`:             true,
	}

	for _, d := range diags {
		require.Equal(t, diag.Warning, d.Severity)
		require.Contains(t, expectedErrors, d.Summary)
		delete(expectedErrors, d.Summary)
	}

	require.Equal(t, "${workspace.root_path}/file1.py", b.Config.Resources.Jobs["test_job"].Tasks[0].SparkPythonTask.PythonFile)
	require.Equal(t, "${workspace.file_path}/notebook1", b.Config.Resources.Jobs["test_job"].Tasks[1].NotebookTask.NotebookPath)
	require.Equal(t, "${workspace.artifact_path}/jar1.jar", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[1].Libraries[0].Jar)
	require.Equal(t, "${workspace.file_path}/notebook2", b.Config.Resources.Jobs["test_job"].Tasks[2].NotebookTask.NotebookPath)
	require.Equal(t, "${workspace.artifact_path}/jar2.jar", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[2].Libraries[0].Jar)
	require.Equal(t, "${workspace.resource_path}/notebook3", b.Config.Resources.Jobs["test_job"].Tasks[3].NotebookTask.NotebookPath)
	require.Equal(t, "${workspace.resource_path}/jar3.jar", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[3].Libraries[0].Jar)
	require.Equal(t, "${workspace.file_path}/file2.py", b.Config.Resources.Jobs["test_job"].Tasks[4].SparkPythonTask.PythonFile)
	require.Equal(t, "--input=${workspace.file_path}/in.txt --output=${workspace.file_path}/out.txt", b.Config.Resources.Jobs["test_job"].Tasks[4].SparkPythonTask.Parameters[0])
	require.Equal(t, "--cp=${workspace.root_path}/lib.jar:${workspace.artifact_path}/dep.jar", b.Config.Resources.Jobs["test_job"].Tasks[4].SparkPythonTask.Parameters[1])
}
