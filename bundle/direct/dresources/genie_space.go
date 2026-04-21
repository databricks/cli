package dresources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

type ResourceGenieSpace struct {
	client *databricks.WorkspaceClient
}

func (*ResourceGenieSpace) New(client *databricks.WorkspaceClient) *ResourceGenieSpace {
	return &ResourceGenieSpace{client: client}
}

func (*ResourceGenieSpace) PrepareState(input *resources.GenieSpace) *resources.GenieSpaceConfig {
	return &input.GenieSpaceConfig
}

func (r *ResourceGenieSpace) RemapState(state *resources.GenieSpaceConfig) *resources.GenieSpaceConfig {
	forceSendFields := utils.FilterFields[resources.GenieSpaceConfig](state.ForceSendFields, []string{
		"SpaceId",
		"SerializedSpace",
	}...)

	return &resources.GenieSpaceConfig{
		Description:     state.Description,
		Title:           state.Title,
		WarehouseId:     state.WarehouseId,
		ParentPath:      state.ParentPath,
		SerializedSpace: state.SerializedSpace,

		ForceSendFields: forceSendFields,

		// Clear output only fields. They should not show up on remote diff computation.
		SpaceId: "",
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

	forceSendFields := utils.FilterFields[resources.GenieSpaceConfig](space.ForceSendFields)

	return &resources.GenieSpaceConfig{
		Description:     space.Description,
		Title:           space.Title,
		WarehouseId:     space.WarehouseId,
		ParentPath:      "",
		SerializedSpace: space.SerializedSpace,

		// Output only fields
		SpaceId:         space.SpaceId,
		ForceSendFields: forceSendFields,
	}, nil
}

func prepareGenieSpaceRequest(config *resources.GenieSpaceConfig) (string, error) {
	v := config.SerializedSpace
	if serializedSpace, ok := v.(string); ok {
		return serializedSpace, nil
	} else if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal serialized_space: %w", err)
		}
		return string(b), nil
	}
	return "", nil
}

func responseToGenieSpaceConfig(space *dashboards.GenieSpace, serializedSpace string) *resources.GenieSpaceConfig {
	forceSendFields := utils.FilterFields[resources.GenieSpaceConfig](space.ForceSendFields)

	return &resources.GenieSpaceConfig{
		Description:     space.Description,
		Title:           space.Title,
		WarehouseId:     space.WarehouseId,
		ParentPath:      "",
		SerializedSpace: serializedSpace,

		// Output only fields
		SpaceId:         space.SpaceId,
		ForceSendFields: forceSendFields,
	}
}

func (r *ResourceGenieSpace) DoCreate(ctx context.Context, config *resources.GenieSpaceConfig) (string, *resources.GenieSpaceConfig, error) {
	serializedSpace, err := prepareGenieSpaceRequest(config)
	if err != nil {
		return "", nil, err
	}

	req := dashboards.GenieCreateSpaceRequest{
		Description:     config.Description,
		Title:           config.Title,
		WarehouseId:     config.WarehouseId,
		ParentPath:      config.ParentPath,
		SerializedSpace: serializedSpace,

		ForceSendFields: utils.FilterFields[dashboards.GenieCreateSpaceRequest](config.ForceSendFields),
	}

	createResp, err := r.client.Genie.CreateSpace(ctx, req)

	// The API returns 404 if the parent directory doesn't exist.
	// Create it and retry once.
	if err != nil && apierr.IsMissing(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath) //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
		if err != nil {
			return "", nil, fmt.Errorf("failed to create parent directory: %w", err)
		}
		createResp, err = r.client.Genie.CreateSpace(ctx, req)
	}
	if err != nil {
		return "", nil, err
	}

	return createResp.SpaceId, responseToGenieSpaceConfig(createResp, serializedSpace), nil
}

func (r *ResourceGenieSpace) DoUpdate(ctx context.Context, id string, config *resources.GenieSpaceConfig, _ *PlanEntry) (*resources.GenieSpaceConfig, error) {
	serializedSpace, err := prepareGenieSpaceRequest(config)
	if err != nil {
		return nil, err
	}

	updateResp, err := r.client.Genie.UpdateSpace(ctx, dashboards.GenieUpdateSpaceRequest{
		SpaceId:         id,
		Description:     config.Description,
		Title:           config.Title,
		WarehouseId:     config.WarehouseId,
		SerializedSpace: serializedSpace,
		// Etag is for optimistic concurrency; we apply updates unconditionally.
		Etag: "",

		ForceSendFields: utils.FilterFields[dashboards.GenieUpdateSpaceRequest](config.ForceSendFields),
	})
	if err != nil {
		return nil, err
	}

	return responseToGenieSpaceConfig(updateResp, serializedSpace), nil
}

func (r *ResourceGenieSpace) DoDelete(ctx context.Context, id string) error {
	return r.client.Genie.TrashSpace(ctx, dashboards.GenieTrashSpaceRequest{
		SpaceId: id,
	})
}
