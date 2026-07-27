package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// recordedOperation is an applied resource operation, serialized and waiting to be
// uploaded to the deployment metadata service (DMS).
//
// The payload is built on the apply worker rather than in the uploader so the
// queue does not hold on to the live resource struct, and so a malformed state
// fails the resource that produced it instead of the drain at the end of apply.
type recordedOperation struct {
	action     bundledeployments.OperationActionType
	resourceID string

	// state is the serialized local config after the operation. It is nil for a
	// delete, where the resource no longer exists.
	state json.RawMessage
}

// newRecordedOperation serializes an applied operation for upload. state is the
// local config after the operation and must be nil for delete operations.
func newRecordedOperation(action deployplan.ActionType, resourceID string, state any) (recordedOperation, error) {
	actionType, err := deployActionToSDK(action)
	if err != nil {
		return recordedOperation{}, err
	}

	op := recordedOperation{action: actionType, resourceID: resourceID}

	// The DMS Operation.State field carries the serialized config so the backend
	// can serve it as resource state. It is intentionally left unset for delete,
	// where the resource no longer exists.
	//
	// Redact sensitive fields, matching what dstate.SaveState writes to the local
	// state file: DMS state is read back as resource state, so recording secrets
	// in plaintext would both leak them to the service and reintroduce them into
	// a local state file via the read path.
	if state != nil {
		raw, err := structwalk.RedactSensitiveFields(state, dyn.SensitiveValueRedacted)
		if err != nil {
			return recordedOperation{}, fmt.Errorf("serializing state: %w", err)
		}
		op.state = raw
	}

	return op, nil
}

// mergeAction returns the action to record when a later operation coalesces into
// one still queued for the same resource (see operationQueue.record). The state
// uploaded is the later one, but the action must not be downgraded: Create and
// Recreate tell DMS the resource ID is new, and a subsequent Update only refines
// the state of that same new resource. Recording the pair as an Update would
// claim the resource already existed. A Delete is the exception - the resource is
// gone, so nothing earlier is worth reporting.
func mergeAction(queued, next bundledeployments.OperationActionType) bundledeployments.OperationActionType {
	if next == bundledeployments.OperationActionTypeOperationActionTypeDelete {
		return next
	}
	if queued == bundledeployments.OperationActionTypeOperationActionTypeCreate ||
		queued == bundledeployments.OperationActionTypeOperationActionTypeRecreate {
		return queued
	}
	return next
}

// operationUploader records an applied resource operation with DMS. Uploads run
// on the operationQueue workers, off the apply path.
type operationUploader interface {
	upload(ctx context.Context, resourceKey string, op recordedOperation) error
}

// operationRecorder uploads operations via the DMS CreateOperation API.
type operationRecorder struct {
	client bundledeployments.BundleDeploymentsInterface
	// parent is the version the operations are recorded under, formatted as
	// "deployments/{deployment_id}/versions/{version_id}".
	parent string
}

// NewOperationRecorder returns an operationUploader backed by the DMS
// CreateOperation API. deploymentID and version identify the deployment version
// assigned by DMS that the operations are recorded under.
func NewOperationRecorder(client bundledeployments.BundleDeploymentsInterface, deploymentID string, version int64) operationUploader {
	return &operationRecorder{
		client: client,
		parent: fmt.Sprintf("deployments/%s/versions/%d", deploymentID, version),
	}
}

func (r *operationRecorder) upload(ctx context.Context, resourceKey string, op recordedOperation) error {
	// DMS resource keys are unprefixed (e.g. "jobs.foo"), while the CLI's state
	// keys carry a leading "resources." (e.g. "resources.jobs.foo"). Strip it on
	// the way out; the read path re-adds it (see dstate.fetchDeploymentResources).
	dmsKey := strings.TrimPrefix(resourceKey, "resources.")

	operation := bundledeployments.Operation{
		ActionType:  op.action,
		ResourceId:  op.resourceID,
		ResourceKey: dmsKey,
		Status:      bundledeployments.OperationStatusOperationStatusSucceeded,
	}
	if op.state != nil {
		operation.State = &op.state
	}

	_, err := r.client.CreateOperation(ctx, bundledeployments.CreateOperationRequest{
		Parent:      r.parent,
		ResourceKey: dmsKey,
		Operation:   operation,
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
