package appdeploy_test

import (
	"testing"

	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	sdkapps "github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForDeploymentToCompleteNilStatus(t *testing.T) {
	server := testserver.New(t)

	getDeploymentCalled := 0
	server.Handle("GET", "/api/2.0/apps/{appName}/deployments/{deploymentId}", func(req testserver.Request) any {
		getDeploymentCalled++
		return sdkapps.AppDeployment{
			DeploymentId: req.Vars["deploymentId"],
			Status: &sdkapps.AppDeploymentStatus{
				State: sdkapps.AppDeploymentStateSucceeded,
			},
		}
	})

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())

	app := &sdkapps.App{
		Name: "test-app",
		ActiveDeployment: &sdkapps.AppDeployment{
			DeploymentId: "dep-1",
			Status:       nil,
		},
		PendingDeployment: &sdkapps.AppDeployment{
			DeploymentId: "dep-2",
			Status:       nil,
		},
	}
	err = appdeploy.WaitForDeploymentToComplete(ctx, client, app)
	require.NoError(t, err)
	assert.Equal(t, 2, getDeploymentCalled)
}

func TestWaitForDeploymentToCompleteNilDeployments(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	app := &sdkapps.App{}
	err := appdeploy.WaitForDeploymentToComplete(ctx, nil, app)
	assert.NoError(t, err)
}

func TestWaitForDeploymentToCompleteNonInProgress(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	app := &sdkapps.App{
		ActiveDeployment: &sdkapps.AppDeployment{
			Status: &sdkapps.AppDeploymentStatus{
				State: sdkapps.AppDeploymentStateSucceeded,
			},
		},
		PendingDeployment: &sdkapps.AppDeployment{
			Status: &sdkapps.AppDeploymentStatus{
				State: sdkapps.AppDeploymentStateFailed,
			},
		},
	}
	err := appdeploy.WaitForDeploymentToComplete(ctx, nil, app)
	assert.NoError(t, err)
}
