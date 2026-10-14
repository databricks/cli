// Code ported from github.com/databricks/databricks-sdk-go.

package workspaceapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/listing"
)

type workspaceImpl struct {
	client *apiclient.Client
}

func (a *workspaceImpl) Delete(ctx context.Context, request Delete) error {
	path := "/api/2.0/workspace/delete"
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPost, path, headers, queryParams, request, nil)
	return err
}

func (a *workspaceImpl) Export(ctx context.Context, request ExportRequest) (*ExportResponse, error) {
	var exportResponse ExportResponse
	path := "/api/2.0/workspace/export"
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &exportResponse)
	return &exportResponse, err
}

func (a *workspaceImpl) GetPermissionLevels(ctx context.Context, request GetWorkspaceObjectPermissionLevelsRequest) (*GetWorkspaceObjectPermissionLevelsResponse, error) {
	var getWorkspaceObjectPermissionLevelsResponse GetWorkspaceObjectPermissionLevelsResponse
	path := fmt.Sprintf("/api/2.0/permissions/%v/%v/permissionLevels", request.WorkspaceObjectType, request.WorkspaceObjectId)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &getWorkspaceObjectPermissionLevelsResponse)
	return &getWorkspaceObjectPermissionLevelsResponse, err
}

func (a *workspaceImpl) GetPermissions(ctx context.Context, request GetWorkspaceObjectPermissionsRequest) (*WorkspaceObjectPermissions, error) {
	var workspaceObjectPermissions WorkspaceObjectPermissions
	path := fmt.Sprintf("/api/2.0/permissions/%v/%v", request.WorkspaceObjectType, request.WorkspaceObjectId)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &workspaceObjectPermissions)
	return &workspaceObjectPermissions, err
}

func (a *workspaceImpl) GetStatus(ctx context.Context, request GetStatusRequest) (*ObjectInfo, error) {
	var objectInfo ObjectInfo
	path := "/api/2.0/workspace/get-status"
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &objectInfo)
	return &objectInfo, err
}

func (a *workspaceImpl) Import(ctx context.Context, request Import) error {
	path := "/api/2.0/workspace/import"
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPost, path, headers, queryParams, request, nil)
	return err
}

// Lists the contents of a directory, or the object if it is not a directory. If
// the input path does not exist, this call returns an error
// `RESOURCE_DOES_NOT_EXIST`.
func (a *workspaceImpl) List(ctx context.Context, request ListWorkspaceRequest) listing.Iterator[ObjectInfo] {

	getNextPage := func(ctx context.Context, req ListWorkspaceRequest) (*ListResponse, error) {
		return a.internalList(ctx, req)
	}
	getItems := func(resp *ListResponse) []ObjectInfo {
		return resp.Objects
	}

	iterator := listing.NewIterator(
		&request,
		getNextPage,
		getItems,
		nil)
	return iterator
}

// Lists the contents of a directory, or the object if it is not a directory. If
// the input path does not exist, this call returns an error
// `RESOURCE_DOES_NOT_EXIST`.
func (a *workspaceImpl) ListAll(ctx context.Context, request ListWorkspaceRequest) ([]ObjectInfo, error) {
	iterator := a.List(ctx, request)
	return listing.ToSlice[ObjectInfo](ctx, iterator)
}

func (a *workspaceImpl) internalList(ctx context.Context, request ListWorkspaceRequest) (*ListResponse, error) {
	var listResponse ListResponse
	path := "/api/2.0/workspace/list"
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &listResponse)
	return &listResponse, err
}

func (a *workspaceImpl) Mkdirs(ctx context.Context, request Mkdirs) error {
	path := "/api/2.0/workspace/mkdirs"
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPost, path, headers, queryParams, request, nil)
	return err
}

func (a *workspaceImpl) SetPermissions(ctx context.Context, request WorkspaceObjectPermissionsRequest) (*WorkspaceObjectPermissions, error) {
	var workspaceObjectPermissions WorkspaceObjectPermissions
	path := fmt.Sprintf("/api/2.0/permissions/%v/%v", request.WorkspaceObjectType, request.WorkspaceObjectId)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPut, path, headers, queryParams, request, &workspaceObjectPermissions)
	return &workspaceObjectPermissions, err
}

func (a *workspaceImpl) UpdatePermissions(ctx context.Context, request WorkspaceObjectPermissionsRequest) (*WorkspaceObjectPermissions, error) {
	var workspaceObjectPermissions WorkspaceObjectPermissions
	path := fmt.Sprintf("/api/2.0/permissions/%v/%v", request.WorkspaceObjectType, request.WorkspaceObjectId)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPatch, path, headers, queryParams, request, &workspaceObjectPermissions)
	return &workspaceObjectPermissions, err
}
