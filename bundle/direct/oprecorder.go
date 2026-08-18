package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// maxOperationStateSize is the largest serialized state DMS accepts per operation. More is
// rejected server-side, so fail early with a message naming the resource.
const maxOperationStateSize = 64 * 1024

// maxOperationErrorMessageSize is the largest error message DMS accepts. A longer one is
// truncated rather than rejected, so recording cannot fail and hide the error it reports.
const maxOperationErrorMessageSize = 16 * 1024

// operationStatusInProgress marks an operation whose writes are not finished. Not taken from
// the SDK: the enum is generated from the OpenAPI spec, which trails the service proto
// (databricks-eng/universe#2394529).
const operationStatusInProgress bundledeployments.OperationStatus = "OPERATION_STATUS_IN_PROGRESS"

// stagedSequenceID is what CreateVersion leaves on every operation it stages, and so the
// precondition for the first update of a resource.
const stagedSequenceID = "0"

// recordedOperation is an applied resource operation waiting to be uploaded. It is built on
// the apply worker, not in the uploader, so a malformed state fails the resource that
// produced it rather than the drain at the end of apply.
type recordedOperation struct {
	resourceID string
	status     bundledeployments.OperationStatus

	// errorMessage is set only when status is failed, which the service enforces.
	errorMessage string

	// state is the serialized config after the operation: nil for a delete, and the
	// pre-deploy state for a failure (see newFailedOperation).
	state json.RawMessage

	// updateFields is the mask to send when updating an operation the service already has.
	// It is taken literally: a field named here is written, one left out keeps its value.
	updateFields []string
}

// describesResource is the update mask for an operation that says how the resource looks:
// every field an update may change. Any other path is rejected with INVALID_PARAMETER_VALUE,
// so this doubles as the canonical field list and order.
var describesResource = []string{"state", "error_message", "resource_id", "status"}

// failedKeepingState is the update mask for a failure: mark it failed and leave state alone.
// State means the resource is as it was written; no state means a delete went through and
// nothing replaced it, so the resource really is gone and the deployment should say so.
var failedKeepingState = []string{"error_message", "status"}

// newStateOperation describes a state write for upload. state is the serialized
// RecordedState envelope the state DB just persisted, and nil for a delete, where
// the resource is gone. It errors when the state exceeds maxOperationStateSize.
func newStateOperation(info dstate.OperationInfo, resourceID string, state json.RawMessage) (recordedOperation, error) {
	// The action is not recorded - the version stages it - but an unrecordable one means
	// there is no operation to update, so fail the resource rather than the whole drain.
	if _, err := DeployActionToSDK(info.Action); err != nil {
		return recordedOperation{}, err
	}

	if len(state) > maxOperationStateSize {
		return recordedOperation{}, fmt.Errorf("serialized state is %d bytes, which exceeds the %d byte limit for recording deployment history", len(state), maxOperationStateSize)
	}

	status := bundledeployments.OperationStatusOperationStatusSucceeded
	if info.InProgress {
		status = operationStatusInProgress
	}

	return recordedOperation{
		resourceID:   resourceID,
		status:       status,
		state:        state,
		updateFields: describesResource,
	}, nil
}

// newFailedOperation records an operation that did not apply, so the history says why a
// resource failed rather than leaving it pending. It carries no state: the version staged the
// operation already, so a failure only ever narrows an existing record.
func newFailedOperation(action deployplan.ActionType, resourceID string, cause error) (recordedOperation, error) {
	if _, err := DeployActionToSDK(action); err != nil {
		return recordedOperation{}, err
	}

	// Summarized, not cause.Error(): for an API failure that adds the status and error
	// code, which is often the most actionable part of the history.
	message := diag.FormatAPIErrorSummary(cause)
	if len(message) > maxOperationErrorMessageSize {
		message = message[:maxOperationErrorMessageSize]
		// The cut can land inside a rune, and the service stores a string. Drop the partial
		// one: at most UTFMax-1 bytes of it can be left, so a message that was already
		// invalid loses those bytes rather than being stripped away entirely.
		for range utf8.UTFMax - 1 {
			if utf8.ValidString(message) {
				break
			}
			message = message[:len(message)-1]
		}
	}

	return recordedOperation{
		resourceID:   resourceID,
		status:       bundledeployments.OperationStatusOperationStatusFailed,
		errorMessage: message,
		updateFields: failedKeepingState,
	}, nil
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

	// mu guards sequenceIDs.
	mu sync.Mutex

	// sequenceIDs holds the sequence id the service last returned per resource key, echoed as
	// the precondition on the next update. A key absent from the map has not been written yet,
	// so its staged operation is still at stagedSequenceID.
	sequenceIDs map[string]string
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
		ops:         ops,
		parent:      fmt.Sprintf("deployments/%s/versions/%d", deploymentID, version),
		sequenceIDs: make(map[string]string),
	}
}

func (r *operationRecorder) upload(ctx context.Context, resourceKey string, op recordedOperation) error {
	// The read path re-adds the prefix; see dstate.ResourceKeyPrefix.
	dmsKey := strings.TrimPrefix(resourceKey, dstate.ResourceKeyPrefix)

	operation := bundledeployments.Operation{
		ResourceId:   op.resourceID,
		ResourceKey:  dmsKey,
		Status:       op.status,
		ErrorMessage: op.errorMessage,
		// The service stores state as an opaque string, so the serialized envelope goes
		// on the wire as-is. Empty means unset, which is what a delete records.
		State: string(op.state),
	}

	r.mu.Lock()
	sequenceID, written := r.sequenceIDs[dmsKey]
	r.mu.Unlock()
	if !written {
		sequenceID = stagedSequenceID
	}

	update := updateOperationRequest{
		ErrorMessage: operation.ErrorMessage,
		Status:       operation.Status,
		SequenceId:   sequenceID,
	}
	// Send only what the mask names. The service would ignore the rest, and state is the
	// largest field by far, so a failure that keeps the recorded state sends none.
	if slices.Contains(op.updateFields, "state") {
		update.State = operation.State
		update.ResourceId = operation.ResourceId
	}

	// action_type is fixed when the version stages the operation, so it is not sent.
	result, err := r.ops.UpdateOperation(ctx, r.parent, dmsKey, op.updateFields, update)
	if err != nil {
		return err
	}

	// The next write for this resource echoes the sequence id this one earned.
	r.mu.Lock()
	r.sequenceIDs[dmsKey] = result.SequenceId
	r.mu.Unlock()

	return nil
}

// DeployActionToSDK maps a deployplan action to its DMS operation action type.
// Only actions that mutate a resource are recordable; Skip and Undefined never
// reach a recorder and are rejected rather than silently coerced.
func DeployActionToSDK(a deployplan.ActionType) (bundledeployments.OperationActionType, error) {
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
