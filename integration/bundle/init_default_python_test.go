package bundle_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/python/pythontest"
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
	ctx, replacements := testcli.WithReplacementsMap(ctx)
	replacements.Set(uniqueProjectId, "$UNIQUE_PRJ")

	testcli.PrepareReplacements(t, replacements, wt.W)

	user, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)
	if user != nil {
		testcli.PrepareReplacementsUser(t, replacements, *user)
	}

	tmpDir1, pythonExe := pythontest.RequirePythonVENV(t, ctx, pythonVersion, true)
	extras, ok := extraInstalls[pythonVersion]
	if ok {
		args := append([]string{"pip", "install", "--python", pythonExe}, extras...)
		testutil.RunCommand(t, "uv", args...)
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
	err = os.WriteFile(filepath.Join(tmpDir1, "config.json"), b, 0o644)
	require.NoError(t, err)

	testcli.RequireOutput(t, ctx, []string{"bundle", "init", "default-python", "--config-file", "config.json"}, "testdata/default_python/bundle_init.txt")
	testutil.Chdir(t, projectName)

	testcli.RequireOutput(t, ctx, []string{"bundle", "validate"}, "testdata/default_python/bundle_validate.txt")

	testcli.RequireOutput(t, ctx, []string{"bundle", "deploy"}, "testdata/default_python/bundle_deploy.txt")
	t.Cleanup(func() {
		// Delete the stack
		testcli.RequireSuccessfulRun(t, ctx, "bundle", "destroy", "--auto-approve")
	})

	ignoredFields := []string{
		"/resources/jobs/project_name_$UNIQUE_PRJ_job/email_notifications",
		"/resources/jobs/project_name_$UNIQUE_PRJ_job/job_clusters/0/new_cluster/node_type_id",
		"/resources/jobs/project_name_$UNIQUE_PRJ_job/url",
		"/resources/pipelines/project_name_$UNIQUE_PRJ_pipeline/catalog",
		"/resources/pipelines/project_name_$UNIQUE_PRJ_pipeline/url",
		"/workspace/current_user",
	}

	testcli.RequireOutputJQ(t, ctx, []string{"bundle", "summary", "--output", "json"}, "testdata/default_python/bundle_summary.txt", ignoredFields)
}
