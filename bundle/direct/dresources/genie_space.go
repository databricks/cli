package dresources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

var pathSerializedSpace = structpath.MustParsePath("serialized_space")

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
	return responseToGenieSpaceConfig(space, space.SerializedSpace), nil
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
	// Drop ParentPath from ForceSendFields. We always clear ParentPath
	// below because the GET Genie space API does not reliably return it,
	// and keeping it in ForceSendFields would force-emit parent_path: ""
	// in state output even though the field is logically unset.
	forceSendFields := utils.FilterFields[resources.GenieSpaceConfig](space.ForceSendFields, "ParentPath")

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

func isMissingGenieParentPathError(err error) bool {
	if apierr.IsMissing(err) {
		return true
	}

	var apiErr *apierr.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	// Genie reports a missing parent folder inconsistently across environments.
	// Some workspaces return a standard missing-resource error, while others
	// return INVALID_PARAMETER_VALUE with a NOT_FOUND message embedded in the
	// text. Treat both forms as "create the parent directory and retry once".
	return apiErr.StatusCode == http.StatusBadRequest &&
		apiErr.ErrorCode == "INVALID_PARAMETER_VALUE" &&
		strings.Contains(apiErr.Message, "Tree node with path") &&
		strings.Contains(apiErr.Message, "does not exist")
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

	// Retry once after creating the parent directory when the workspace folder
	// is missing. Genie can surface this either as a standard missing-resource
	// error or as INVALID_PARAMETER_VALUE with a "Tree node ... does not exist"
	// message depending on the backend.
	if err != nil && isMissingGenieParentPathError(err) {
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

func (r *ResourceGenieSpace) DoUpdate(ctx context.Context, id string, config *resources.GenieSpaceConfig, entry *PlanEntry) (*resources.GenieSpaceConfig, error) {
	serializedSpace, err := prepareGenieSpaceRequest(config)
	if err != nil {
		return nil, err
	}

	// serialized_space is in ignore_remote_changes (we cannot diff structured
	// local YAML against remote JSON), so a UI edit produces no plan entry.
	// If we still sent the unchanged local body on every update, the next
	// update triggered by another field would clobber the UI edit. Only
	// send it when the user actually changed it locally.
	var excludeForceSend []string
	if !hasUpdate(entry, pathSerializedSpace) {
		serializedSpace = ""
		excludeForceSend = append(excludeForceSend, "SerializedSpace")
	}

	updateResp, err := r.client.Genie.UpdateSpace(ctx, dashboards.GenieUpdateSpaceRequest{
		SpaceId:         id,
		Description:     config.Description,
		Title:           config.Title,
		WarehouseId:     config.WarehouseId,
		SerializedSpace: serializedSpace,
		// Etag is for optimistic concurrency; we apply updates unconditionally.
		Etag: "",

		ForceSendFields: utils.FilterFields[dashboards.GenieUpdateSpaceRequest](config.ForceSendFields, excludeForceSend...),
	})
	if err != nil {
		return nil, err
	}

	// When the request omitted serialized_space, use the value the response
	// echoes back so RemapState records the latest body.
	respSerialized := serializedSpace
	if respSerialized == "" {
		respSerialized = updateResp.SerializedSpace
	}

	return responseToGenieSpaceConfig(updateResp, respSerialized), nil
}

// hasUpdate reports whether entry has an Update-action change at the given path.
// HasChange alone matches Skip-action changes too, which we cannot use to drive
// request shaping for fields covered by ignore_remote_changes.
func hasUpdate(entry *PlanEntry, path *structpath.PathNode) bool {
	if entry == nil {
		return false
	}
	for s, change := range entry.Changes {
		if change.Action != deployplan.Update {
			continue
		}
		node, err := structpath.ParsePath(s)
		if err != nil {
			continue
		}
		if node.HasPrefix(path) {
			return true
		}
	}
	return false
}

func (r *ResourceGenieSpace) DoDelete(ctx context.Context, id string) error {
	return r.client.Genie.TrashSpace(ctx, dashboards.GenieTrashSpaceRequest{
		SpaceId: id,
	})
}
