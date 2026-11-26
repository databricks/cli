package libraries

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitchToPatchedWheels(t *testing.T) {
	dir := t.TempDir()
	wheel := filepath.Join("dist", "my.whl")
	patched := filepath.Join(".databricks", "patched", "my-patched.whl")

	b := &bundle.Bundle{
		SyncRootPath: dir,
		SyncRoot:     vfs.MustNew(dir),
		Config: config.Root{
			Artifacts: config.Artifacts{
				"my": {
					Files: []config.ArtifactFile{
						{
							Source:  filepath.Join(dir, wheel),
							Patched: filepath.Join(dir, patched),
						},
					},
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey: "task",
									Libraries: []compute.Library{
										{Whl: wheel},
										{Jar: wheel},
										{Whl: "other.whl"},
									},
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
											TaskKey:   "nested",
											Libraries: []compute.Library{{Whl: wheel}},
										},
									},
								},
							},
							Environments: []jobs.JobEnvironment{
								{
									EnvironmentKey: "env",
									Spec: &compute.Environment{
										Dependencies: []string{wheel + "[train]", "faker"},
									},
								},
							},
						},
					},
				},
				Clusters: map[string]*resources.Cluster{
					"cluster": {
						Libraries: []compute.Library{{Whl: wheel}},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
							Environment: &pipelines.PipelinesEnvironment{
								Dependencies: []string{wheel},
							},
							Testing: &pipelines.PipelineTesting{
								Environment: &pipelines.PipelineTestEnvironment{
									Dependencies: []string{wheel},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []diag.Location{{File: filepath.Join(dir, "databricks.yml")}})

	diags := bundle.Apply(t.Context(), b, SwitchToPatchedWheels())
	require.Empty(t, diags)

	job := b.Config.Resources.Jobs["job"]
	assert.Equal(t, []compute.Library{
		{Whl: patched},
		{Jar: wheel},
		{Whl: "other.whl"},
	}, job.Tasks[0].Libraries)
	assert.Equal(t, []compute.Library{{Whl: patched}}, job.Tasks[0].ForEachTask.Task.Libraries)
	assert.Equal(t, []string{patched + "[train]", "faker"}, job.Environments[0].Spec.Dependencies)
	assert.Equal(t, []compute.Library{{Whl: patched}}, b.Config.Resources.Clusters["cluster"].Libraries)

	// Pipeline environment dependencies are not switched, unlike pipeline testing dependencies.
	pipeline := b.Config.Resources.Pipelines["pipeline"]
	assert.Equal(t, []string{wheel}, pipeline.Environment.Dependencies)
	assert.Equal(t, []string{patched}, pipeline.Testing.Environment.Dependencies)

	// A switched value keeps the location of the original value.
	for _, path := range []string{
		"resources.jobs.job.tasks[0].libraries[0].whl",
		"resources.jobs.job.tasks[0].for_each_task.task.libraries[0].whl",
		"resources.jobs.job.environments[0].spec.dependencies[0]",
		"resources.clusters.cluster.libraries[0].whl",
		"resources.pipelines.pipeline.testing.environment.dependencies[0]",
	} {
		v, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString(path))
		require.NoError(t, err, path)
		assert.Equal(t, filepath.Join(dir, "databricks.yml"), v.Location().File, path)
	}
}
