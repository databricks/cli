package bundle_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/python/pythontest"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/require"
)

var pythonVersions = []string{
	"3.8",
	"3.9",
	"3.10",
	"3.11",
	"3.12",
	"3.13",
}

var pythonVersionsShort = []string{
	"3.9",
	"3.12",
}

var extraInstalls = map[string][]string{
	"3.12": {"setuptools"},
	"3.13": {"setuptools"},
}

func TestDefaultPython(t *testing.T) {
	versions := pythonVersions
	if testing.Short() {
		versions = pythonVersionsShort
	}

	for _, pythonVersion := range versions {
		t.Run(pythonVersion, func(t *testing.T) {
			testDefaultPython(t, pythonVersion)
		})
	}
}

func testDefaultPython(t *testing.T, pythonVersion string) {
	ctx, wt := acc.WorkspaceTest(t)

	uniqueProjectId := testutil.RandomName("")
	ctx, replacements := testdiff.WithReplacementsMap(ctx)
	replacements.Set(uniqueProjectId, "$UNIQUE_PRJ")

	user, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)
	require.NotNil(t, user)
	testdiff.PrepareReplacementsUser(t, replacements, *user)
	testdiff.PrepareReplacementsWorkspaceClient(t, replacements, wt.W)
	testdiff.PrepareReplacementsUUID(t, replacements)
	testdiff.PrepareReplacementsNumber(t, replacements)
	testdiff.PrepareReplacementsTemporaryDirectory(t, replacements)

	tmpDir := t.TempDir()
	testutil.Chdir(t, tmpDir)

	opts := pythontest.VenvOpts{
		PythonVersion: pythonVersion,
		Dir:           tmpDir,
	}

	pythontest.RequireActivatedPythonEnv(t, ctx, &opts)
	extras, ok := extraInstalls[pythonVersion]
	if ok {
		args := append([]string{"pip", "install", "--python", opts.PythonExe}, extras...)
		cmd := exec.Command("uv", args...)
		require.NoError(t, cmd.Run())
	}

	projectName := "project_name_" + uniqueProjectId

	initConfig := map[string]string{
		"project_name":     projectName,
		"include_notebook": "yes",
		"include_python":   "yes",
		"include_dlt":      "yes",
	}
	b, err := json.Marshal(initConfig)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "config.json"), b, 0o644)
	require.NoError(t, err)

	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "init", "default-python", "--config-file", "config.json"},
		testutil.TestData("testdata/default_python/bundle_init.txt"),
	)
	testutil.Chdir(t, projectName)

	t.Cleanup(func() {
		// Delete the stack
		testcli.RequireSuccessfulRun(t, ctx, "bundle", "destroy", "--auto-approve")
	})

	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "validate"},
		testutil.TestData("testdata/default_python/bundle_validate.txt"),
	)
	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "deploy"},
		testutil.TestData("testdata/default_python/bundle_deploy.txt"),
	)

	testcli.AssertOutputJQ(
		t,
		ctx,
		[]string{"bundle", "summary", "--output", "json"},
		testutil.TestData("testdata/default_python/bundle_summary.txt"),
		[]string{
			"/bundle/terraform/exec_path",
			"/resources/jobs/project_name_$UNIQUE_PRJ_job/email_notifications",
			"/resources/jobs/project_name_$UNIQUE_PRJ_job/job_clusters/0/new_cluster/node_type_id",
			"/resources/jobs/project_name_$UNIQUE_PRJ_job/url",
			"/resources/pipelines/project_name_$UNIQUE_PRJ_pipeline/catalog",
			"/resources/pipelines/project_name_$UNIQUE_PRJ_pipeline/url",
			"/workspace/current_user",
		},
	)
}
