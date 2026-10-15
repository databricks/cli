package dresources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/apierrs"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkgenie "github.com/databricks/sdk-go/genie/v1"
)

// ResourceGenieSpace mirrors the dashboard resource pattern (see dashboard.go),
// with these intentional divergences:
//   - No Published wrapper: Genie spaces have no publish lifecycle, so
//     PrepareState returns the config directly.
//   - responseToGenieSpaceConfig maps fewer fields: Genie has no LifecycleState /
//     CreateTime / Path / UpdateTime output-only fields to scrub.
//   - DoUpdate omits the etag (dashboard sends it as an If-Match guard): the
//     backend bumps the etag when it migrates serialized_space to a newer
//     schema version, so sending a stale etag would 409 the update after a
//     migration. Drift is still detected on read via OverrideChangeDesc.
//   - DoCreate has expanded missing-parent-path detection: see
//     isMissingGenieParentPathError below.
//
// Permissions follow the standard /permissions/genie/{id} endpoint and are wired
// up via the generic permissions adapter (permissions.go).
type ResourceGenieSpace struct {
	client *sdkclients.Clients
}

func (*ResourceGenieSpace) New(client *sdkclients.Clients) *ResourceGenieSpace {
	return &ResourceGenieSpace{client: client}
}

func (*ResourceGenieSpace) PrepareState(input *resources.GenieSpace) *resources.GenieSpaceConfig {
	return &input.GenieSpaceConfig
}

func (r *ResourceGenieSpace) DoRead(ctx context.Context, id string) (*resources.GenieSpaceConfig, error) {
	space, err := r.client.Genie.GenieGetSpace(ctx, sdkgenie.GenieGetSpaceRequest{
		SpaceId:                new(id),
		IncludeSerializedSpace: new(true), // otherwise etag isn't returned
	})
	if err != nil {
		return nil, err
	}
	return responseToGenieSpaceConfig(space, utils.Deref(space.SerializedSpace)), nil
}

// prepareGenieSpaceRequest returns the serialized_space body to send to the API.
// ConfigureGenieSpaceSerializedSpace normalizes serialized_space to a JSON string
// (read from file_path, or marshalled from inline YAML) before the deploy engine
// runs, so the value is always a string or unset by this point.
func prepareGenieSpaceRequest(config *resources.GenieSpaceConfig) (string, error) {
	switch v := config.SerializedSpace.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("internal error: serialized_space should have been normalized to a string, got %T", v)
	}
}

func responseToGenieSpaceConfig(space *sdkgenie.GenieSpace, serializedSpace string) *resources.GenieSpaceConfig {
	return &resources.GenieSpaceConfig{
		Description:     utils.Deref(space.Description),
		Etag:            utils.Deref(space.Etag),
		Title:           utils.Deref(space.Title),
		WarehouseId:     utils.Deref(space.WarehouseId),
		ParentPath:      ensureWorkspacePrefix(utils.Deref(space.ParentPath)),
		SerializedSpace: serializedSpace,
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
	if apierrs.IsMissing(err) {
		return true
	}

	if _, ok := apierrs.Get(err); !ok {
		return false
	}

	message := apierrs.Message(err)
	return apierrs.StatusCode(err) == http.StatusBadRequest &&
		apierrs.ErrorCode(err) == "INVALID_PARAMETER_VALUE" &&
		strings.Contains(message, "Tree node with path") &&
		strings.Contains(message, "does not exist")
}

func (r *ResourceGenieSpace) DoCreate(ctx context.Context, config *resources.GenieSpaceConfig) (string, *resources.GenieSpaceConfig, error) {
	serializedSpace, err := prepareGenieSpaceRequest(config)
	if err != nil {
		return "", nil, err
	}

	req := sdkgenie.GenieCreateSpaceRequest{
		Description:     nonEmpty(config.Description),
		Title:           nonEmpty(config.Title),
		WarehouseId:     nonEmpty(config.WarehouseId),
		ParentPath:      nonEmpty(config.ParentPath),
		SerializedSpace: nonEmpty(serializedSpace),
	}

	createResp, err := r.client.Genie.CreateSpace(ctx, req)

	// Retry once after creating the parent directory when the workspace folder
	// is missing. Genie can surface this either as a standard missing-resource
	// error or as INVALID_PARAMETER_VALUE with a "Tree node ... does not exist"
	// message depending on the backend.
	if err != nil && isMissingGenieParentPathError(err) {
		err = r.client.Workspace.MkdirsByPath(ctx, config.ParentPath)
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
	config.Etag = utils.Deref(createResp.Etag)

	return utils.Deref(createResp.SpaceId), responseToGenieSpaceConfig(createResp, serializedSpace), nil
}

func (r *ResourceGenieSpace) DoUpdate(ctx context.Context, id string, config *resources.GenieSpaceConfig, _ *PlanEntry) (*resources.GenieSpaceConfig, error) {
	serializedSpace, err := prepareGenieSpaceRequest(config)
	if err != nil {
		return nil, err
	}

	// Always send serialized_space so the deploy converges the space to the
	// bundle config (the bundle is the source of truth, mirroring dashboards).
	// We cannot use the etag as an If-Match guard: the backend bumps it when it
	// migrates serialized_space to a newer schema version, so a stale etag would
	// 409 a legitimate update after a migration. Etag is therefore left empty;
	// out-of-band drift is surfaced on read via OverrideChangeDesc instead.
	//
	// description is always sent so that clearing it in config clears it on the space.
	// Verified against a real workspace: UpdateSpace accepts {"description": ""} as a clear.
	updateResp, err := r.client.Genie.UpdateSpace(ctx, sdkgenie.GenieUpdateSpaceRequest{
		SpaceId:         new(id),
		Description:     new(config.Description),
		Title:           nonEmpty(config.Title),
		WarehouseId:     nonEmpty(config.WarehouseId),
		ParentPath:      nonEmpty(config.ParentPath),
		SerializedSpace: nonEmpty(serializedSpace),
		Etag:            nil, // Not set by DABs
	})
	if err != nil {
		return nil, err
	}

	// Persist the new etag in state (see DoCreate for the rationale). Record the
	// body we sent, mirroring DoCreate, so config-side and state-side match.
	config.Etag = utils.Deref(updateResp.Etag)

	return responseToGenieSpaceConfig(updateResp, serializedSpace), nil
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

func (r *ResourceGenieSpace) DoDelete(ctx context.Context, id string, _ *resources.GenieSpaceConfig) error {
	return r.client.Genie.GenieTrashSpace(ctx, sdkgenie.GenieTrashSpaceRequest{
		SpaceId: new(id),
	})
}
