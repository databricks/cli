package dms

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWriter returns a writer for version 2 of dep-1, recording through c.
func testWriter(c *Client) *OperationWriter {
	return &OperationWriter{
		client:       c,
		deploymentID: "dep-1",
		version:      2,
		sequenceIDs:  make(map[ResourceKey]string),
	}
}

func writeState(t *testing.T, w *OperationWriter, key ResourceKey, resourceID string) error {
	t.Helper()
	update, err := NewStateUpdate(resourceID, json.RawMessage(`{"state":{}}`), false)
	require.NoError(t, err)
	return w.Write(t.Context(), key, update)
}

func TestWriterSendsTheStagedSequenceThenWhatTheServiceReturns(t *testing.T) {
	// One operation per resource per version, so every write updates the same operation: the
	// first at the sequence id staging left, each one after at the id the service returned.
	c, api := newFakeAPI(t, reply{status: http.StatusOK, body: `{"sequence_id":"7"}`})
	w := testWriter(c)

	require.NoError(t, writeState(t, w, "jobs.foo", "job-123"))
	require.NoError(t, writeState(t, w, "jobs.foo", "job-456"))

	sent := api.sent()
	require.Len(t, sent, 2)
	assert.Equal(t, "/api/2.0/bundle/deployments/dep-1/versions/2/operations/jobs.foo", sent[0].path)
	assert.Contains(t, sent[0].body, `"sequence_id":"`+stagedSequenceID+`"`)
	assert.Contains(t, sent[1].body, `"sequence_id":"7"`)
}

func TestWriterTracksSequencePerResource(t *testing.T) {
	// Each resource has its own staged operation, so each one's first write echoes the staged
	// sequence id rather than a sequence another resource earned.
	c, api := newFakeAPI(t, reply{status: http.StatusOK, body: `{"sequence_id":"7"}`})
	w := testWriter(c)

	require.NoError(t, writeState(t, w, "jobs.foo", "id-1"))
	require.NoError(t, writeState(t, w, "jobs.bar", "id-2"))

	sent := api.sent()
	require.Len(t, sent, 2)
	assert.Contains(t, sent[0].body, `"sequence_id":"`+stagedSequenceID+`"`)
	assert.Contains(t, sent[1].body, `"sequence_id":"`+stagedSequenceID+`"`)
}

func TestWriterErrorKeepsTheSequence(t *testing.T) {
	// A failed write leaves the recorded sequence id alone, so the next write for that resource
	// still carries the precondition the service last gave us rather than nothing.
	c, api := newFakeAPI(t,
		reply{status: http.StatusOK, body: `{"sequence_id":"9"}`},
		reply{status: http.StatusBadRequest, body: `{"error_code":"INVALID_PARAMETER_VALUE","message":"boom"}`},
		reply{status: http.StatusOK, body: `{"sequence_id":"11"}`},
	)
	w := testWriter(c)

	require.NoError(t, writeState(t, w, "jobs.foo", "job-1"))
	require.ErrorContains(t, writeState(t, w, "jobs.foo", "job-2"), "boom")
	require.NoError(t, writeState(t, w, "jobs.foo", "job-3"))

	sent := api.sent()
	require.Len(t, sent, 3)
	assert.Contains(t, sent[2].body, `"sequence_id":"9"`)
}
