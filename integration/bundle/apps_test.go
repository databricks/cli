package bundle_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeployBundleWithApp(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	// TODO: should only skip app run when app can be created with no_compute option.
	if testing.Short() {
		t.Log("Skip the app creation and run in short mode")
		return
	}

	if testutil.GetCloud(t) == testutil.GCP {
		t.Skip("Skipping test for GCP cloud because /api/2.0/apps is temporarily unavailable there.")
	}

	uniqueId := uuid.New().String()
	appId := fmt.Sprintf("app-%s", uuid.New().String()[0:8])
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

	deployBundle(t, ctx, root)

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

	job, err := wt.W.Jobs.GetBySettingsName(ctx, fmt.Sprintf("test-job-with-cluster-%s", uniqueId))
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
