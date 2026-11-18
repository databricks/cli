package appdeploy

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	sdkapps "github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler testserver.HandlerFunc) *databricks.WorkspaceClient {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/apps/{name}/deployments/{deployment_id}", handler)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	return w
}

func deploymentIn(state sdkapps.AppDeploymentState) testserver.HandlerFunc {
	return func(req testserver.Request) any {
		return sdkapps.AppDeployment{
			DeploymentId: req.Vars["deployment_id"],
			Status:       &sdkapps.AppDeploymentStatus{State: state, Message: "app exited unexpectedly"},
		}
	}
}

func inProgressApp() *sdkapps.App {
	return &sdkapps.App{
		Name: "myapp",
		ActiveDeployment: &sdkapps.AppDeployment{
			DeploymentId: "d1",
			Status:       &sdkapps.AppDeploymentStatus{State: sdkapps.AppDeploymentStateInProgress},
		},
	}
}

func TestWaitForDeploymentToCompleteIgnoresFailedDeployment(t *testing.T) {
	w := newTestClient(t, deploymentIn(sdkapps.AppDeploymentStateFailed))
	assert.NoError(t, WaitForDeploymentToComplete(cmdio.MockDiscard(t.Context()), w, inProgressApp()))
}

func TestWaitForDeploymentToCompleteSucceeded(t *testing.T) {
	w := newTestClient(t, deploymentIn(sdkapps.AppDeploymentStateSucceeded))
	assert.NoError(t, WaitForDeploymentToComplete(cmdio.MockDiscard(t.Context()), w, inProgressApp()))
}

func TestWaitForDeploymentToCompleteReturnsGetError(t *testing.T) {
	w := newTestClient(t, func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "NOT_FOUND", "message": "deployment not found"},
		}
	})
	err := WaitForDeploymentToComplete(cmdio.MockDiscard(t.Context()), w, inProgressApp())
	assert.ErrorContains(t, err, "deployment not found")
}
