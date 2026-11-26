package mutator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
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
	location := diag.Location{File: filepath.Join(tmpDir, "resources", "job_1.yml")}
	path := dyn.MustPathFromString("resources.jobs.job1.tasks[0].notebook_task.notebook_path")
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPath(v, path, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			return dyn.NewValue(value.MustString(), []diag.Location{location}), nil
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
							TriggersState: nil,
						},
					},
				},
			},
		},
		BundleRootPath: tmpDir,
	}

	location := diag.Location{File: filepath.Join(tmpDir, "resources", "run.yml")}
	path := dyn.MustPathFromString("resources.job_runs.run1.lifecycle.triggers[0].on_file_change")
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPath(v, path, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
			return dyn.NewValue(value.MustString(), []diag.Location{location}), nil
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
	value, err := normalizePath("/notebook.py", "", errors.New("no file"), "/tmp")
	assert.NoError(t, err)
	assert.Equal(t, "/notebook.py", value)
}

func TestNormalizePath_url(t *testing.T) {
	value, err := normalizePath("s3:///path/to/notebook.py", "", errors.New("no file"), "/tmp")
	assert.NoError(t, err)
	assert.Equal(t, "s3:///path/to/notebook.py", value)
}

func TestNormalizePath_requirementsFile(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "resources")
	value, err := normalizePath("-r ../requirements.txt", dir, nil, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-r requirements.txt", value)

	value, err = normalizePath("-r      ../requirements.txt", dir, nil, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-r requirements.txt", value)
}

func TestNormalizePath_environmentDependency(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "resources")
	value, err := normalizePath("-e ../file.py", dir, nil, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "-e file.py", value)
}

func TestLocationDirectory(t *testing.T) {
	loc := diag.Location{File: "file", Line: 1, Column: 2}
	dir, err := locationDirectory(loc)
	assert.NoError(t, err)
	assert.Equal(t, ".", dir)
}

func TestLocationDirectoryNoFile(t *testing.T) {
	loc := diag.Location{}
	_, err := locationDirectory(loc)
	assert.Error(t, err)
}

func TestNormalizePath_noDirectory(t *testing.T) {
	dir, dirErr := baseDirectory(diag.Location{}, "/tmp/.databricks/bundle/dev/variable-overrides.json", "/tmp")
	_, err := normalizePath("notebook.py", dir, dirErr, "/tmp")
	assert.EqualError(t, err, "unable to determine directory for a value at notebook.py: no file in location")
}

func TestBaseDirectory(t *testing.T) {
	root := t.TempDir()
	overridesFile := filepath.Join(root, ".databricks", "bundle", "dev", "variable-overrides.json")

	dir, err := baseDirectory(diag.Location{File: overridesFile}, overridesFile, root)
	require.NoError(t, err)
	assert.Equal(t, root, dir)

	dir, err = baseDirectory(diag.Location{File: filepath.Join(root, "resources", "job.yml")}, overridesFile, root)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "resources"), dir)

	_, err = baseDirectory(diag.Location{}, overridesFile, root)
	assert.Error(t, err)
}

// Values read from the overrides file must carry the exact path NormalizePaths matches on.
func TestBaseDirectory_valueFromOverridesFile(t *testing.T) {
	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
		Config:         config.Root{Bundle: config.Bundle{Target: "dev"}},
	}
	overridesFile := variableOverridesFilePath(b)
	require.NoError(t, os.MkdirAll(filepath.Dir(overridesFile), 0o700))
	require.NoError(t, os.WriteFile(overridesFile, []byte(`{"task": {"python_file": "./src/main.py"}}`), 0o600))

	values, diags := readVariablesFromFile(b)
	require.NoError(t, diags.Error())
	v, err := dyn.GetByPath(values, dyn.MustPathFromString("task.python_file"))
	require.NoError(t, err)

	dir, err := baseDirectory(v.Location(), overridesFile, root)
	require.NoError(t, err)
	assert.Equal(t, root, dir)
}
