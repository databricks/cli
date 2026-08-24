package mutator

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePaths(t *testing.T) {
	tmpDir := t.TempDir()
	m := NormalizePaths()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: jobs.JobSettings{
						Tasks: []jobs.Task{
							{
								NotebookTask: &jobs.NotebookTask{
									NotebookPath: "../src/notebook.py",
								},
							},
						},
					}},
				},
			},
		},
		BundleRootPath: tmpDir,
	}

	// update config as if 'notebook_path' property is defined in resources/job_1.yml
	location := dyn.Location{File: filepath.Join(tmpDir, "resources", "job_1.yml")}
	path := dyn.MustPathFromString("resources.jobs.job1.tasks[0].notebook_task.notebook_path")
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPath(v, path, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			return dyn.NewValue(value.MustString(), []dyn.Location{location}), nil
		})
	})
	require.NoError(t, err)

	diags := bundle.Apply(t.Context(), b, m)
	require.NoError(t, diags.Error())

	newValue, err := dyn.GetByPath(b.Config.Value(), path)
	require.NoError(t, err)
	require.Equal(t, "src/notebook.py", newValue.MustString())
}

func TestNormalizePaths_jobRunOnFileChange(t *testing.T) {
	tmpDir := t.TempDir()
	pattern := "../data/*.txt"
	m := NormalizePaths()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				JobRuns: map[string]*resources.JobRun{
					"run1": {
						Lifecycle: &resources.JobRunLifecycle{
							Triggers: []resources.JobRunTrigger{
								{OnFileChange: &pattern},
							},
						},
					},
				},
			},
		},
		BundleRootPath: tmpDir,
	}

	location := dyn.Location{File: filepath.Join(tmpDir, "resources", "run.yml")}
	path := dyn.MustPathFromString("resources.job_runs.run1.lifecycle.triggers[0].on_file_change")
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPath(v, path, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			return dyn.NewValue(value.MustString(), []dyn.Location{location}), nil
		})
	})
	require.NoError(t, err)

	diags := bundle.Apply(t.Context(), b, m)
	require.NoError(t, diags.Error())

	newValue, err := dyn.GetByPath(b.Config.Value(), path)
	require.NoError(t, err)
	require.Equal(t, "data/*.txt", newValue.MustString())
}

func TestNormalizePath_absolutePath(t *testing.T) {
	value, err := normalizePath("/notebook.py", dyn.Location{}, "/tmp")
	assert.NoError(t, err)
	assert.Equal(t, "/notebook.py", value)
}

func TestNormalizePath_url(t *testing.T) {
	value, err := normalizePath("s3:///path/to/notebook.py", dyn.Location{}, "/tmp")
	assert.NoError(t, err)
	assert.Equal(t, "s3:///path/to/notebook.py", value)
}

func TestNormalizePath_requirementsFile(t *testing.T) {
	tmpDir := t.TempDir()
	location := dyn.Location{File: filepath.Join(tmpDir, "resources", "job_1.yml")}
	value, err := normalizePath("-r ../requirements.txt", location, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-r requirements.txt", value)

	value, err = normalizePath("-r      ../requirements.txt", location, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-r requirements.txt", value)
}

func TestNormalizePath_environmentDependency(t *testing.T) {
	tmpDir := t.TempDir()
	location := dyn.Location{File: filepath.Join(tmpDir, "resources", "job_1.yml")}
	value, err := normalizePath("-e ../file.py", location, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-e file.py", value)
}

func TestLocationDirectory(t *testing.T) {
	loc := dyn.Location{File: "file", Line: 1, Column: 2}
	dir, err := locationDirectory(loc)
	assert.NoError(t, err)
	assert.Equal(t, ".", dir)
}

func TestLocationDirectoryNoFile(t *testing.T) {
	loc := dyn.Location{}
	_, err := locationDirectory(loc)
	assert.Error(t, err)
}
