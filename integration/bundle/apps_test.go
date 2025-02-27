package bundle_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeployBundleWithApp(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	if testutil.GetCloud(t) == testutil.GCP {
		t.Skip("Skipping test for GCP cloud because /api/2.0/apps is temporarily unavailable there.")
	}

	uniqueId := uuid.New().String()
	appId := "app-" + uuid.New().String()[0:8]
	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	instancePoolId := env.Get(ctx, "TEST_INSTANCE_POOL_ID")

	root := initTestTemplate(t, ctx, "apps", map[string]any{
		"unique_id":        uniqueId,
		"app_id":           appId,
		"node_type_id":     nodeTypeId,
		"spark_version":    defaultSparkVersion,
		"instance_pool_id": instancePoolId,
	})

	t.Cleanup(func() {
		destroyBundle(t, ctx, root)
		app, err := wt.W.Apps.Get(ctx, apps.GetAppRequest{Name: "test-app"})
		if err != nil {
			require.ErrorContains(t, err, "does not exist")
		} else {
			require.Contains(t, []apps.ApplicationState{apps.ApplicationStateUnavailable}, app.AppStatus.State)
		}
	})

	ctx, replacements := testdiff.WithReplacementsMap(ctx)
	replacements.Set(uniqueId, "$UNIQUE_PRJ")

	user, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)
	require.NotNil(t, user)
	testdiff.PrepareReplacementsUser(t, replacements, *user)
	testdiff.PrepareReplacementsWorkspaceClient(t, replacements, wt.W)
	testdiff.PrepareReplacementsUUID(t, replacements)
	testdiff.PrepareReplacementsNumber(t, replacements)
	testdiff.PrepareReplacementsTemporaryDirectory(t, replacements)

	testutil.Chdir(t, root)
	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "validate"},
		testutil.TestData("testdata/apps/bundle_validate.txt"),
	)
	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "deploy", "--force-lock", "--auto-approve"},
		testutil.TestData("testdata/apps/bundle_deploy.txt"),
	)

	// App should exists after bundle deployment
	app, err := wt.W.Apps.Get(ctx, apps.GetAppRequest{Name: appId})
	require.NoError(t, err)
	require.NotNil(t, app)

	// Check app config
	currentUser, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	pathToAppYml := fmt.Sprintf("/Workspace/Users/%s/.bundle/%s/files/app/app.yml", currentUser.UserName, uniqueId)
	reader, err := wt.W.Workspace.Download(ctx, pathToAppYml)
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)

	job, err := wt.W.Jobs.GetBySettingsName(ctx, "test-job-with-cluster-"+uniqueId)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, fmt.Sprintf(`command:
  - flask
  - --app
  - app
  - run
env:
  - name: JOB_ID
    value: "%d"`, job.JobId))

	// Redeploy bundle with changed config env for app and confirm it's updated in app.yaml
	deployBundleWithArgs(t, ctx, root, `--var="env_var_name=ANOTHER_JOB_ID"`, "--force-lock", "--auto-approve")
	reader, err = wt.W.Workspace.Download(ctx, pathToAppYml)
	require.NoError(t, err)

	data, err = io.ReadAll(reader)
	require.NoError(t, err)

	content = string(data)
	require.Contains(t, content, fmt.Sprintf(`command:
  - flask
  - --app
  - app
  - run
env:
  - name: ANOTHER_JOB_ID
    value: "%d"`, job.JobId))

	if testing.Short() {
		t.Log("Skip the app run in short mode")
		return
	}

	// Try to run the app
	_, out := runResourceWithStderr(t, ctx, root, "test_app")
	require.Contains(t, out, app.Url)

	// App should be in the running state
	app, err = wt.W.Apps.Get(ctx, apps.GetAppRequest{Name: appId})
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, apps.ApplicationStateRunning, app.AppStatus.State)

	// Stop the app
	wait, err := wt.W.Apps.Stop(ctx, apps.StopAppRequest{Name: appId})
	require.NoError(t, err)
	app, err = wait.Get()
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, apps.ApplicationStateUnavailable, app.AppStatus.State)

	// Try to run the app again
	_, out = runResourceWithStderr(t, ctx, root, "test_app")
	require.Contains(t, out, app.Url)

	// App should be in the running state
	app, err = wt.W.Apps.Get(ctx, apps.GetAppRequest{Name: appId})
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, apps.ApplicationStateRunning, app.AppStatus.State)

	// Redeploy it again just to check that it can be redeployed
	deployBundle(t, ctx, root)
}
