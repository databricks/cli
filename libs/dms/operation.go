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

// StatusInProgress marks an operation whose writes are not finished. Not taken from the
// SDK: the enum is generated from the OpenAPI spec, which trails the service proto
// (databricks-eng/universe#2394529).
const StatusInProgress bundledeployments.OperationStatus = "OPERATION_STATUS_IN_PROGRESS"

// OperationUpdate is one write to an operation the version staged: the fields it claims
// and their values. It is built where the outcome is known, so a malformed one fails the
// resource that produced it rather than the upload at the end of apply.
type OperationUpdate struct {
	// Fields is the mask to send. It is taken literally: a field named here is written,
	// one left out keeps its value. See the package doc.
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
		status = StatusInProgress
	}

	return OperationUpdate{
		Fields:     DescribesResource,
		State:      state,
		ResourceID: resourceID,
		Status:     status,
	}, nil
}

// NewFailureUpdate records that an operation did not apply, so the history says why a
// resource failed rather than leaving it pending. It claims no state, so whatever an
// earlier write recorded stands.
func NewFailureUpdate(resourceID string, cause error) OperationUpdate {
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

	return OperationUpdate{
		Fields:       KeepsState,
		ResourceID:   resourceID,
		Status:       bundledeployments.OperationStatusOperationStatusFailed,
		ErrorMessage: message,
	}
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
