package direct

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// opRecorder records a resource operation with the deployment metadata service
// (DMS) after it has been applied to the workspace. state is the serialized
// local config after the operation and must be nil for delete operations.
type opRecorder interface {
	record(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any) error
	// recordInitialRegister records the one-time registration of an existing
	// resource that was already managed by DABs but not yet tracked by DMS.
	recordInitialRegister(ctx context.Context, resourceKey, resourceID string, state any) error
}

// operationRecorder records operations via the DMS CreateOperation API.
type operationRecorder struct {
	client sdkbundle.BundleInterface
	// parent is the version the operations are recorded under, formatted as
	// "deployments/{deployment_id}/versions/{version_id}".
	parent string
}

// NewOperationRecorder returns an opRecorder backed by the DMS CreateOperation
// API. deploymentID and version identify the deployment version assigned by DMS
// that the operations are recorded under.
func NewOperationRecorder(client sdkbundle.BundleInterface, deploymentID string, version int64) opRecorder {
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
	return r.recordAction(ctx, resourceKey, actionType, resourceID, state)
}

// recordInitialRegister records an INITIAL_REGISTER operation. The migrate and
// bind workflows adopt an existing resource into the direct engine without a
// deploy-time create/update, so the first operation DMS sees for that resource
// is its registration rather than a mutation.
func (r *operationRecorder) recordInitialRegister(ctx context.Context, resourceKey, resourceID string, state any) error {
	return r.recordAction(ctx, resourceKey, sdkbundle.OperationActionTypeOperationActionTypeInitialRegister, resourceID, state)
}

func (r *operationRecorder) recordAction(ctx context.Context, resourceKey string, actionType sdkbundle.OperationActionType, resourceID string, state any) error {
	op := sdkbundle.Operation{
		ActionType:  actionType,
		ResourceId:  resourceID,
		ResourceKey: resourceKey,
		Status:      sdkbundle.OperationStatusOperationStatusSucceeded,
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

	_, err := r.client.CreateOperation(ctx, sdkbundle.CreateOperationRequest{
		Parent:      r.parent,
		ResourceKey: resourceKey,
		Operation:   op,
	})
	return err
}

// deployActionToSDK maps a deployplan action to its DMS operation action type.
// Only actions that mutate a resource are recordable; Skip and Undefined never
// reach a recorder and are rejected rather than silently coerced.
func deployActionToSDK(a deployplan.ActionType) (sdkbundle.OperationActionType, error) {
	switch a {
	case deployplan.Create:
		return sdkbundle.OperationActionTypeOperationActionTypeCreate, nil
	case deployplan.Update:
		return sdkbundle.OperationActionTypeOperationActionTypeUpdate, nil
	case deployplan.UpdateWithID:
		return sdkbundle.OperationActionTypeOperationActionTypeUpdateWithId, nil
	case deployplan.Recreate:
		return sdkbundle.OperationActionTypeOperationActionTypeRecreate, nil
	case deployplan.Resize:
		return sdkbundle.OperationActionTypeOperationActionTypeResize, nil
	case deployplan.Delete:
		return sdkbundle.OperationActionTypeOperationActionTypeDelete, nil
	default:
		return "", fmt.Errorf("cannot record operation: unsupported action %q", a)
	}
}
