package dms

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWriter records the writes it receives and optionally blocks until block is
// closed, so a test can hold the writer and observe what coalesces behind it.
//
// Keys arrive in the DMS form, which is what the sink puts on the wire.
type fakeWriter struct {
	block   chan struct{}
	started chan ResourceKey
	err     error

	mu     sync.Mutex
	writes []string
}

func (f *fakeWriter) Write(ctx context.Context, key ResourceKey, update OperationUpdate) error {
	if f.started != nil {
		f.started <- key
	}
	if f.block != nil {
		<-f.block
	}

	f.mu.Lock()
	f.writes = append(f.writes, string(key)+"="+string(update.State))
	f.mu.Unlock()

	return f.err
}

func (f *fakeWriter) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.writes...)
}

// envelope is the serialized state the state DB hands the sink. The sink passes it through
// untouched, so the shape only has to look like what goes on the wire.
func envelope(name string) json.RawMessage {
	return json.RawMessage(`{"state":{"name":"` + name + `"}}`)
}

func recordState(t *testing.T, s *OperationSink, resourceKey, name string) {
	t.Helper()
	s.RecordOperation(t.Context(), resourceKey, false, "id-1", envelope(name))
}

func TestOperationSinkKeepsWritingAfterGoingIdle(t *testing.T) {
	// The writer parks on an empty queue instead of returning. Apply can spend long stretches
	// inside resource CRUD with nothing to record, and a writer that exited while idle would
	// silently drop everything recorded after it.
	f := &fakeWriter{}
	s := NewOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	require.Eventually(t, func() bool { return len(f.recorded()) == 1 }, 5*time.Second, time.Millisecond)

	// The queue is drained and the writer idle; what is recorded now still has to go.
	recordState(t, s, "resources.jobs.bar", "v1")
	require.NoError(t, s.Close())

	assert.Len(t, f.recorded(), 2)
}

func TestOperationSinkCoalescesWritesBehindAWrite(t *testing.T) {
	// Hold the writer on the first write so the two behind it pile up. They carry
	// the resource's full state, so only the newest needs to go: the resource costs
	// two requests rather than three.
	f := &fakeWriter{block: make(chan struct{}), started: make(chan ResourceKey, 2)}
	s := NewOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	assert.Equal(t, ResourceKey("jobs.foo"), <-f.started)

	recordState(t, s, "resources.jobs.foo", "v2")
	recordState(t, s, "resources.jobs.foo", "v3")

	close(f.block)
	require.NoError(t, s.Close())

	assert.Equal(t, []string{
		`jobs.foo={"state":{"name":"v1"}}`,
		`jobs.foo={"state":{"name":"v3"}}`,
	}, f.recorded())
}

func TestOperationSinkRecordDuringWriteIsStillWritten(t *testing.T) {
	f := &fakeWriter{block: make(chan struct{}), started: make(chan ResourceKey, 2)}
	s := NewOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	assert.Equal(t, ResourceKey("jobs.foo"), <-f.started)

	// The writer has taken this key off pending and is writing it right now.
	recordState(t, s, "resources.jobs.foo", "v2")

	close(f.block)
	require.NoError(t, s.Close())

	// Two writes, in order: an in-flight request cannot be recalled, so v2 goes up
	// after v1 rather than replacing it. The service ends up with the newest state.
	assert.Equal(t, []string{
		`jobs.foo={"state":{"name":"v1"}}`,
		`jobs.foo={"state":{"name":"v2"}}`,
	}, f.recorded())
	assert.Empty(t, s.pending)
}

func TestOperationSinkRecordWaitsWhenTheQueueIsFull(t *testing.T) {
	// Recording holds the deploy back once every slot is taken. started is buffered for every
	// write: nothing reads it after the first, and a writer blocked sending to it would
	// never drain the queue.
	f := &fakeWriter{block: make(chan struct{}), started: make(chan ResourceKey, operationSinkQueueSize+4)}
	s := NewOperationSink(t.Context(), f)

	// One key is taken off the queue and stuck in the writer; the rest fill it.
	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, ResourceKey("jobs.busy"), <-f.started)
	for i := range operationSinkQueueSize {
		recordState(t, s, "resources.jobs.job"+strconv.Itoa(i), "v1")
	}

	// The next distinct resource has nowhere to go until the writer moves on. Called
	// directly rather than through recordState: its assertions may only run on the
	// test's own goroutine.
	late := envelope("v1")
	blocked := make(chan struct{})
	go func() {
		s.RecordOperation(t.Context(), "resources.jobs.late", false, "id-1", late)
		close(blocked)
	}()

	select {
	case <-blocked:
		t.Fatal("recording did not wait for a full queue, so the deploy can outrun the service")
	case <-time.After(50 * time.Millisecond):
	}

	close(f.block)
	<-blocked
	require.NoError(t, s.Close())

	assert.Len(t, f.recorded(), operationSinkQueueSize+2)
}

func TestOperationSinkReturnsWriteError(t *testing.T) {
	writeErr := errors.New("boom")
	f := &fakeWriter{err: writeErr}
	s := NewOperationSink(t.Context(), f)

	assert.NoError(t, s.FirstErr())

	recordState(t, s, "resources.jobs.foo", "v1")

	err := s.Close()
	require.Error(t, err)
	assert.ErrorIs(t, err, writeErr)
	assert.ErrorContains(t, err, "resources.jobs.foo")
	// Reported after the fact too, so apply can check between resources.
	assert.Error(t, s.FirstErr())
}

func TestOperationSinkKeepsRecordingAfterWriteError(t *testing.T) {
	// One failed write must not drop the records for everything behind it, so DMS
	// ends up as close to reality as it can get.
	f := &fakeWriter{err: errors.New("boom")}
	s := NewOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	recordState(t, s, "resources.jobs.bar", "v1")

	require.Error(t, s.Close())
	assert.Len(t, f.recorded(), 2)
}

func TestOperationSinkFailsOnOversizedState(t *testing.T) {
	// The service will not take a state this large, so the resource cannot be recorded.
	// Failing here says so, where reporting nothing would leave DMS without the resource
	// and the next plan would create it again.
	f := &fakeWriter{}
	s := NewOperationSink(t.Context(), f)

	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", envelope(strings.Repeat("x", maxStateSize)))

	err := s.Close()
	require.Error(t, err)
	assert.ErrorContains(t, err, "exceeds the 65536 byte limit")
	assert.Empty(t, f.recorded())
}

func TestOperationSinkCloseIsIdempotent(t *testing.T) {
	f := &fakeWriter{}
	s := NewOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")

	require.NoError(t, s.Close())
	require.NoError(t, s.Close())
	assert.Len(t, f.recorded(), 1)
}
