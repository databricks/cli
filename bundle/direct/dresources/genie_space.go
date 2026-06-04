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

// ResourceGenieSpace mirrors the dashboard resource pattern (see dashboard.go),
// with these intentional divergences:
//   - No Published wrapper: Genie spaces have no publish lifecycle, so
//     PrepareState returns the config directly.
//   - RemapState filters fewer fields: Genie has no LifecycleState / CreateTime /
//     Path / UpdateTime output-only fields to scrub.
//   - DoUpdate omits serialized_space when unchanged: serialized_space is in
//     ignore_remote_changes (see resources.yml), so a UI edit produces no plan
//     entry. Sending the local body anyway would clobber the UI edit on every
//     unrelated update.
//   - DoCreate has expanded missing-parent-path detection: see
//     isMissingGenieParentPathError below.
//
// Permissions follow the standard /permissions/genie/{id} endpoint and are wired
// up via the generic permissions adapter (permissions.go).
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
		Etag:            state.Etag,
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
	forceSendFields := utils.FilterFields[resources.GenieSpaceConfig](space.ForceSendFields)

	return &resources.GenieSpaceConfig{
		Description:     space.Description,
		Etag:            space.Etag,
		Title:           space.Title,
		WarehouseId:     space.WarehouseId,
		ParentPath:      ensureWorkspacePrefix(space.ParentPath),
		SerializedSpace: serializedSpace,

		// Output only fields
		SpaceId:         space.SpaceId,
		ForceSendFields: forceSendFields,
	}
}

// isMissingGenieParentPathError reports whether the given Create error means
// "the parent workspace folder does not exist", so DoCreate can mkdir and retry.
//
// Dashboard handles the equivalent condition with a plain apierr.IsMissing
// check (see ResourceDashboard.DoCreate). Genie cannot, because it surfaces
// the same condition in two different shapes depending on the workspace's
// backend version:
//
//  1. Standard missing-resource error: HTTP 404, ErrorCode RESOURCE_DOES_NOT_EXIST.
//     Caught by apierr.IsMissing. Observed on workspaces running the newer
//     Genie service implementation.
//  2. HTTP 400 with ErrorCode INVALID_PARAMETER_VALUE and a message of the
//     form "Tree node with path '<path>' does not exist". Observed on
//     workspaces still backed by the legacy implementation during integration
//     testing in early 2026 (aws-prod-ucws and azure-prod-ucws clusters at
//     the time). The string match is intentional: there is no distinct error
//     code to key on.
//
// Both forms unambiguously mean "create the parent and retry once".
func isMissingGenieParentPathError(err error) bool {
	if apierr.IsMissing(err) {
		return true
	}

	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return false
	}

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

	// Persist the etag in state. The deploy framework saves `config` (the input
	// to DoCreate) as the state record, so mutating it here is what gets the
	// backend-returned etag onto disk for the next plan's drift check.
	// Matches the dashboard pattern (dashboard.go DoCreate).
	config.Etag = createResp.Etag

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
	sentSerialized := true
	if !hasUpdate(entry, pathSerializedSpace) {
		serializedSpace = ""
		sentSerialized = false
		excludeForceSend = append(excludeForceSend, "SerializedSpace")
	}

	updateResp, err := r.client.Genie.UpdateSpace(ctx, dashboards.GenieUpdateSpaceRequest{
		SpaceId:         id,
		Description:     config.Description,
		Title:           config.Title,
		WarehouseId:     config.WarehouseId,
		ParentPath:      config.ParentPath,
		SerializedSpace: serializedSpace,
		// Send the etag we last observed. The backend uses it as an If-Match
		// guard against concurrent writes, and OverrideChangeDesc uses the
		// post-update etag to detect drift on subsequent plans.
		Etag: config.Etag,

		ForceSendFields: utils.FilterFields[dashboards.GenieUpdateSpaceRequest](config.ForceSendFields, excludeForceSend...),
	})
	if err != nil {
		return nil, err
	}

	// Persist the new etag in state (see DoCreate for the rationale).
	config.Etag = updateResp.Etag

	// Decide what to record as the new state's serialized_space.
	//   - If we sent a new body, use it.
	//   - If we omitted it (UI-edit protection above) but the API echoed back
	//     a value, record that — it's the most up-to-date view we have.
	//   - If neither side carries a value, keep whatever was already in state.
	//     Otherwise RemapState would blank the field on every unrelated update.
	respSerialized := serializedSpace
	if !sentSerialized {
		respSerialized = updateResp.SerializedSpace
		if respSerialized == "" {
			if prior, ok := config.SerializedSpace.(string); ok {
				respSerialized = prior
			}
		}
	}

	return responseToGenieSpaceConfig(updateResp, respSerialized), nil
}

// OverrideChangeDesc handles the etag field. The user never sets it directly;
// we compare the stored etag against the remote one and Skip if they match.
// This mirrors ResourceDashboard.OverrideChangeDesc.
func (r *ResourceGenieSpace) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, _ *resources.GenieSpaceConfig) error {
	switch path.String() {
	case "etag":
		// change.New is always nil for etag because it's not present in the
		// user-authored config. Compare stored etag with remote one to decide
		// whether anything changed out-of-band since the last deploy.
		if change.Old == change.Remote {
			change.Action = deployplan.Skip
		} else {
			change.Action = deployplan.Update
		}
	}
	return nil
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

func (r *ResourceGenieSpace) DoDelete(ctx context.Context, id string, _ *resources.GenieSpaceConfig) error {
	return r.client.Genie.TrashSpace(ctx, dashboards.GenieTrashSpaceRequest{
		SpaceId: id,
	})
}
