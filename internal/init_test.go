package internal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBundleInitErrorOnUnknownFields(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	_, _, err := RequireErrorRun(t, "bundle", "init", "./testdata/init/field-does-not-exist", "--output-dir", tmpDir)
	assert.EqualError(t, err, "failed to compute file content for bar.tmpl. variable \"does_not_exist\" not defined")
}

// This test tests the MLOps Stacks DAB e2e and thus there's a couple of special
// considerations to take note of:
//
//  1. Upstream changes to the MLOps Stacks DAB can cause this test to fail.
//     In which case we should do one of:
//     (a) Update this test to reflect the changes
//     (b) Update the MLOps Stacks DAB to not break this test. Skip this test
//     temporarily until the MLOps Stacks DAB is updated
//
//  2. While rare and to be avoided if possible, the CLI reserves the right to
//     make changes that can break the MLOps Stacks DAB. In which case we should
//     skip this test until the MLOps Stacks DAB is updated to work again.
func TestAccBundleInitOnMlopsStacks(t *testing.T) {
	env := GetEnvOrSkipTest(t, "CLOUD_ENV")
	if env == "gcp" {
		t.Skip("MLOps Stacks is not supported in GCP")
	}
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	// Create a config file with the project name and root dir
	initConfig := map[string]string{
		"input_project_name":                    "project_name",
		"input_root_dir":                        "repo_name",
		"input_include_models_in_unity_catalog": "no",
		"input_cloud":                           env,
	}
	b, err := json.Marshal(initConfig)
	require.NoError(t, err)
	os.WriteFile(filepath.Join(tmpDir1, "config.json"), b, 0644)

	// Run bundle init
	assert.NoFileExists(t, filepath.Join(tmpDir2, "repo_name", "project_name", "README.md"))
	RequireSuccessfulRun(t, "bundle", "init", "mlops-stacks", "--output-dir", tmpDir2, "--config-file", filepath.Join(tmpDir1, "config.json"))

	// Assert that the README.md file was created
	assert.FileExists(t, filepath.Join(tmpDir2, "repo_name", "project_name", "README.md"))
	assertLocalFileContents(t, filepath.Join(tmpDir2, "repo_name", "project_name", "README.md"), "This directory contains python code, notebooks and ML asset configs related to one ML project.")
	assertLocalFileContents(t, filepath.Join(tmpDir2, "repo_name", "project_name", "README.md"), "# project_name")

	// Validate the stack
	testutil.Chdir(t, filepath.Join(tmpDir2, "repo_name", "project_name"))
	RequireSuccessfulRun(t, "bundle", "validate")

	// Deploy the stack
	RequireSuccessfulRun(t, "bundle", "deploy")
	t.Cleanup(func() {
		// Delete the stack
		RequireSuccessfulRun(t, "bundle", "destroy", "--auto-approve")
	})

	// Get summary of the bundle deployment
	stdout, _ := RequireSuccessfulRun(t, "bundle", "summary", "--output", "json")
	summary := &config.Root{}
	err = json.Unmarshal(stdout.Bytes(), summary)
	require.NoError(t, err)

	// Assert resource Ids are not empty
	assert.NotEmpty(t, summary.Resources.Experiments["experiment"].ID)
	assert.NotEmpty(t, summary.Resources.Models["model"].ID)
	assert.NotEmpty(t, summary.Resources.Jobs["batch_inference_job"].ID)
	assert.NotEmpty(t, summary.Resources.Jobs["model_training_job"].ID)

	// Assert the batch inference job actually exists
	batchJobId, err := strconv.ParseInt(summary.Resources.Jobs["batch_inference_job"].ID, 10, 64)
	require.NoError(t, err)
	job, err := w.Jobs.GetByJobId(context.Background(), batchJobId)
	assert.NoError(t, err)
	assert.Equal(t, "dev-project_name-batch-inference-job", job.Settings.Name)
}
