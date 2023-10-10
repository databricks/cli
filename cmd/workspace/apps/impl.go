package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/client"
)

// unexported type that holds implementations of just ServingEndpoints API methods
type appsImpl struct {
	client *client.DatabricksClient
}

func (a *appsImpl) Deploy(ctx context.Context, request *DeployAppRequest) (*DeploymentResponse, error) {
	var deploymentDetailed DeploymentResponse
	path := "/api/2.0/preview/apps/deployments"
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	err := a.client.Do(ctx, http.MethodPost, path, headers, request, &deploymentDetailed)
	return &deploymentDetailed, err
}

func (a *appsImpl) Delete(ctx context.Context, request *DeleteAppRequest) (*DeleteResponse, error) {
	var deleteResponse DeleteResponse
	path := fmt.Sprintf("/api/2.0/preview/apps/instances/%s", request.Name)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	err := a.client.Do(ctx, http.MethodDelete, path, headers, nil, &deleteResponse)
	return &deleteResponse, err
}

func (a *appsImpl) Get(ctx context.Context, request *GetAppRequest) (*GetResponse, error) {
	var getResponse GetResponse
	var path string
	if request.Name == "" {
		path = fmt.Print("/api/2.0/preview/apps/instances")
	} else {
		path = fmt.Sprintf("/api/2.0/preview/apps/instances/%s", request.Name)
	}
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	err := a.client.Do(ctx, http.MethodGet, path, headers, nil, &getResponse)
	return &getResponse, err
}
