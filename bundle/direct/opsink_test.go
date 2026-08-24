package direct

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWriter records the writes it receives and optionally blocks until block is
// closed, so a test can hold the writer and observe what coalesces behind it.
//
// Keys arrive in the DMS form, which is what the sink puts on the wire.
type fakeWriter struct {
	block   chan struct{}
	started chan dms.ResourceKey
	err     error

	mu     sync.Mutex
	writes []string
}

func (f *fakeWriter) Write(ctx context.Context, key dms.ResourceKey, update dms.OperationUpdate) error {
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

// envelope builds the serialized RecordedState the state DB hands the sink.
func envelope(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(dstate.RecordedState{State: json.RawMessage(`{"name":"` + name + `"}`)})
	require.NoError(t, err)
	return raw
}

func recordState(t *testing.T, s *operationSink, resourceKey, name string) {
	t.Helper()
	s.RecordOperation(t.Context(), resourceKey, false, "id-1", envelope(t, name))
}

func TestOperationSinkKeepsWritingAfterGoingIdle(t *testing.T) {
	// The writer parks on an empty queue instead of returning. Apply can spend long stretches
	// inside resource CRUD with nothing to record, and a writer that exited while idle would
	// silently drop everything recorded after it.
	f := &fakeWriter{}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	require.Eventually(t, func() bool { return len(f.recorded()) == 1 }, 5*time.Second, time.Millisecond)

	// The queue is drained and the writer idle; what is recorded now still has to go.
	recordState(t, s, "resources.jobs.bar", "v1")
	require.NoError(t, s.close())

	assert.Len(t, f.recorded(), 2)
}

func TestOperationSinkCoalescesWritesBehindAWrite(t *testing.T) {
	// Hold the writer on the first write so the two behind it pile up. They carry
	// the resource's full state, so only the newest needs to go: the resource costs
	// two requests rather than three.
	f := &fakeWriter{block: make(chan struct{}), started: make(chan dms.ResourceKey, 2)}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	assert.Equal(t, dms.ResourceKey("jobs.foo"), <-f.started)

	recordState(t, s, "resources.jobs.foo", "v2")
	recordState(t, s, "resources.jobs.foo", "v3")

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`jobs.foo={"state":{"name":"v1"}}`,
		`jobs.foo={"state":{"name":"v3"}}`,
	}, f.recorded())
}

func TestOperationSinkRecordDuringWriteIsStillWritten(t *testing.T) {
	f := &fakeWriter{block: make(chan struct{}), started: make(chan dms.ResourceKey, 2)}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	assert.Equal(t, dms.ResourceKey("jobs.foo"), <-f.started)

	// The writer has taken this key off pending and is writing it right now.
	recordState(t, s, "resources.jobs.foo", "v2")

	close(f.block)
	require.NoError(t, s.close())

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
	f := &fakeWriter{block: make(chan struct{}), started: make(chan dms.ResourceKey, operationSinkQueueSize+4)}
	s := newOperationSink(t.Context(), f)

	// One key is taken off the queue and stuck in the writer; the rest fill it.
	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, dms.ResourceKey("jobs.busy"), <-f.started)
	for i := range operationSinkQueueSize {
		recordState(t, s, "resources.jobs.job"+strconv.Itoa(i), "v1")
	}

	// The next distinct resource has nowhere to go until the writer moves on. Called
	// directly rather than through recordState: its assertions may only run on the
	// test's own goroutine.
	late := envelope(t, "v1")
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
	require.NoError(t, s.close())

	assert.Len(t, f.recorded(), operationSinkQueueSize+2)
}

func TestOperationSinkReturnsWriteError(t *testing.T) {
	writeErr := errors.New("boom")
	f := &fakeWriter{err: writeErr}
	s := newOperationSink(t.Context(), f)

	assert.NoError(t, s.firstErr())

	recordState(t, s, "resources.jobs.foo", "v1")

	err := s.close()
	require.Error(t, err)
	assert.ErrorIs(t, err, writeErr)
	assert.ErrorContains(t, err, "resources.jobs.foo")
	// Reported after the fact too, so apply can check between resources.
	assert.Error(t, s.firstErr())
}

func TestOperationSinkKeepsRecordingAfterWriteError(t *testing.T) {
	// One failed write must not drop the records for everything behind it, so DMS
	// ends up as close to reality as it can get.
	f := &fakeWriter{err: errors.New("boom")}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")
	recordState(t, s, "resources.jobs.bar", "v1")

	require.Error(t, s.close())
	assert.Len(t, f.recorded(), 2)
}

func TestOperationSinkFailsOnOversizedState(t *testing.T) {
	// The service will not take a state this large (the limit lives in libs/dms), so the
	// resource cannot be recorded. Failing here says so, where reporting nothing would leave
	// DMS without the resource and the next plan would create it again.
	f := &fakeWriter{}
	s := newOperationSink(t.Context(), f)

	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", envelope(t, strings.Repeat("x", 64*1024)))

	err := s.close()
	require.Error(t, err)
	assert.ErrorContains(t, err, "exceeds the 65536 byte limit")
	assert.Empty(t, f.recorded())
}

func TestOperationSinkCloseIsIdempotent(t *testing.T) {
	f := &fakeWriter{}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.foo", "v1")

	require.NoError(t, s.close())
	require.NoError(t, s.close())
	assert.Len(t, f.recorded(), 1)
}

func TestNilOperationSinkIsNoOp(t *testing.T) {
	// Recording off: apply holds no sink at all, so every method has to work on nil.
	var s *operationSink
	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", nil)
	s.recordFailure("resources.jobs.foo", "id-1", errors.New("boom"))
	assert.NoError(t, s.firstErr())
	assert.NoError(t, s.close())
}
