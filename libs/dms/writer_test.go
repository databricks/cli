package dms

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWriter returns a writer for version 2 of dep-1, recording through raw.
func testWriter(raw *fakeRaw) OperationWriter {
	return &operationWriter{
		client:       &Client{raw: raw},
		deploymentID: "dep-1",
		version:      2,
		sequenceIDs:  make(map[ResourceKey]string),
	}
}

func writeState(t *testing.T, w OperationWriter, key ResourceKey, resourceID string, state json.RawMessage) {
	t.Helper()
	update, err := NewStateUpdate(resourceID, state, false)
	require.NoError(t, err)
	require.NoError(t, w.Write(t.Context(), key, update))
}

func TestWriterFirstWriteUpdatesTheStagedOperation(t *testing.T) {
	f := newFakeRaw("1")
	w := testWriter(f)

	writeState(t, w, "jobs.foo", "job-123", json.RawMessage(`{"state":{}}`))

	require.Len(t, f.updates, 1)
	c := f.updates[0]
	// The version already staged this operation, so the first write updates it and echoes
	// the sequence id staging left.
	assert.Equal(t, "dep-1", c.deploymentID)
	assert.Equal(t, int64(2), c.version)
	assert.Equal(t, ResourceKey("jobs.foo"), c.key)
	assert.Equal(t, stagedSequenceID, c.sequenceID)
	assert.Equal(t, "job-123", c.update.ResourceID)
}

func TestWriterSecondWriteEchoesTheServiceSequence(t *testing.T) {
	// One operation per resource per version: the second write updates the same operation,
	// echoing the sequence id the service returned as its precondition.
	f := newFakeRaw("7")
	w := testWriter(f)

	writeState(t, w, "jobs.foo", "", nil)
	writeState(t, w, "jobs.foo", "job-456", json.RawMessage(`{"state":{}}`))

	require.Len(t, f.updates, 2)
	assert.Equal(t, stagedSequenceID, f.updates[0].sequenceID)
	assert.Equal(t, "7", f.updates[1].sequenceID)
}

func TestWriterTracksSequencePerResource(t *testing.T) {
	// Each resource has its own staged operation, so each one's first write echoes the staged
	// sequence id rather than a sequence another resource earned.
	f := newFakeRaw("1")
	w := testWriter(f)

	writeState(t, w, "jobs.foo", "id-1", json.RawMessage(`{"state":{}}`))
	writeState(t, w, "jobs.bar", "id-2", json.RawMessage(`{"state":{}}`))

	require.Len(t, f.updates, 2)
	assert.Equal(t, stagedSequenceID, f.updates[0].sequenceID)
	assert.Equal(t, stagedSequenceID, f.updates[1].sequenceID)
}

func TestWriterErrorKeepsTheSequence(t *testing.T) {
	// A failed write returns its error and leaves the recorded sequence id alone, so a later
	// write for the same resource still carries the precondition the service last gave us.
	f := newFakeRaw("9")
	f.failOn = 1
	w := testWriter(f)

	writeState(t, w, "jobs.foo", "job-1", json.RawMessage(`{"state":{}}`))

	second, err := NewStateUpdate("job-2", json.RawMessage(`{"state":{}}`), false)
	require.NoError(t, err)
	err = w.Write(t.Context(), "jobs.foo", second)
	require.ErrorContains(t, err, "injected error")

	// The third write is what proves the sequence id survived the failure.
	writeState(t, w, "jobs.foo", "job-3", json.RawMessage(`{"state":{}}`))

	require.Len(t, f.updates, 3)
	assert.Equal(t, "9", f.updates[2].sequenceID)
}

func TestUpdateRequestSendsOnlyWhatTheMaskNames(t *testing.T) {
	// A failure keeps whatever state an earlier write recorded, so it must send neither
	// state nor resource_id: an empty state would drop the resource from the deployment.
	failure := NewFailureUpdate("job-1", errors.New("boom"))

	body := newUpdateRequest(failure, "3")

	assert.Empty(t, body.State)
	assert.Empty(t, body.ResourceId)
	assert.Equal(t, "3", body.SequenceId)
	assert.Equal(t, "boom", body.ErrorMessage)
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, body.Status)
}

func TestUpdateRequestSendsStateWhenNamed(t *testing.T) {
	update, err := NewStateUpdate("job-1", json.RawMessage(`{"state":{"name":"foo"}}`), false)
	require.NoError(t, err)

	body := newUpdateRequest(update, stagedSequenceID)

	assert.JSONEq(t, `{"state":{"name":"foo"}}`, body.State)
	assert.Equal(t, "job-1", body.ResourceId)
}

func TestUpdateRequestSendsEachFieldOnItsOwnMaskEntry(t *testing.T) {
	// resource_id does not travel with state: a mask that names one and not the other sends
	// exactly that.
	update := OperationUpdate{
		Fields:     FieldResourceID | FieldStatus,
		State:      json.RawMessage(`{"state":{"name":"foo"}}`),
		ResourceID: "job-1",
		Status:     bundledeployments.OperationStatusOperationStatusSucceeded,
	}

	body := newUpdateRequest(update, "4")

	assert.Empty(t, body.State)
	assert.Equal(t, "job-1", body.ResourceId)
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusSucceeded, body.Status)
}
