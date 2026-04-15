package appdeploy

import (
	"testing"

	sdkapps "github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
)

func TestWaitForDeploymentToCompleteNilStatus(t *testing.T) {
	ctx := t.Context()

	// ActiveDeployment with nil Status should not panic.
	app := &sdkapps.App{
		ActiveDeployment: &sdkapps.AppDeployment{
			Status: nil,
		},
		PendingDeployment: &sdkapps.AppDeployment{
			Status: nil,
		},
	}
	err := WaitForDeploymentToComplete(ctx, nil, app)
	assert.NoError(t, err)
}

func TestWaitForDeploymentToCompleteNilDeployments(t *testing.T) {
	ctx := t.Context()

	app := &sdkapps.App{}
	err := WaitForDeploymentToComplete(ctx, nil, app)
	assert.NoError(t, err)
}

func TestWaitForDeploymentToCompleteNonInProgress(t *testing.T) {
	ctx := t.Context()

	// Status with a non-InProgress state should not wait.
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
	err := WaitForDeploymentToComplete(ctx, nil, app)
	assert.NoError(t, err)
}
