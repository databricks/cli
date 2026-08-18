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
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
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

	mu            sync.Mutex
	writes        []string
	resourceIDs   map[dms.ResourceKey]string
	statuses      map[dms.ResourceKey]bundledeployments.OperationStatus
	errorMessages map[dms.ResourceKey]string
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
	if f.resourceIDs == nil {
		f.resourceIDs = map[dms.ResourceKey]string{}
		f.statuses = map[dms.ResourceKey]bundledeployments.OperationStatus{}
		f.errorMessages = map[dms.ResourceKey]string{}
	}
	f.resourceIDs[key] = update.ResourceID
	f.statuses[key] = update.Status
	f.errorMessages[key] = update.ErrorMessage
	f.mu.Unlock()

	return f.err
}

func (f *fakeWriter) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.writes...)
}

func (f *fakeWriter) resourceIDFor(key dms.ResourceKey) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resourceIDs[key]
}

func (f *fakeWriter) statusFor(key dms.ResourceKey) bundledeployments.OperationStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.statuses[key]
}

func (f *fakeWriter) errorMessageFor(key dms.ResourceKey) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.errorMessages[key]
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

func TestOperationSinkWritesEachOperation(t *testing.T) {
	f := &fakeWriter{}
	s := newOperationSink(t.Context(), f)

	for i := range 20 {
		recordState(t, s, "resources.jobs.job"+strconv.Itoa(i), "n")
	}
	require.NoError(t, s.close())

	assert.Len(t, f.recorded(), 20)
}

func TestOperationSinkKeepsWritingAfterGoingIdle(t *testing.T) {
	// The writer parks on an empty queue instead of returning. Apply spends most of a
	// deploy inside resource CRUD, so the queue is empty far more often than not, and a
	// writer that exited while idle would silently drop everything recorded after it.
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

func TestOperationSinkCoalescedFailureKeepsTheStateItReplaces(t *testing.T) {
	// The create writes state and then fails before the write goes out. The failure must not
	// replace that state with its own emptiness, which would drop the resource from the
	// deployment.
	f := &fakeWriter{block: make(chan struct{}), started: make(chan dms.ResourceKey, 2)}
	s := newOperationSink(t.Context(), f)

	// Occupy the writer with an unrelated resource, so the two writes below both
	// land in pending and coalesce.
	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, dms.ResourceKey("jobs.busy"), <-f.started)

	s.RecordOperation(t.Context(), "resources.job_runs.my_run", false, "run-1", envelope(t, "the run"))
	// The id is empty: the resource was created in this deploy, so there is no pre-deploy
	// record to report.
	s.recordFailure("resources.job_runs.my_run", "", errors.New("run did not succeed: FAILED"))

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`jobs.busy={"state":{"name":"v1"}}`,
		`job_runs.my_run={"state":{"name":"the run"}}`,
	}, f.recorded())
	assert.Equal(t, "run-1", f.resourceIDFor("job_runs.my_run"))
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, f.statusFor("job_runs.my_run"))
	assert.Equal(t, "run did not succeed: FAILED", f.errorMessageFor("job_runs.my_run"))
}

func TestOperationSinkCoalescedFailureAfterADeleteKeepsTheResourceGone(t *testing.T) {
	// The recreate's delete writes no state. When the create then fails, the failure takes
	// that absent state rather than the pre-deploy one, so the resource stays gone.
	f := &fakeWriter{block: make(chan struct{}), started: make(chan dms.ResourceKey, 2)}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, dms.ResourceKey("jobs.busy"), <-f.started)

	s.RecordOperation(t.Context(), "resources.schemas.foo", true, "old-id", nil)
	s.recordFailure("resources.schemas.foo", "old-id", errors.New("Catalog 'other' does not exist"))

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`jobs.busy={"state":{"name":"v1"}}`,
		`schemas.foo=`,
	}, f.recorded())
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, f.statusFor("schemas.foo"))
}

func TestOperationSinkCoalescedFailureDoesNotRevertToPriorState(t *testing.T) {
	// An update that succeeded and then failed waiting carries the pre-deploy id, which the
	// write it supersedes has moved past. Reporting it would record the resource as it was
	// before the deploy, and the next plan would read that back as current.
	f := &fakeWriter{block: make(chan struct{}), started: make(chan dms.ResourceKey, 2)}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, dms.ResourceKey("jobs.busy"), <-f.started)

	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-new", envelope(t, "after the update"))
	s.recordFailure("resources.jobs.foo", "id-old", errors.New("waiting after updating: timed out"))

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`jobs.busy={"state":{"name":"v1"}}`,
		`jobs.foo={"state":{"name":"after the update"}}`,
	}, f.recorded())
	assert.Equal(t, "id-new", f.resourceIDFor("jobs.foo"))
	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, f.statusFor("jobs.foo"))
}

func TestOperationSinkCoalescedDeleteStillClearsState(t *testing.T) {
	// A delete legitimately carries no state, and coalescing must let it through: the
	// resource is gone, and keeping the state it replaces would leave it listed.
	f := &fakeWriter{block: make(chan struct{}), started: make(chan dms.ResourceKey, 2)}
	s := newOperationSink(t.Context(), f)

	recordState(t, s, "resources.jobs.busy", "v1")
	assert.Equal(t, dms.ResourceKey("jobs.busy"), <-f.started)

	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", envelope(t, "before"))
	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", nil)

	close(f.block)
	require.NoError(t, s.close())

	assert.Equal(t, []string{
		`jobs.busy={"state":{"name":"v1"}}`,
		`jobs.foo=`,
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

	recordState(t, s, "resources.jobs.foo", "v1")

	err := s.close()
	require.Error(t, err)
	assert.ErrorIs(t, err, writeErr)
	assert.ErrorContains(t, err, "resources.jobs.foo")
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

func TestOperationSinkFirstErrIsWhatStopsTheDeploy(t *testing.T) {
	f := &fakeWriter{err: errors.New("boom")}
	s := newOperationSink(t.Context(), f)

	assert.NoError(t, s.firstErr())

	recordState(t, s, "resources.jobs.foo", "v1")
	require.Error(t, s.close())

	// Reported after the fact too, so the caller can check once more before it
	// completes the version.
	assert.Error(t, s.firstErr())
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
	var s *operationSink
	s.RecordOperation(t.Context(), "resources.jobs.foo", false, "id-1", nil)
	s.recordFailure("resources.jobs.foo", "id-1", errors.New("boom"))
	assert.NoError(t, s.firstErr())
	assert.NoError(t, s.close())
}

func TestNewOperationSinkNilWriterIsNil(t *testing.T) {
	// Recording off: the sink is nil so the state DB's nil check leaves it unset.
	assert.Nil(t, newOperationSink(t.Context(), nil))
}
