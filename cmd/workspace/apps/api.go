package apps

import (
	"context"

	"github.com/databricks/databricks-sdk-go/client"
)

type AppsAPI struct {
	impl AppsService
}

func NewApps(client *client.DatabricksClient) *AppsAPI {
	return &AppsAPI{
		impl: &appsImpl{
			client: client,
		},
	}
}

// WithImpl could be used to override low-level API implementations for unit
// testing purposes with [github.com/golang/mock] or other mocking frameworks.
func (a *AppsAPI) WithImpl(impl AppsService) *AppsAPI {
	a.impl = impl
	return a
}

// Impl returns low-level ServingEndpoints API implementation
func (a *AppsAPI) Impl() AppsService {
	return a.impl
}

// Create a new serving endpoint.
func (a *AppsAPI) Deploy(ctx context.Context, deployApp *DeployAppRequest) (*DeploymentResponse, error) {
	deploymentDetailed, err := a.impl.Deploy(ctx, deployApp)
	if err != nil {
		return nil, err
	}
	return deploymentDetailed, nil
}

func (a *AppsAPI) Delete(ctx context.Context, deleteApp *DeleteAppRequest) (*DeleteResponse, error) {
	deploymentDetailed, err := a.impl.Delete(ctx, deleteApp)
	if err != nil {
		return nil, err
	}
	return deploymentDetailed, nil
}

func (a *AppsAPI) Get(ctx context.Context, listApp *GetAppRequest) (*GetResponse, error) {
	appsList, err := a.impl.Get(ctx, listApp)
	if err != nil {
		return nil, err
	}
	return appsList, nil
}
