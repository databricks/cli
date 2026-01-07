package dresources

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

// Genie API reference: https://docs.databricks.com/api/workspace/genie
type ResourceGenieSpace struct {
	client *databricks.WorkspaceClient
}

// ensureWorkspacePrefix adds the /Workspace prefix to the parent path if it's not already present.
// The backend removes this prefix from parent path, and thus it needs to be added back
// to match the local configuration.
func ensureGenieWorkspacePrefix(parentPath string) string {
	if parentPath == "" {
		return parentPath
	}
	if parentPath == "/Workspace" || strings.HasPrefix(parentPath, "/Workspace/") {
		return parentPath
	}
	return path.Join("/Workspace", parentPath)
}

func (*ResourceGenieSpace) New(client *databricks.WorkspaceClient) *ResourceGenieSpace {
	return &ResourceGenieSpace{client: client}
}

func (*ResourceGenieSpace) PrepareState(input *resources.GenieSpace) *resources.GenieSpaceConfig {
	return &input.GenieSpaceConfig
}

func (r *ResourceGenieSpace) RemapState(state *resources.GenieSpaceConfig) *resources.GenieSpaceConfig {
	return &resources.GenieSpaceConfig{
		Title:           state.Title,
		Description:     state.Description,
		ParentPath:      state.ParentPath,
		WarehouseId:     state.WarehouseId,
		SerializedSpace: state.SerializedSpace,

		// Clear output-only fields. They should not show up on remote diff computation.
		SpaceId:         "",
		ForceSendFields: nil,
	}
}

func (r *ResourceGenieSpace) DoRead(ctx context.Context, id string) (*resources.GenieSpaceConfig, error) {
	space, err := r.client.Genie.GetSpace(ctx, dashboards.GenieGetSpaceRequest{
		SpaceId:                id,
		IncludeSerializedSpace: true,
		ForceSendFields:        nil,
	})
	if err != nil {
		return nil, err
	}

	return &resources.GenieSpaceConfig{
		SpaceId:         space.SpaceId,
		Title:           space.Title,
		Description:     space.Description,
		ParentPath:      "",
		WarehouseId:     space.WarehouseId,
		SerializedSpace: space.SerializedSpace,
		ForceSendFields: nil,
		// Note: ParentPath is not returned by GetSpace API, so we can't set it here.
		// This means parent_path changes won't be detected via remote drift.
		// However, FieldTriggers ensures parent_path changes trigger recreate locally.
	}, nil
}

// prepareGenieSpaceRequest returns the serialized_space string from the config.
func prepareGenieSpaceRequest(config *resources.GenieSpaceConfig) string {
	return config.SerializedSpace
}

func responseToGenieState(resp *dashboards.GenieSpace, serializedSpace, parentPath string) *resources.GenieSpaceConfig {
	return &resources.GenieSpaceConfig{
		SpaceId:         resp.SpaceId,
		Title:           resp.Title,
		Description:     resp.Description,
		ParentPath:      ensureGenieWorkspacePrefix(parentPath),
		WarehouseId:     resp.WarehouseId,
		SerializedSpace: serializedSpace,
		ForceSendFields: nil,
	}
}

func (r *ResourceGenieSpace) DoCreate(ctx context.Context, config *resources.GenieSpaceConfig) (string, *resources.GenieSpaceConfig, error) {
	serializedSpace := prepareGenieSpaceRequest(config)

	createReq := dashboards.GenieCreateSpaceRequest{
		WarehouseId:     config.WarehouseId,
		SerializedSpace: serializedSpace,
		Title:           config.Title,
		Description:     config.Description,
		ParentPath:      config.ParentPath,
		ForceSendFields: nil,
	}

	resp, err := r.client.Genie.CreateSpace(ctx, createReq)

	// The API returns 404 if the parent directory doesn't exist.
	// If the parent directory doesn't exist, create it and try again.
	if err != nil && apierr.IsMissing(err) && config.ParentPath != "" {
		mkdirErr := r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
		if mkdirErr != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", mkdirErr)
		}
		resp, err = r.client.Genie.CreateSpace(ctx, createReq)
	}
	if err != nil {
		return "", nil, err
	}

	return resp.SpaceId, responseToGenieState(resp, serializedSpace, config.ParentPath), nil
}

func (r *ResourceGenieSpace) DoUpdate(ctx context.Context, id string, config *resources.GenieSpaceConfig, _ *Changes) (*resources.GenieSpaceConfig, error) {
	serializedSpace := prepareGenieSpaceRequest(config)

	resp, err := r.client.Genie.UpdateSpace(ctx, dashboards.GenieUpdateSpaceRequest{
		SpaceId:         id,
		SerializedSpace: serializedSpace,
		Title:           config.Title,
		Description:     config.Description,
		WarehouseId:     config.WarehouseId,
		ForceSendFields: nil,
	})
	if err != nil {
		return nil, err
	}

	return responseToGenieState(resp, serializedSpace, config.ParentPath), nil
}

func (r *ResourceGenieSpace) DoDelete(ctx context.Context, id string) error {
	return r.client.Genie.TrashSpace(ctx, dashboards.GenieTrashSpaceRequest{
		SpaceId: id,
	})
}

func (*ResourceGenieSpace) FieldTriggers(isLocal bool) map[string]deployplan.ActionType {
	triggers := map[string]deployplan.ActionType{
		// Change in parent_path should trigger a recreate since Genie API
		// doesn't support updating parent_path.
		"parent_path": deployplan.ActionTypeRecreate,
	}

	if !isLocal {
		// For remote diff, skip serialized_space comparison since the format
		// may differ between local config and API response.
		triggers["serialized_space"] = deployplan.ActionTypeSkip
	}

	return triggers
}
