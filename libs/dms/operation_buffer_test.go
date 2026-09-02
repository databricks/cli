package dms

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

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
		queue:   make(chan string, bufferedOperations),
		pending: make(map[string]OperationUpdate),
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
