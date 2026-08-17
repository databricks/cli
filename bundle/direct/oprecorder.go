package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/databricks-sdk-go/client"
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

// operationStatusInProgress marks an operation whose writes are not finished; see
// dstate.OperationInfo.InProgress for when a write asks for it.
//
// Declared here rather than used from the SDK: the enum value is generated from the
// OpenAPI spec, which trails the service proto (databricks-eng/universe#2394529).
const operationStatusInProgress bundledeployments.OperationStatus = "OPERATION_STATUS_IN_PROGRESS"

// recordedOperation is an applied resource operation, serialized and waiting to be
// uploaded to the deployment metadata service (DMS).
//
// The payload is built on the apply worker rather than in the uploader so the sink
// does not hold on to the live resource struct, and so a malformed state fails the
// resource that produced it instead of the drain at the end of apply.
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

	// updateFields is the update mask to send if this operation updates one the service
	// already has. The service takes it literally: a field named here is written, a field
	// left out keeps the value it had.
	updateFields []string
}

// describesResource is the update mask for an operation that says how the resource
// looks: everything an update is allowed to change.
var describesResource = []string{"state", "error_message", "resource_id", "status"}

// failedKeepingState is the update mask for a failure. A failure says why the
// resource stopped rather than how it looks, and the state it reports is the pre-deploy
// one - older than whatever a write already recorded - so it leaves state alone.
var failedKeepingState = []string{"error_message", "status"}

// isFailure reports whether the operation records a resource that did not apply.
func (op recordedOperation) isFailure() bool {
	return op.status == bundledeployments.OperationStatusOperationStatusFailed
}

// checkStateSize rejects a state the service will not accept.
func checkStateSize(state json.RawMessage) error {
	if len(state) > maxOperationStateSize {
		return fmt.Errorf("serialized state is %d bytes, which exceeds the %d byte limit for recording deployment history", len(state), maxOperationStateSize)
	}
	return nil
}

// newStateOperation describes a state write for upload. state is the serialized
// RecordedState envelope the state DB just persisted, and nil for a delete, where
// the resource is gone. It errors when the state exceeds maxOperationStateSize.
func newStateOperation(info dstate.OperationInfo, resourceID string, state json.RawMessage) (recordedOperation, error) {
	actionType, err := deployActionToSDK(info.Action)
	if err != nil {
		return recordedOperation{}, err
	}

	if err := checkStateSize(state); err != nil {
		return recordedOperation{}, err
	}

	status := bundledeployments.OperationStatusOperationStatusSucceeded
	if info.InProgress {
		status = operationStatusInProgress
	}

	return recordedOperation{
		action:       actionType,
		resourceID:   resourceID,
		status:       status,
		state:        state,
		updateFields: describesResource,
	}, nil
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

	// A guard: this state came back from the state DB, so it was within the limit when
	// it was written.
	if err := checkStateSize(priorState); err != nil {
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
		updateFields: failedKeepingState,
	}, nil
}

// priorRecord returns the resource's id and state from before this deploy, in the
// same envelope form the success path uploads, or empty values when the resource has
// no prior record (a create). A failed operation reports these unchanged: the resource
// is whatever it was before the attempt.
//
// Both come from the same pre-deploy entry because the service requires an id
// alongside state: state describes a resource that exists, so it needs the id to say
// which one. Reading the id from live state instead would return "" for a failed
// recreate, whose delete step already dropped it, and the mismatch is rejected.
func priorRecord(db *dstate.DeploymentState, resourceKey string) (string, json.RawMessage) {
	entry, ok := db.GetResourceEntry(resourceKey)
	if !ok || len(entry.State) == 0 {
		return "", nil
	}

	raw, err := json.Marshal(dstate.RecordedState{State: entry.State, DependsOn: entry.DependsOn})
	if err != nil {
		return "", nil
	}
	return entry.ID, raw
}

// operationUploader records an applied resource operation with DMS. Uploads run on
// the operationSink goroutine, off the apply path.
type operationUploader interface {
	upload(ctx context.Context, resourceKey string, op recordedOperation) error
}

// operationRecorder uploads operations via the DMS operations API.
type operationRecorder struct {
	ops operationClient
	// parent is the version the operations are recorded under, formatted as
	// "deployments/{deployment_id}/versions/{version_id}".
	parent string

	// mu guards recorded.
	mu sync.Mutex

	// recorded holds what the service has per resource key, which is how a resource
	// already recorded in this version is recognised. The service names operations
	// "operations/{resource_key}", so it keeps one per resource per version: the second
	// write for a resource has to update that operation.
	recorded map[string]recordedState
}

// NewOperationRecorder returns an operationUploader backed by the DMS operations
// API. deploymentID and version identify the deployment version assigned by DMS
// that the operations are recorded under.
func NewOperationRecorder(apiClient *client.DatabricksClient, deploymentID string, version int64) operationUploader {
	return newOperationRecorder(newAPIOperationClient(apiClient), deploymentID, version)
}

// newOperationRecorder is the internal constructor, so tests can supply their own
// operationClient.
func newOperationRecorder(ops operationClient, deploymentID string, version int64) operationUploader {
	return &operationRecorder{
		ops:      ops,
		parent:   fmt.Sprintf("deployments/%s/versions/%d", deploymentID, version),
		recorded: make(map[string]recordedState),
	}
}

// recordedState is what the service holds for a resource in this version.
type recordedState struct {
	// sequenceID is echoed as the concurrency precondition on the next update.
	sequenceID string

	// hasState says whether the recorded operation describes the resource. A failure
	// must not disturb that description, and must supply one when it is missing.
	hasState bool
}

func (r *operationRecorder) upload(ctx context.Context, resourceKey string, op recordedOperation) error {
	// The read path re-adds the prefix; see dstate.ResourceKeyPrefix.
	dmsKey := strings.TrimPrefix(resourceKey, dstate.ResourceKeyPrefix)

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

	r.mu.Lock()
	has, recorded := r.recorded[dmsKey]
	r.mu.Unlock()

	mask := op.updateFields
	if !has.hasState {
		// Nothing recorded describes the resource yet, so this operation has to, even a
		// failure that would rather leave state alone: a resource recorded without state
		// is dropped from the deployment, and the next plan creates it again.
		mask = describesResource
	}

	var result operationResponse
	var err error
	if recorded {
		// Only the masked fields and sequence_id are read on an update; action_type
		// stays as the operation was created, so sending it would just be misleading.
		result, err = r.ops.UpdateOperation(ctx, r.parent, dmsKey, mask, updateOperationRequest{
			State:        operation.State,
			ErrorMessage: operation.ErrorMessage,
			ResourceId:   operation.ResourceId,
			Status:       operation.Status,
			SequenceId:   has.sequenceID,
		})
	} else {
		result, err = r.ops.CreateOperation(ctx, r.parent, dmsKey, operation)
	}
	if err != nil {
		return err
	}

	// Remember what the service now holds, so the next write for this resource updates
	// rather than re-creates, and a failure can tell whether the resource is already
	// described. A mask that left state out leaves whatever was there.
	hasState := has.hasState
	if slices.Contains(mask, "state") {
		hasState = op.state != nil
	}

	r.mu.Lock()
	r.recorded[dmsKey] = recordedState{sequenceID: result.SequenceId, hasState: hasState}
	r.mu.Unlock()

	return nil
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
