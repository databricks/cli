package mutator

import (
	"context"
	pathlib "path"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestNormalizePaths_Jobs(t *testing.T) {
	tmpDir := t.TempDir()

	m := NormalizePaths()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{
						Tasks: []jobs.Task{
							{
								NotebookTask: &jobs.NotebookTask{
									NotebookPath: "src/notebook.py",
								},
							},
						},
					}},
				},
			},
		},
	}

	b.BundleRootPath = tmpDir

	location := dyn.Location{File: pathlib.Join(tmpDir, "resources/job_1.yml")}

	path := dyn.MustPathFromString("resources.jobs.job1.tasks[0].notebook_task.notebook_path")
	value := dyn.NewValue("../src/job_1.py", []dyn.Location{location})

	updateValue(t, b, path, value)

	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	newValue, err := dyn.GetByPath(b.Config.Value(), path)
	require.NoError(t, err)
	require.Equal(t, "src/job_1.py", newValue.MustString())
}

func TestNormalizePaths_Pipelines(t *testing.T) {
	tmpDir := t.TempDir()

	m := NormalizePaths()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						CreatePipeline: &pipelines.CreatePipeline{
							Libraries: []pipelines.PipelineLibrary{
								{
									File: &pipelines.FileLibrary{
										Path: "../src/library.py",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	b.BundleRootPath = tmpDir

	location := dyn.Location{File: pathlib.Join(tmpDir, "resources/pipeline_1.yml")}

	path := dyn.MustPathFromString("resources.pipelines.pipeline1.libraries[0].file.path")
	value := dyn.NewValue("../src/library.py", []dyn.Location{location})

	updateValue(t, b, path, value)

	diags := bundle.Apply(context.Background(), b, m)
	require.NoError(t, diags.Error())

	newValue, err := dyn.GetByPath(b.Config.Value(), path)
	require.NoError(t, err)
	require.Equal(t, "src/library.py", newValue.MustString())
	require.NotEqual(t, "", newValue.Directory())
}

func updateValue(t *testing.T, b *bundle.Bundle, path dyn.Path, newValue dyn.Value) {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.SetByPath(v, path, newValue)
	})
	require.NoError(t, err)
}
