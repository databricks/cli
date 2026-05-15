package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableReferencesWithSourceLinkedDeployment(t *testing.T) {
	testCases := []struct {
		enabled bool
		assert  func(t *testing.T, b *bundle.Bundle)
	}{
		{
			true,
			func(t *testing.T, b *bundle.Bundle) {
				// Variables that use workspace file path should have SyncRootValue during resolution phase
				require.Equal(t, "sync/root/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])

				// The file path itself should remain the same
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
		{
			false,
			func(t *testing.T, b *bundle.Bundle) {
				require.Equal(t, "file/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
	}

	for _, testCase := range testCases {
		b := &bundle.Bundle{
			SyncRootPath: "sync/root/path",
			Config: config.Root{
				Presets: config.Presets{
					SourceLinkedDeployment: &testCase.enabled,
				},
				Workspace: config.Workspace{
					FilePath: "file/path",
				},
				Resources: config.Resources{
					Pipelines: map[string]*resources.Pipeline{
						"pipeline1": {
							CreatePipeline: pipelines.CreatePipeline{
								Configuration: map[string]string{
									"source": "${workspace.file_path}",
								},
							},
						},
					},
				},
			},
		}

		diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources("workspace"))
		require.NoError(t, diags.Error())
		testCase.assert(t, b)
	}
}

// TestResolveVariableReferencesExcludePaths verifies that paths listed in excludePaths
// are skipped during resolution and left as unresolved variable references.
// This is used by ProcessStaticResources for immutable bundles so that
// ${workspace.file_path} and ${workspace.artifact_path} can be resolved later
// (in the Build phase, after artifacts are built and the correct snapshot path is known).
func TestResolveVariableReferencesExcludePaths(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath:     "/snapshot/path/src/files",
				ArtifactPath: "/snapshot/path/src/artifacts",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									SparkPythonTask: &jobs.SparkPythonTask{
										PythonFile: "${workspace.file_path}/main.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// With exclusion: ${workspace.file_path} should remain unresolved.
	diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResourcesExcluding("workspace.file_path", "workspace.artifact_path"))
	require.NoError(t, diags.Error())
	assert.Equal(t, "${workspace.file_path}/main.py", b.Config.Resources.Jobs["job1"].Tasks[0].SparkPythonTask.PythonFile,
		"reference should remain unresolved when path is excluded")

	// Without exclusion: ${workspace.file_path} should resolve normally.
	diags = bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources())
	require.NoError(t, diags.Error())
	assert.Equal(t, "/snapshot/path/src/files/main.py", b.Config.Resources.Jobs["job1"].Tasks[0].SparkPythonTask.PythonFile,
		"reference should resolve after exclusion is lifted")
}
