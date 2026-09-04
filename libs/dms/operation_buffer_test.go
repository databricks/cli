package dms

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// What the service ends up holding - the version a deploy claims, the operations it stages, the
// completion it reports - is asserted end to end by acceptance/bundle/dms. What is left here is
// what a deploy cannot reach: the queue's coalescing and the size limit. No test here goes
// through the API - the queue is driven directly, so nothing waits on a request.

// queued builds a buffer whose queue nothing drains, so a test drives record and take itself.
func queued() *OperationBuffer {
	return &OperationBuffer{
		queue:       make(chan string, bufferedOperations),
		pending:     make(map[string]OperationUpdate),
		latestState: make(map[string]OperationUpdate),
	}
}

func stateUpdate(t *testing.T, name string) OperationUpdate {
	t.Helper()
	update, err := NewStateUpdate("id-1", json.RawMessage(`{"state":{"name":"`+name+`"}}`), false)
	require.NoError(t, err)
	return update
}

func TestBufferCoalescesWhileAKeyIsPending(t *testing.T) {
	s := queued()

	// Two writes for one resource with nothing draining. They carry the resource's full
	// state, so only the newest needs to go: one slot in the queue, not two.
	s.record("resources.jobs.foo", stateUpdate(t, "v1"))
	s.record("resources.jobs.foo", stateUpdate(t, "v2"))

	assert.Len(t, s.queue, 1)
	update, ok := s.take("resources.jobs.foo")
	require.True(t, ok)
	assert.JSONEq(t, `{"state":{"name":"v2"}}`, string(update.State))

	// Taken means a request has it, and an in-flight request cannot be recalled, so the next
	// write gets its own slot rather than joining it.
	s.record("resources.jobs.foo", stateUpdate(t, "v3"))
	assert.Len(t, s.queue, 2)
	update, ok = s.take("resources.jobs.foo")
	require.True(t, ok)
	assert.JSONEq(t, `{"state":{"name":"v3"}}`, string(update.State))
}

func TestBufferFailsOnOversizedState(t *testing.T) {
	// The service will not take a state this large, so the resource cannot be recorded.
	// Failing here says so, where reporting nothing would leave DMS without the resource
	// and the next plan would create it again.
	s := queued()

	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", json.RawMessage(strings.Repeat("x", maxStateSize+1)))

	assert.ErrorContains(t, s.Err(), "exceeds the 65536 byte limit")
	assert.Empty(t, s.queue)
	assert.Empty(t, s.pending)
}

func TestDrainIsIdempotent(t *testing.T) {
	// Nothing recorded, so no transport is needed: this is the second drain, which must not
	// panic on an already closed queue.
	s := queued()
	s.done = make(chan struct{})
	s.stopQueue = sync.OnceFunc(func() { close(s.queue) })
	go s.run(t.Context())

	require.NoError(t, s.Drain())
	require.NoError(t, s.Drain())
}

func TestRecordFailureKeepsTheStateTheLastWriteRecorded(t *testing.T) {
	s := queued()

	// A write records the resource's post-apply state - a rename, so a new id and new state - and
	// the writer sends it (take), leaving nothing pending.
	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-new", json.RawMessage(`{"state":{"name":"v2"}}`))
	_, ok := s.take("resources.jobs.foo")
	require.True(t, ok)
	require.Empty(t, s.pending)

	// The post-update wait then fails. The failure carries the pre-deploy state (old id, old
	// state), which it must not use: the write already recorded v2, which the service may hold.
	s.RecordFailure("resources.jobs.foo", "id-old", json.RawMessage(`{"state":{"name":"v1"}}`), errors.New("wait timed out"))

	update, ok := s.take("resources.jobs.foo")
	require.True(t, ok)
	assert.JSONEq(t, `{"state":{"name":"v2"}}`, string(update.State), "keeps the state the write recorded")
	assert.Equal(t, "id-new", update.ResourceID, "keeps the id the write recorded, not the pre-deploy id")
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, update.Status)
	assert.Contains(t, update.ErrorMessage, "wait timed out")
}

func TestRecordFailureUsesTheGivenStateWhenNoWriteRecorded(t *testing.T) {
	// No write recorded anything - the operation failed before it saved state, e.g. a create that
	// never landed - so the failure falls back to the state passed in, what the resource still has.
	s := queued()

	s.RecordFailure("resources.jobs.foo", "id-1", json.RawMessage(`{"state":{"name":"v1"}}`), errors.New("create failed"))

	update, ok := s.take("resources.jobs.foo")
	require.True(t, ok)
	assert.JSONEq(t, `{"state":{"name":"v1"}}`, string(update.State))
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, update.Status)
}
