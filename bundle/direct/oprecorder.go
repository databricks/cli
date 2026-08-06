package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// maxOperationStateSize is the largest serialized state DMS accepts per
// operation. Uploading more is rejected server-side, so fail early with a
// message that names the resource.
const maxOperationStateSize = 64 * 1024

// maxOperationErrorMessageSize is the largest error message DMS accepts per
// operation. A longer message is truncated rather than rejected, so a failing
// resource is still recorded with its error instead of the recording itself
// failing and masking the error we are trying to report.
const maxOperationErrorMessageSize = 16 * 1024

// recordedOperation is an applied resource operation, serialized and waiting to be
// uploaded to the deployment metadata service (DMS).
//
// The payload is built on the apply worker rather than in the uploader so the
// queue does not hold on to the live resource struct, and so a malformed state
// fails the resource that produced it instead of the drain at the end of apply.
type recordedOperation struct {
	action     bundledeployments.OperationActionType
	resourceID string
	status     bundledeployments.OperationStatus

	// errorMessage is why the operation failed. It is set only when status is
	// failed, which the service enforces.
	errorMessage string

	// state is the serialized local config after the operation. It is nil for a
	// delete, where the resource no longer exists, and for a failure, where the
	// resource was not written.
	state json.RawMessage
}

// newRecordedOperation serializes an applied operation for upload. state is the
// local config after the operation and must be nil for delete operations. It
// errors when the serialized state exceeds maxOperationStateSize.
func newRecordedOperation(action deployplan.ActionType, resourceID string, state any, dependsOn []deployplan.DependsOnEntry) (recordedOperation, error) {
	actionType, err := deployActionToSDK(action)
	if err != nil {
		return recordedOperation{}, err
	}

	op := recordedOperation{
		action:     actionType,
		resourceID: resourceID,
		status:     bundledeployments.OperationStatusOperationStatusSucceeded,
	}

	// Operation.State carries the serialized state, which DMS serves back as
	// resource state. Unset for delete: the resource is gone.
	if state != nil {
		config, err := json.Marshal(state)
		if err != nil {
			return recordedOperation{}, fmt.Errorf("serializing state: %w", err)
		}
		raw, err := json.Marshal(dstate.RecordedState{State: config, DependsOn: dependsOn})
		if err != nil {
			return recordedOperation{}, fmt.Errorf("serializing state: %w", err)
		}
		if len(raw) > maxOperationStateSize {
			return recordedOperation{}, fmt.Errorf("serialized state is %d bytes, which exceeds the %d byte limit for recording deployment history", len(raw), maxOperationStateSize)
		}
		op.state = raw
	}

	return op, nil
}

// newFailedOperation records an operation that did not apply, so the deployment
// history says why a resource failed rather than just omitting it.
//
// priorState is the resource's state from before the deploy, carried through
// unchanged: an action other than a create acts on a resource that still exists, and
// the service rejects such an operation without state because dropping it would
// leave DMS unable to describe a resource it still owns. It is nil for a create,
// where there is no prior state and no resource to describe - which is also why the
// resourceID may be empty for CREATE and RECREATE.
func newFailedOperation(action deployplan.ActionType, resourceID string, priorState json.RawMessage, cause error) (recordedOperation, error) {
	actionType, err := deployActionToSDK(action)
	if err != nil {
		return recordedOperation{}, err
	}

	message := cause.Error()
	if len(message) > maxOperationErrorMessageSize {
		message = message[:maxOperationErrorMessageSize]
	}

	return recordedOperation{
		action:       actionType,
		resourceID:   resourceID,
		status:       bundledeployments.OperationStatusOperationStatusFailed,
		errorMessage: message,
		state:        priorState,
	}, nil
}

// priorState returns the resource's recorded state from before this deploy, in the
// same envelope form the success path uploads, or nil when the resource has none
// (a create). A failed operation reports this unchanged: the resource is whatever it
// was before the attempt.
func priorState(db *dstate.DeploymentState, resourceKey string) json.RawMessage {
	entry, ok := db.GetResourceEntry(resourceKey)
	if !ok || len(entry.State) == 0 {
		return nil
	}

	raw, err := json.Marshal(dstate.RecordedState{State: entry.State, DependsOn: entry.DependsOn})
	if err != nil {
		return nil
	}
	return raw
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
		ActionType:   op.action,
		ResourceId:   op.resourceID,
		ResourceKey:  dmsKey,
		Status:       op.status,
		ErrorMessage: op.errorMessage,
	}
	if op.state != nil {
		// DMS types state as a string, so the JSON goes on the wire as a quoted
		// string rather than an embedded object. The SDK field is a json.RawMessage,
		// so quote the payload here; sending the object directly is rejected with
		// "Invalid value: {...} for expected type: STRING".
		quoted, err := json.Marshal(string(op.state))
		if err != nil {
			return fmt.Errorf("serializing state: %w", err)
		}
		raw := json.RawMessage(quoted)
		operation.State = &raw
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
