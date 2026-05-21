package libraries

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestSwitchToPatchedWheelsUpdatesPipelineEnvironmentDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "dist", "my_test_code-0.0.1-py3-none-any.whl")
	sourceRel := filepath.Join("dist", "my_test_code-0.0.1-py3-none-any.whl")
	patched := filepath.Join(tmpDir, ".databricks", "bundle", "default", "patched_wheels", "wheel_my_test_code", "my_test_code-0.0.1+123-py3-none-any.whl")
	patchedRel := filepath.Join(".databricks", "bundle", "default", "patched_wheels", "wheel_my_test_code", "my_test_code-0.0.1+123-py3-none-any.whl")

	b := &bundle.Bundle{
		SyncRootPath: tmpDir,
		SyncRoot:     vfs.MustNew(tmpDir),
		Config: config.Root{
			Artifacts: config.Artifacts{
				"wheel": {
					Files: []config.ArtifactFile{
						{
							Source:  source,
							Patched: patched,
						},
					},
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Environments: []jobs.JobEnvironment{
								{
									EnvironmentKey: "env",
									Spec: &compute.Environment{
										Dependencies: []string{
											sourceRel,
											"simplejson",
										},
									},
								},
							},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
							Environment: &pipelines.PipelinesEnvironment{
								Dependencies: []string{
									sourceRel,
									"simplejson",
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, SwitchToPatchedWheels())
	require.Empty(t, diags)

	require.Equal(t, []string{
		patchedRel,
		"simplejson",
	}, b.Config.Resources.Jobs["job"].Environments[0].Spec.Dependencies)
	require.Equal(t, []string{
		patchedRel,
		"simplejson",
	}, b.Config.Resources.Pipelines["pipeline"].Environment.Dependencies)
}
