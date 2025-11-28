package deploy

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWaitForAppsDeletion(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						App: apps.App{
							Name: "my_app",
						},
					},
				},
			},
		},
	}

	mwc := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(mwc.WorkspaceClient)

	appApi := mwc.GetMockAppsAPI()

	appApi.EXPECT().GetByName(mock.Anything, "my_app").Return(&apps.App{
		Name: "my_app",
		ComputeStatus: &apps.ComputeStatus{
			State: apps.ComputeStateDeleting,
		},
	}, nil).Once()

	appApi.EXPECT().GetByName(mock.Anything, "my_app").Return(nil, &apierr.APIError{
		StatusCode: 404,
		Message:    "App not found",
	}).Once()

	err := waitForAppsDeletion(ctx, b)
	require.NoError(t, err)
}

func TestWaitForAppsDeletion_NoApps(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{},
	}

	err := waitForAppsDeletion(ctx, b)
	require.NoError(t, err)
}

func TestWaitForAppsDeletion_AppNotDeleting(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						App: apps.App{
							Name: "my_app",
						},
					},
				},
			},
		},
	}

	mwc := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(mwc.WorkspaceClient)

	appApi := mwc.GetMockAppsAPI()

	appApi.EXPECT().GetByName(mock.Anything, "my_app").Return(&apps.App{
		Name: "my_app",
		ComputeStatus: &apps.ComputeStatus{
			State: apps.ComputeStateActive,
		},
	}, nil).Once()

	err := waitForAppsDeletion(ctx, b)
	require.NoError(t, err)
}

func TestPrepareEnvironment(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						App: apps.App{
							Name: "my_app",
						},
					},
				},
			},
		},
	}

	mwc := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(mwc.WorkspaceClient)

	appApi := mwc.GetMockAppsAPI()

	appApi.EXPECT().GetByName(mock.Anything, "my_app").Return(&apps.App{
		Name: "my_app",
		ComputeStatus: &apps.ComputeStatus{
			State: apps.ComputeStateActive,
		},
	}, nil).Once()

	m := PrepareEnvironment()
	require.Equal(t, "deploy:prepare-environment", m.Name())

	diags := m.Apply(ctx, b)
	require.Empty(t, diags)
}
