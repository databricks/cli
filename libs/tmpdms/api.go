package tmpdms

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

const basePath = "/api/2.0/bundle"

// DeploymentMetadataAPI is a client for the Deployment Metadata Service.
//
// This is a temporary implementation that will be replaced by the SDK-generated
// client once the proto definitions land in the Go SDK. The method signatures
// and types are designed to match what the SDK will generate, so migration
// should be a straightforward import path change.
type DeploymentMetadataAPI struct {
	api *client.DatabricksClient
}

func NewDeploymentMetadataAPI(w *databricks.WorkspaceClient) (*DeploymentMetadataAPI, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment metadata API client: %w", err)
	}
	return &DeploymentMetadataAPI{api: apiClient}, nil
}

func (a *DeploymentMetadataAPI) CreateDeployment(ctx context.Context, request CreateDeploymentRequest) (*Deployment, error) {
	var resp Deployment
	path := basePath + "/deployments"
	query := map[string]any{"deployment_id": request.DeploymentID}
	err := a.api.Do(ctx, http.MethodPost, path, nil, query, request.Deployment, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) GetDeployment(ctx context.Context, request GetDeploymentRequest) (*Deployment, error) {
	var resp Deployment
	path := fmt.Sprintf("%s/deployments/%s", basePath, request.DeploymentID)
	err := a.api.Do(ctx, http.MethodGet, path, nil, nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) DeleteDeployment(ctx context.Context, request DeleteDeploymentRequest) (*Deployment, error) {
	var resp Deployment
	path := fmt.Sprintf("%s/deployments/%s", basePath, request.DeploymentID)
	err := a.api.Do(ctx, http.MethodDelete, path, nil, nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) CreateVersion(ctx context.Context, request CreateVersionRequest) (*Version, error) {
	var resp Version
	path := fmt.Sprintf("%s/deployments/%s/versions", basePath, request.DeploymentID)
	query := map[string]any{"version_id": request.VersionID}
	err := a.api.Do(ctx, http.MethodPost, path, nil, query, request.Version, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) GetVersion(ctx context.Context, request GetVersionRequest) (*Version, error) {
	var resp Version
	path := fmt.Sprintf("%s/deployments/%s/versions/%s", basePath, request.DeploymentID, request.VersionID)
	err := a.api.Do(ctx, http.MethodGet, path, nil, nil, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) Heartbeat(ctx context.Context, request HeartbeatRequest) (*HeartbeatResponse, error) {
	var resp HeartbeatResponse
	path := fmt.Sprintf("%s/deployments/%s/versions/%s/heartbeat", basePath, request.DeploymentID, request.VersionID)
	err := a.api.Do(ctx, http.MethodPost, path, nil, nil, struct{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) CompleteVersion(ctx context.Context, request CompleteVersionRequest) (*Version, error) {
	var resp Version
	path := fmt.Sprintf("%s/deployments/%s/versions/%s/complete", basePath, request.DeploymentID, request.VersionID)
	err := a.api.Do(ctx, http.MethodPost, path, nil, nil, request, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) CreateOperation(ctx context.Context, request CreateOperationRequest) (*Operation, error) {
	var resp Operation
	path := fmt.Sprintf("%s/deployments/%s/versions/%s/operations", basePath, request.DeploymentID, request.VersionID)
	query := map[string]any{"resource_key": request.ResourceKey}
	err := a.api.Do(ctx, http.MethodPost, path, nil, query, request.Operation, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *DeploymentMetadataAPI) ListResources(ctx context.Context, request ListResourcesRequest) ([]Resource, error) {
	var allResources []Resource
	pageToken := ""

	for {
		var resp ListResourcesResponse
		path := fmt.Sprintf("%s/deployments/%s/resources", basePath, request.DeploymentID)

		q := map[string]any{
			"page_size": 1000,
		}
		if pageToken != "" {
			q["page_token"] = pageToken
		}

		err := a.api.Do(ctx, http.MethodGet, path, nil, q, nil, &resp)
		if err != nil {
			return nil, err
		}

		allResources = append(allResources, resp.Resources...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return allResources, nil
}
