package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newInitTerraform builds a *Terraform wired to a fakeRunner so Init can be
// exercised without a real terraform binary install.
func newInitTerraform(t *testing.T) (*Terraform, *fakeRunner) {
	t.Helper()
	root := t.TempDir()
	workingDir := filepath.Join(root, ".databricks", "ucm", "dev", "terraform")
	require.NoError(t, os.MkdirAll(workingDir, 0o700))

	runner := &fakeRunner{}
	tf := &Terraform{
		ExecPath:      "/stub/terraform",
		WorkingDir:    workingDir,
		Env:           map[string]string{"DATABRICKS_HOST": "https://example.cloud.databricks.com"},
		runnerFactory: newFakeRunnerFactory(runner),
	}
	return tf, runner
}

func TestInitRendersMainTfJson(t *testing.T) {
	u, _ := newRenderUcm(t)
	tf, runner := newInitTerraform(t)
	tf.WorkingDir, _ = WorkingDir(u)

	require.NoError(t, tf.Init(t.Context(), u))

	_, err := os.Stat(filepath.Join(tf.WorkingDir, MainConfigFileName))
	require.NoError(t, err, "main.tf.json should exist after Init")
	assert.Equal(t, 1, runner.InitCalls)
	assert.Equal(t, 1, runner.SetEnvCalls)
	assert.Equal(t, tf.Env, runner.LastEnv)
}

func TestInitIsIdempotent(t *testing.T) {
	u, _ := newRenderUcm(t)
	tf, runner := newInitTerraform(t)
	tf.WorkingDir, _ = WorkingDir(u)

	require.NoError(t, tf.Init(t.Context(), u))
	require.NoError(t, tf.Init(t.Context(), u))

	assert.Equal(t, 1, runner.InitCalls, "second Init should skip the underlying terraform init")
	// SetEnv is called once, when the runner is first bound.
	assert.Equal(t, 1, runner.SetEnvCalls)
}

func TestInitPropagatesRunnerError(t *testing.T) {
	u, _ := newRenderUcm(t)
	tf, runner := newInitTerraform(t)
	tf.WorkingDir, _ = WorkingDir(u)
	runner.InitErr = assert.AnError

	err := tf.Init(t.Context(), u)
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}
