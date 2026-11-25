package appdeploy

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/retry"
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

const rejectedStartMessage = "Cannot start app test-app as its compute is in ACTIVE state. App compute needs to be STOPPED to start."

// newStartTestClient serves a start that is always rejected with 400 and GET /apps/{name} from get.
func newStartTestClient(t *testing.T, get testserver.HandlerFunc) (*databricks.WorkspaceClient, *int) {
	server := testserver.New(t)
	startCalls := 0
	server.Handle("POST", "/api/2.0/apps/{name}/start", func(req testserver.Request) any {
		startCalls++
		return testserver.Response{
			StatusCode: 400,
			Body:       map[string]string{"error_code": "BAD_REQUEST", "message": rejectedStartMessage},
		}
	})
	server.Handle("GET", "/api/2.0/apps/{name}", get)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	return w, &startCalls
}

// A start sent after the caller last saw the compute stopped (a retried deploy, or a concurrent
// start) is rejected. An ACTIVE compute makes that rejection a success, and a STARTING one is
// waited for; any other state keeps the error.
func TestStart_RejectedStart(t *testing.T) {
	tests := []struct {
		state   sdkapps.ComputeState
		wantErr bool
	}{
		{state: sdkapps.ComputeStateActive, wantErr: false},
		{state: sdkapps.ComputeStateStarting, wantErr: false},
		{state: sdkapps.ComputeStateStopped, wantErr: true},
		{state: sdkapps.ComputeStateDeleting, wantErr: true},
		{state: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			getCalls := 0
			w, startCalls := newStartTestClient(t, func(req testserver.Request) any {
				getCalls++
				app := sdkapps.App{Name: req.Vars["name"]}
				state := tt.state
				// A STARTING compute becomes ACTIVE on the next poll.
				if state == sdkapps.ComputeStateStarting && getCalls > 1 {
					state = sdkapps.ComputeStateActive
				}
				if state != "" {
					app.ComputeStatus = &sdkapps.ComputeStatus{State: state}
				}
				return app
			})

			app, err := Start(t.Context(), w, "test-app", nil)
			assert.Equal(t, 1, *startCalls)
			if tt.wantErr {
				require.ErrorContains(t, err, "Cannot start app test-app")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, sdkapps.ComputeStateActive, app.ComputeStatus.State)
		})
	}
}

// A failed read after a rejected start must stay retryable: the 400 alone is not transient.
func TestStart_RejectedStartReadFailureIsTransient(t *testing.T) {
	w, _ := newStartTestClient(t, func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: 500,
			Body:       map[string]string{"error_code": "INTERNAL_ERROR", "message": "boom"},
		}
	})

	_, err := Start(t.Context(), w, "test-app", nil)
	require.ErrorContains(t, err, "Cannot start app test-app")
	require.ErrorContains(t, err, "boom")
	assert.True(t, retry.IsTransient(t.Context(), err))
}
