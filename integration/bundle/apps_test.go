package bundle_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccDeployBundleWithApp(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	uniqueId := uuid.New().String()
	appId := fmt.Sprintf("app-%s", uuid.New().String()[0:8])
	nodeTypeId := internal.GetNodeTypeId(env.Get(ctx, "CLOUD_ENV"))
	instancePoolId := env.Get(ctx, "TEST_INSTANCE_POOL_ID")

	root, err := initTestTemplate(t, ctx, "apps", map[string]any{
		"unique_id":        uniqueId,
		"app_id":           appId,
		"node_type_id":     nodeTypeId,
		"spark_version":    defaultSparkVersion,
		"instance_pool_id": instancePoolId,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, root)
		require.NoError(t, err)

		app, err := wt.W.Apps.Get(ctx, apps.GetAppRequest{Name: "test-app"})
		if err != nil {
			require.ErrorContains(t, err, "does not exist")
		} else {
			require.Contains(t, []apps.ApplicationState{apps.ApplicationStateUnavailable}, app.AppStatus.State)
		}
	})

	err = deployBundle(t, ctx, root)
	require.NoError(t, err)

	// App should exists after bundle deployment
	app, err := wt.W.Apps.Get(ctx, apps.GetAppRequest{Name: appId})
	require.NoError(t, err)
	require.NotNil(t, app)

	// Try to run the app
	_, out, err := runResourceWithStderr(t, ctx, root, "test_app")
	require.NoError(t, err)
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
	_, out, err = runResourceWithStderr(t, ctx, root, "test_app")
	require.NoError(t, err)
	require.Contains(t, out, app.Url)

	// App should be in the running state
	app, err = wt.W.Apps.Get(ctx, apps.GetAppRequest{Name: appId})
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, apps.ApplicationStateRunning, app.AppStatus.State)
}
