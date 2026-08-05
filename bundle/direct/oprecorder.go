package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// opRecorder records a resource operation with the deployment metadata service
// (DMS) after it has been applied to the workspace. state is the serialized
// local config after the operation and must be nil for delete operations.
type opRecorder interface {
	record(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any) error
}

// recordOperation reports an applied resource operation to DMS. It is a no-op
// unless the bundle opted into recording deployment history (OpRec is set).
// state is the serialized local config after the operation and must be nil for
// delete operations.
func (b *DeploymentBundle) recordOperation(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any) error {
	if b.OpRec == nil {
		return nil
	}
	return b.OpRec.record(ctx, resourceKey, action, resourceID, state)
}

// operationRecorder records operations via the DMS CreateOperation API.
type operationRecorder struct {
	client bundledeployments.BundleDeploymentsInterface
	// parent is the version the operations are recorded under, formatted as
	// "deployments/{deployment_id}/versions/{version_id}".
	parent string
}

// NewOperationRecorder returns an opRecorder backed by the DMS CreateOperation
// API. deploymentID and version identify the deployment version assigned by DMS
// that the operations are recorded under.
func NewOperationRecorder(client bundledeployments.BundleDeploymentsInterface, deploymentID string, version int64) opRecorder {
	return &operationRecorder{
		client: client,
		parent: fmt.Sprintf("deployments/%s/versions/%d", deploymentID, version),
	}
}

func (r *operationRecorder) record(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any) error {
	actionType, err := deployActionToSDK(action)
	if err != nil {
		return err
	}

	// DMS resource keys are unprefixed (e.g. "jobs.foo"), while the CLI's state
	// keys carry a leading "resources." (e.g. "resources.jobs.foo"). Strip it on
	// the way out; the read path re-adds it (see dstate.fetchDeploymentResources).
	dmsKey := strings.TrimPrefix(resourceKey, "resources.")

	op := bundledeployments.Operation{
		ActionType:  actionType,
		ResourceId:  resourceID,
		ResourceKey: dmsKey,
		Status:      bundledeployments.OperationStatusOperationStatusSucceeded,
	}

	// The DMS Operation.State field carries the serialized config so the backend
	// can serve it as resource state. It is intentionally left unset for delete,
	// where the resource no longer exists.
	if state != nil {
		raw, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("serializing state: %w", err)
		}
		msg := json.RawMessage(raw)
		op.State = &msg
	}

	_, err = r.client.CreateOperation(ctx, bundledeployments.CreateOperationRequest{
		Parent:      r.parent,
		ResourceKey: dmsKey,
		Operation:   op,
	})
	return err
}

// deployActionToSDK maps a deployplan action to its DMS operation action type.
// Only actions that mutate a resource are recordable; Skip and Undefined never
// reach a recorder and are rejected rather than silently coerced.
func deployActionToSDK(a deployplan.ActionType) (bundledeployments.OperationActionType, error) {
	switch a {
	case deployplan.Create:
		return bundledeployments.OperationActionTypeOperationActionTypeCreate, nil
	case deployplan.Update:
		return bundledeployments.OperationActionTypeOperationActionTypeUpdate, nil
	case deployplan.UpdateWithID:
		return bundledeployments.OperationActionTypeOperationActionTypeUpdateWithId, nil
	case deployplan.Recreate:
		return bundledeployments.OperationActionTypeOperationActionTypeRecreate, nil
	case deployplan.Resize:
		return bundledeployments.OperationActionTypeOperationActionTypeResize, nil
	case deployplan.Delete:
		return bundledeployments.OperationActionTypeOperationActionTypeDelete, nil
	default:
		return "", fmt.Errorf("cannot record operation: unsupported action %q", a)
	}
}
