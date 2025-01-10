package bundle_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleInitErrorOnUnknownFields(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	_, _, err := testcli.RequireErrorRun(t, ctx, "bundle", "init", "./testdata/init/field-does-not-exist", "--output-dir", tmpDir)
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
func TestBundleInitOnMlopsStacks(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	projectName := testutil.RandomName("project_name_")
	env := testutil.GetCloud(t).String()

	// Create a config file with the project name and root dir
	initConfig := map[string]string{
		"input_project_name":                    projectName,
		"input_root_dir":                        "repo_name",
		"input_include_models_in_unity_catalog": "no",
		"input_cloud":                           strings.ToLower(env),
	}
	b, err := json.Marshal(initConfig)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir1, "config.json"), b, 0o644)
	require.NoError(t, err)

	// Run bundle init
	assert.NoFileExists(t, filepath.Join(tmpDir2, "repo_name", projectName, "README.md"))
	testcli.RequireSuccessfulRun(t, ctx, "bundle", "init", "mlops-stacks", "--output-dir", tmpDir2, "--config-file", filepath.Join(tmpDir1, "config.json"))

	// Assert that the README.md file was created
	contents := testutil.ReadFile(t, filepath.Join(tmpDir2, "repo_name", projectName, "README.md"))
	assert.Contains(t, contents, "# "+projectName)

	// Validate the stack
	testutil.Chdir(t, filepath.Join(tmpDir2, "repo_name", projectName))
	testcli.RequireSuccessfulRun(t, ctx, "bundle", "validate")

	// Deploy the stack
	testcli.RequireSuccessfulRun(t, ctx, "bundle", "deploy")
	t.Cleanup(func() {
		// Delete the stack
		testcli.RequireSuccessfulRun(t, ctx, "bundle", "destroy", "--auto-approve")
	})

	// Get summary of the bundle deployment
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "bundle", "summary", "--output", "json")
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
	assert.Contains(t, job.Settings.Name, fmt.Sprintf("dev-%s-batch-inference-job", projectName))
}

func TestBundleInitHelpers(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	var smallestNode string
	switch testutil.GetCloud(t) {
	case testutil.Azure:
		smallestNode = "Standard_D3_v2"
	case testutil.GCP:
		smallestNode = "n1-standard-4"
	case testutil.AWS:
		smallestNode = "i3.xlarge"
	default:
		t.Fatal("Unknown cloud environment")
	}

	tests := []struct {
		funcName string
		expected string
	}{
		{
			funcName: "{{short_name}}",
			expected: iamutil.GetShortUserName(me),
		},
		{
			funcName: "{{user_name}}",
			expected: me.UserName,
		},
		{
			funcName: "{{workspace_host}}",
			expected: w.Config.Host,
		},
		{
			funcName: "{{is_service_principal}}",
			expected: strconv.FormatBool(iamutil.IsServicePrincipal(me)),
		},
		{
			funcName: "{{smallest_node_type}}",
			expected: smallestNode,
		},
	}

	for _, test := range tests {
		// Setup template to test the helper function.
		tmpDir := t.TempDir()
		tmpDir2 := t.TempDir()

		err := os.Mkdir(filepath.Join(tmpDir, "template"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "template", "foo.txt.tmpl"), []byte(test.funcName), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "databricks_template_schema.json"), []byte("{}"), 0o644)
		require.NoError(t, err)

		// Run bundle init.
		testcli.RequireSuccessfulRun(t, ctx, "bundle", "init", tmpDir, "--output-dir", tmpDir2)

		// Assert that the helper function was correctly computed.
		contents := testutil.ReadFile(t, filepath.Join(tmpDir2, "foo.txt"))
		assert.Contains(t, contents, test.expected)
	}
}
