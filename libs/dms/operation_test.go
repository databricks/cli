package dms

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateUpdateRecordsEnvelopeAsIs(t *testing.T) {
	// The state DB serializes the envelope (see dstate.SaveState); the update carries it
	// through untouched, sensitive fields and all.
	state := json.RawMessage(`{"state":{"name":"foo","token":"super-secret"}}`)

	update, err := NewStateUpdate("job-123", state, false)
	require.NoError(t, err)

	assert.JSONEq(t, string(state), string(update.State))
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusSucceeded, update.Status)
	assert.Equal(t, DescribesResource, update.Fields)
}

func TestNewStateUpdateInProgressIsNotFinished(t *testing.T) {
	// A recreate's delete is half of a larger change, so an interrupted deploy must not
	// leave the resource described as finished.
	update, err := NewStateUpdate("", nil, true)
	require.NoError(t, err)

	assert.Equal(t, StatusInProgress, update.Status)
}

func TestNewStateUpdateRejectsOversizedState(t *testing.T) {
	big := json.RawMessage(strings.Repeat("x", maxStateSize+1))

	_, err := NewStateUpdate("job-123", big, false)
	assert.ErrorContains(t, err, "exceeds the 65536 byte limit")
}

func TestNewFailureUpdateRecordsError(t *testing.T) {
	update := NewFailureUpdate("", errors.New("cluster spec is invalid"))

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, update.Status)
	assert.Equal(t, "cluster spec is invalid", update.ErrorMessage)
	// The resource was never written, so there is no state to serve back for it.
	assert.Nil(t, update.State)
	// The update only marks the operation failed; see KeepsState.
	assert.Equal(t, KeepsState, update.Fields)
}

func TestNewFailureUpdateTruncatesLongError(t *testing.T) {
	// Truncated rather than rejected: a message over the limit would make recording
	// fail and hide the error it is reporting.
	update := NewFailureUpdate("job-123", errors.New(strings.Repeat("x", maxErrorMessageSize+100)))

	assert.Len(t, update.ErrorMessage, maxErrorMessageSize)
}

func TestNewFailureUpdatePreservesUTF8OnTruncation(t *testing.T) {
	// The cut lands one byte into the emoji, so a byte-wise truncation would leave a partial
	// rune behind and the service stores state and messages as strings.
	msg := strings.Repeat("a", maxErrorMessageSize-1) + "❌" + "x"

	update := NewFailureUpdate("job-123", errors.New(msg))

	assert.True(t, utf8.ValidString(update.ErrorMessage))
	// The whole emoji went, so the message is shorter than the limit rather than exactly it.
	assert.Equal(t, strings.Repeat("a", maxErrorMessageSize-1), update.ErrorMessage)
}

func TestMergeLetsAWriteSupersedeAFailure(t *testing.T) {
	// A failure is not the last word. A retry that writes state wins whole - state, id and
	// mask - and the mask names error_message so the recorded failure is cleared. The service
	// rejects a succeeded operation that still carries an error.
	failed := NewFailureUpdate("id-old", errors.New("boom"))
	retried, err := NewStateUpdate("id-new", json.RawMessage(`{"state":{"name":"after"}}`), false)
	require.NoError(t, err)

	merged := failed.Merge(retried)

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusSucceeded, merged.Status)
	assert.Empty(t, merged.ErrorMessage)
	assert.Equal(t, "id-new", merged.ResourceID)
	assert.JSONEq(t, `{"state":{"name":"after"}}`, string(merged.State))
	assert.Equal(t, DescribesResource, merged.Fields)
}

func TestMergeKeepsTheWritesStateAndMask(t *testing.T) {
	// A failure claims only status and error_message, so the write's state, id and mask
	// survive.
	write, err := NewStateUpdate("id-new", nil, false)
	require.NoError(t, err)
	failed := NewFailureUpdate("id-old", errors.New("boom"))

	merged := write.Merge(failed)

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, merged.Status)
	assert.Equal(t, "boom", merged.ErrorMessage)
	assert.Equal(t, "id-new", merged.ResourceID)
	assert.Nil(t, merged.State)
	assert.Equal(t, DescribesResource, merged.Fields)
}

func TestMergeLetsADeleteClearTheState(t *testing.T) {
	// A delete legitimately carries no state, and merging must let it through: the resource
	// is gone, and keeping the state it replaces would leave it listed.
	write, err := NewStateUpdate("id-1", json.RawMessage(`{"state":{"name":"before"}}`), false)
	require.NoError(t, err)
	deleted, err := NewStateUpdate("id-1", nil, false)
	require.NoError(t, err)

	merged := write.Merge(deleted)

	assert.Nil(t, merged.State)
}
