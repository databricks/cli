package dms

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// maxStateSize is the largest serialized state DMS accepts per operation. More than this
// and the resource cannot be recorded at all, so the deploy fails rather than leaving the
// service holding a resource with no state.
const maxStateSize = 64 * 1024

// maxErrorMessageSize is how much of a failure's message the service stores.
const maxErrorMessageSize = 16 * 1024

// StatusPending is what the service calls an operation it has recorded but not applied, which
// is where a recreate sits between deleting the old resource and creating the new one. Not in
// the SDK: the enum ships server-first while it is staged DEVELOPMENT, so the generated client
// carries only the terminal two.
const StatusPending bundledeployments.OperationStatus = "OPERATION_STATUS_PENDING"

// OperationUpdate is one write to an operation the version staged: the fields it claims
// and their values. It is built where the outcome is known, so a malformed one fails the
// resource that produced it rather than the upload at the end of apply.
type OperationUpdate struct {
	// Fields is the mask to send. It is taken literally: a field named here is written,
	// one left out keeps the value it had.
	Fields Fields

	// State is the serialized state after the operation, and nil for a delete.
	State json.RawMessage

	ResourceID   string
	Status       bundledeployments.OperationStatus
	ErrorMessage string
}

// NewStateUpdate describes how the resource looks now: state is the serialized envelope the
// state DB just persisted, and nil for a delete. inProgress marks a write that is half of a
// larger change - a recreate's delete - so an interrupted deploy does not report it finished.
func NewStateUpdate(resourceID string, state json.RawMessage, inProgress bool) (OperationUpdate, error) {
	if len(state) > maxStateSize {
		return OperationUpdate{}, fmt.Errorf("serialized state is %d bytes, which exceeds the %d byte limit for recording deployment history", len(state), maxStateSize)
	}

	status := bundledeployments.OperationStatusOperationStatusSucceeded
	if inProgress {
		status = StatusPending
	}

	// No state means nothing is left to describe, so the write clears it rather than recording
	// it as empty: an empty value is a state the service refuses on a delete that succeeded,
	// while an absent one is how the resource stops being listed at all.
	if state == nil {
		return OperationUpdate{
			Fields: ClearsState,
			Status: status,
		}, nil
	}

	return OperationUpdate{
		Fields:     DescribesResource,
		State:      state,
		ResourceID: resourceID,
		Status:     status,
	}, nil
}

// failureMessage summarizes cause for the history, cut to what the service stores.
func failureMessage(cause error) string {
	// Summarized, not cause.Error(): for an API failure that adds the status and error
	// code, which is often the most actionable part of the history.
	message := diag.FormatAPIErrorSummary(cause)
	if len(message) > maxErrorMessageSize {
		message = message[:maxErrorMessageSize]
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
	return message
}

// failureUpdate marks an operation failed and carries the message, with no state of its own: it
// claims KeepsState so a Merge onto the resource's last write keeps that write's state and id.
// OperationBuffer.RecordFailure uses it when a write already recorded the resource's state.
func failureUpdate(cause error) OperationUpdate {
	return OperationUpdate{
		Fields:       KeepsState,
		Status:       bundledeployments.OperationStatusOperationStatusFailed,
		ErrorMessage: failureMessage(cause),
	}
}

// NewFailureUpdate records that an operation did not apply, so the history says why a resource
// failed rather than leaving it pending. state is what the resource still has, which the service
// requires of a failure on a live one; nil when nothing is left to describe.
func NewFailureUpdate(resourceID string, state json.RawMessage, cause error) OperationUpdate {
	u := failureUpdate(cause)
	if state != nil {
		// The id travels with the state, which the service requires of any mask naming one.
		u.Fields = DescribesResource
		u.State = state
		u.ResourceID = resourceID
	}
	return u
}

// Merge folds a later update into u, for a resource written twice before either upload ran.
// Each field comes from whichever update claimed it, newer winning when both did, and the
// mask is the union. What an update claims is decided where it is built, not here.
func (u OperationUpdate) Merge(newer OperationUpdate) OperationUpdate {
	merged := u
	merged.Fields = u.Fields | newer.Fields

	if newer.Fields.Has(FieldState) {
		merged.State = newer.State
	}
	if newer.Fields.Has(FieldResourceID) {
		merged.ResourceID = newer.ResourceID
	}
	if newer.Fields.Has(FieldErrorMessage) {
		merged.ErrorMessage = newer.ErrorMessage
	}
	if newer.Fields.Has(FieldStatus) {
		merged.Status = newer.Status
	}

	return merged
}
