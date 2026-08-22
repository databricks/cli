package dms

import (
	"encoding/json"
	"testing"

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

func writeState(t *testing.T, w OperationWriter, key ResourceKey, resourceID string) {
	t.Helper()
	update, err := NewStateUpdate(resourceID, json.RawMessage(`{"state":{}}`), false)
	require.NoError(t, err)
	require.NoError(t, w.Write(t.Context(), key, update))
}

func TestWriterSendsTheStagedSequenceThenWhatTheServiceReturns(t *testing.T) {
	// One operation per resource per version, so every write updates the same operation: the
	// first at the sequence id staging left, each one after at the id the service returned.
	f := newFakeRaw("7")
	w := testWriter(f)

	update, err := NewStateUpdate("job-123", json.RawMessage(`{"state":{}}`), false)
	require.NoError(t, err)
	require.NoError(t, w.Write(t.Context(), "jobs.foo", update))
	writeState(t, w, "jobs.foo", "job-456")

	require.Len(t, f.updates, 2)
	assert.Equal(t, updaterCall{
		deploymentID: "dep-1",
		version:      2,
		key:          "jobs.foo",
		sequenceID:   stagedSequenceID,
		update:       update,
	}, f.updates[0])
	assert.Equal(t, "7", f.updates[1].sequenceID)
}

func TestWriterTracksSequencePerResource(t *testing.T) {
	// Each resource has its own staged operation, so each one's first write echoes the staged
	// sequence id rather than a sequence another resource earned.
	f := newFakeRaw("7")
	w := testWriter(f)

	writeState(t, w, "jobs.foo", "id-1")
	writeState(t, w, "jobs.bar", "id-2")

	require.Len(t, f.updates, 2)
	assert.Equal(t, stagedSequenceID, f.updates[0].sequenceID)
	assert.Equal(t, stagedSequenceID, f.updates[1].sequenceID)
}

func TestWriterErrorKeepsTheSequence(t *testing.T) {
	// A failed write leaves the recorded sequence id alone, so the next write for that resource
	// still carries the precondition the service last gave us rather than nothing.
	f := newFakeRaw("9")
	f.failOn = 1
	w := testWriter(f)

	writeState(t, w, "jobs.foo", "job-1")

	update, err := NewStateUpdate("job-2", json.RawMessage(`{"state":{}}`), false)
	require.NoError(t, err)
	require.ErrorContains(t, w.Write(t.Context(), "jobs.foo", update), "injected error")

	writeState(t, w, "jobs.foo", "job-3")

	require.Len(t, f.updates, 3)
	assert.Equal(t, "9", f.updates[2].sequenceID)
}
