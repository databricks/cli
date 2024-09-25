package config_tests

import (
	"os"
	"os/exec"
	pathlib "path"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/stretchr/testify/assert"
)

func TestDataclassNoWheel(t *testing.T) {
	activateVEnv(t)
	setPythonPath(t, "python_import/dataclass_no_wheel/src")

	expected := &resources.Job{
		JobSettings: &jobs.JobSettings{
			Name: "Test Job",
			JobClusters: []jobs.JobCluster{
				{
					JobClusterKey: "my_cluster",
				},
			},
			Tasks: []jobs.Task{
				{
					NotebookTask: &jobs.NotebookTask{
						NotebookPath: "notebooks/my_notebook.py",
					},
					JobClusterKey: "my_cluster",
					TaskKey:       "my_notebook_task",
				},
			},
		},
	}

	b := load(t, "./python_import/dataclass_no_wheel")

	assert.Equal(t, []string{"my_job"}, maps.Keys(b.Config.Resources.Jobs))

	myJob := b.Config.Resources.Jobs["my_job"]
	assert.Equal(t, expected, myJob)

	// NewCluster is reference to a variable and needs to be checked separately
	err := b.Config.Mutate(func(value dyn.Value) (dyn.Value, error) {
		path := dyn.MustPathFromString("resources.jobs.my_job.job_clusters[0].new_cluster")
		value, err := dyn.GetByPath(value, path)
		if err != nil {
			return dyn.InvalidValue, err
		}

		assert.Equal(t, "${var.default_cluster_spec}", value.AsAny())

		return value, nil
	})
	require.NoError(t, err)
}

func setPythonPath(t *testing.T, path string) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	t.Setenv("PYTHONPATH", pathlib.Join(wd, path))
}

func activateVEnv(t *testing.T) {
	dir := t.TempDir()
	venvDir := pathlib.Join(dir, "venv")

	err := exec.Command("python3", "-m", "venv", venvDir).Run()
	require.NoError(t, err)

	// we don't have shell to activate venv, updating PATH is enough

	var venvBinDir string
	if runtime.GOOS == "Windows" {
		venvBinDir = pathlib.Join(venvDir, "Scripts")
		t.Setenv("PATH", venvBinDir+";"+os.Getenv("PATH"))
	} else {
		venvBinDir = pathlib.Join(venvDir, "bin")
		t.Setenv("PATH", venvBinDir+":"+os.Getenv("PATH"))
	}

	err = exec.Command(
		pathlib.Join(venvBinDir, "pip"),
		"install",
		"databricks-pydabs==0.5.1",
	).Run()
	require.NoError(t, err)
}
