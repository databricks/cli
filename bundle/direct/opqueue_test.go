package direct

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUploader records the uploads it receives and optionally blocks until
// release is closed, so a test can hold operations in the queue and observe
// coalescing.
type fakeUploader struct {
	block   chan struct{}
	started chan string
	// done receives the resource key after the upload returns, for tests that need
	// an upload to have completed rather than merely started.
	done chan string
	err  error

	mu          sync.Mutex
	uploads     []string
	actions     map[string]bundledeployments.OperationActionType
	resourceIDs map[string]string
}

func (f *fakeUploader) upload(ctx context.Context, resourceKey string, op recordedOperation) error {
	if f.started != nil {
		f.started <- resourceKey
	}
	if f.block != nil {
		<-f.block
	}

	f.mu.Lock()
	f.uploads = append(f.uploads, resourceKey+"="+string(op.state))
	if f.actions == nil {
		f.actions = map[string]bundledeployments.OperationActionType{}
		f.resourceIDs = map[string]string{}
	}
	f.actions[resourceKey] = op.action
	f.resourceIDs[resourceKey] = op.resourceID
	f.mu.Unlock()

	// Sent outside the lock: a test that stops reading this channel would otherwise
	// hold f.mu and deadlock every other worker.
	if f.done != nil {
		f.done <- resourceKey
	}
	return f.err
}

func (f *fakeUploader) recorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.uploads...)
}

func (f *fakeUploader) actionFor(resourceKey string) bundledeployments.OperationActionType {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.actions[resourceKey]
}

func (f *fakeUploader) resourceIDFor(resourceKey string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resourceIDs[resourceKey]
}

// envelope builds the serialized RecordedState the state DB hands the queue.
func envelope(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(dstate.RecordedState{State: json.RawMessage(`{"name":"` + name + `"}`)})
	require.NoError(t, err)
	return raw
}

func recordState(t *testing.T, q *operationQueue, resourceKey, name string) {
	t.Helper()
	q.RecordOperation(t.Context(), resourceKey, deployplan.Update, "id-1", envelope(t, name))
}

func TestOperationQueueUploadsEachOperation(t *testing.T) {
	f := &fakeUploader{}
	q := newOperationQueue(t.Context(), f)

	for i := range 20 {
		recordState(t, q, "resources.jobs.job"+strconv.Itoa(i), "n")
	}
	require.NoError(t, q.close())

	assert.Len(t, f.recorded(), 20)
}

func TestOperationQueueUploadsEveryWriteForSameResource(t *testing.T) {
	// Hold the first upload so the writes behind it queue up. Each one is its own
	// event, so all three are uploaded, oldest first - a resource can legitimately
	// write state several times in one deploy (see Recreate).
	//
	// started is buffered for all three: every write now uploads, and a worker
	// blocking on an unread send would deadlock the drain below.
	f := &fakeUploader{block: make(chan struct{}), started: make(chan string, 3)}
	q := newOperationQueue(t.Context(), f)

	recordState(t, q, "resources.jobs.foo", "v1")
	// Wait until a worker owns the key, so the operations below are queued behind
	// an in-flight upload rather than racing it.
	assert.Equal(t, "resources.jobs.foo", <-f.started)

	recordState(t, q, "resources.jobs.foo", "v2")
	recordState(t, q, "resources.jobs.foo", "v3")

	close(f.block)
	require.NoError(t, q.close())

	assert.Equal(t, []string{
		`resources.jobs.foo={"state":{"name":"v1"}}`,
		`resources.jobs.foo={"state":{"name":"v2"}}`,
		`resources.jobs.foo={"state":{"name":"v3"}}`,
	}, f.recorded())
}

func TestOperationQueueUploadsQueuedWritesWhileWorkersAreBusy(t *testing.T) {
	// Every worker is parked mid-upload, so the writes below sit in pending rather
	// than being picked up. Both still go out, in order, once a worker frees up.
	//
	// started is buffered for the two foo writes as well: nothing reads it after the
	// loop below, and a worker blocking on the send would deadlock the drain.
	f := &fakeUploader{block: make(chan struct{}), started: make(chan string, operationUploadWorkers+2)}
	q := newOperationQueue(t.Context(), f)

	recordState(t, q, "resources.jobs.hold", "v1")
	assert.Equal(t, "resources.jobs.hold", <-f.started)

	// Occupy the remaining workers so nothing drains the key under test.
	for i := range operationUploadWorkers - 1 {
		recordState(t, q, "resources.jobs.hold"+strconv.Itoa(i), "v1")
		assert.Equal(t, "resources.jobs.hold"+strconv.Itoa(i), <-f.started)
	}

	// A resource whose ID is only known after it was created: the first write has no
	// ID, the second fills it in.
	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "", envelope(t, "created"))
	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "id-1", envelope(t, "updated"))

	close(f.block)
	require.NoError(t, q.close())

	var uploadsForFoo []string
	for _, u := range f.recorded() {
		if strings.HasPrefix(u, "resources.jobs.foo=") {
			uploadsForFoo = append(uploadsForFoo, u)
		}
	}
	assert.Equal(t, []string{
		`resources.jobs.foo={"state":{"name":"created"}}`,
		`resources.jobs.foo={"state":{"name":"updated"}}`,
	}, uploadsForFoo)

	// The ID recorded last is the one the create learned.
	assert.Equal(t, "id-1", f.resourceIDFor("resources.jobs.foo"))
	assert.Equal(t,
		bundledeployments.OperationActionTypeOperationActionTypeCreate,
		f.actionFor("resources.jobs.foo"))
}

func TestOperationQueueRecordDuringUploadIsStillUploaded(t *testing.T) {
	// Record while the key's own upload is in flight: the key is off the queue but
	// still marked, so record does not queue it again. The worker that holds the key
	// has to come back for it, or the operation would be silently dropped.
	f := &fakeUploader{block: make(chan struct{}), started: make(chan string, 1)}
	q := newOperationQueue(t.Context(), f)

	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "", envelope(t, "v1"))
	assert.Equal(t, "resources.jobs.foo", <-f.started)

	// The worker has taken the key off the queue and is uploading v1 right now.
	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "id-1", envelope(t, "v2"))

	close(f.block)
	require.NoError(t, q.close())

	// Two uploads, in order: an in-flight request cannot be recalled, so v2 goes up
	// after v1 rather than replacing it. The service ends up with the newest state.
	assert.Equal(t, []string{
		`resources.jobs.foo={"state":{"name":"v1"}}`,
		`resources.jobs.foo={"state":{"name":"v2"}}`,
	}, f.recorded())
	assert.Equal(t, "id-1", f.resourceIDFor("resources.jobs.foo"))
	assert.Empty(t, q.pending)
	assert.Empty(t, q.queuedOrUploading)
}

func TestOperationQueueReturnsUploadError(t *testing.T) {
	uploadErr := errors.New("boom")
	f := &fakeUploader{err: uploadErr}
	q := newOperationQueue(t.Context(), f)

	recordState(t, q, "resources.jobs.foo", "v1")

	err := q.close()
	require.Error(t, err)
	assert.ErrorIs(t, err, uploadErr)
	assert.Contains(t, err.Error(), "resources.jobs.foo")
}

func TestOperationQueueKeepsRecordingAfterUploadError(t *testing.T) {
	// A failed upload must not stop the ones behind it: every applied resource is
	// recorded best effort, so DMS ends up as close to reality as it can get.
	uploadErr := errors.New("boom")
	f := &fakeUploader{err: uploadErr, done: make(chan string, 1)}
	q := newOperationQueue(t.Context(), f)

	// Wait for the failing upload to finish, so the error is stored before the next
	// record rather than racing it.
	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "id-1", envelope(t, "v1"))
	assert.Equal(t, "resources.jobs.foo", <-f.done)

	// The next resource is still accepted, even though the first upload failed.
	q.RecordOperation(t.Context(), "resources.jobs.bar", deployplan.Create, "id-2", envelope(t, "v1"))

	// Both were attempted, and close still reports the failure so the deploy fails.
	require.ErrorIs(t, q.close(), uploadErr)
	assert.ElementsMatch(t, []string{
		`resources.jobs.foo={"state":{"name":"v1"}}`,
		`resources.jobs.bar={"state":{"name":"v1"}}`,
	}, f.recorded())
	assert.Empty(t, q.pending)
	assert.Empty(t, q.queuedOrUploading)
}

func TestOperationQueueDrainsQueuedOperationsAfterUploadError(t *testing.T) {
	// A failure does not discard work already recorded: the records DMS ends up with
	// have to match the resources that were applied.
	uploadErr := errors.New("boom")
	f := &fakeUploader{err: uploadErr, block: make(chan struct{}), started: make(chan string, 1)}
	q := newOperationQueue(t.Context(), f)

	// Every worker is parked mid-upload, so these stay queued.
	for i := range operationUploadWorkers {
		q.RecordOperation(t.Context(), "resources.jobs.hold"+strconv.Itoa(i), deployplan.Create, "id-1", envelope(t, "v1"))
		assert.Equal(t, "resources.jobs.hold"+strconv.Itoa(i), <-f.started)
	}
	q.RecordOperation(t.Context(), "resources.jobs.queued", deployplan.Create, "id-2", envelope(t, "v1"))

	close(f.block)
	require.ErrorIs(t, q.close(), uploadErr)

	// The queued operation was uploaded rather than dropped on the way out.
	assert.Contains(t, f.recorded(), `resources.jobs.queued={"state":{"name":"v1"}}`)
	assert.Len(t, f.recorded(), operationUploadWorkers+1)
}

func TestOperationQueueRecordDropsUnsupportedAction(t *testing.T) {
	f := &fakeUploader{}
	q := newOperationQueue(t.Context(), f)

	// The state write already succeeded, so an operation that cannot be described is
	// dropped with a warning rather than failing the deploy.
	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Skip, "id-1", nil)

	require.NoError(t, q.close())
	assert.Empty(t, f.recorded())
}

func TestOperationQueueRecordDropsOversizedState(t *testing.T) {
	f := &fakeUploader{}
	q := newOperationQueue(t.Context(), f)

	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "id-1", envelope(t, strings.Repeat("x", maxOperationStateSize)))

	require.NoError(t, q.close())
	assert.Empty(t, f.recorded())
}

func TestOperationQueueCloseIsIdempotent(t *testing.T) {
	f := &fakeUploader{err: errors.New("boom")}
	q := newOperationQueue(t.Context(), f)

	recordState(t, q, "resources.jobs.foo", "v1")

	require.Error(t, q.close())
	// A second close reports the same error instead of panicking on the already
	// closed channel, so callers can both defer close and check it explicitly.
	require.Error(t, q.close())
}

// serialUploader fails if two uploads for the same resource key ever overlap.
type serialUploader struct {
	mu     sync.Mutex
	live   map[string]bool
	last   map[string]string
	uneven bool
}

func (s *serialUploader) upload(ctx context.Context, resourceKey string, op recordedOperation) error {
	s.mu.Lock()
	if s.live[resourceKey] {
		s.uneven = true
	}
	s.live[resourceKey] = true
	s.mu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.live[resourceKey] = false
	s.last[resourceKey] = string(op.state)
	return nil
}

func TestOperationQueueUploadsOneResourceAtATime(t *testing.T) {
	// Two workers must never upload the same resource at the same time. DMS stores
	// one state per resource, so concurrent uploads can finish out of order and
	// leave the older state as the final one.
	//
	// Lots of goroutines record a small set of keys, so the same key is recorded
	// repeatedly while its earlier upload may still be running. serialUploader flags
	// any overlap it sees.
	//
	// Whether a bug shows up depends on how the scheduler interleaves things, so one
	// pass proves little - repeat it to get many chances at a bad ordering.
	const (
		iterations     = 200
		workers        = 10
		perWorker      = 5
		distinctKeyMod = 12
	)

	for range iterations {
		ctx := t.Context()
		u := &serialUploader{live: map[string]bool{}, last: map[string]string{}}
		q := newOperationQueue(ctx, u)

		// The envelopes are built up front: json.Marshal is fine on many goroutines,
		// but the helper takes *testing.T, which is not.
		states := make([]json.RawMessage, workers)
		for w := range workers {
			states[w] = envelope(t, strconv.Itoa(w))
		}

		var wg sync.WaitGroup
		for w := range workers {
			wg.Go(func() {
				for i := range perWorker {
					key := "resources.jobs.job" + strconv.Itoa((w*perWorker+i)%distinctKeyMod)
					q.RecordOperation(ctx, key, deployplan.Update, "id-1", states[w])
				}
			})
		}
		wg.Wait()
		require.NoError(t, q.close())

		require.False(t, u.uneven, "two uploads overlapped for the same resource key")
		// Every distinct key was recorded, and close drained all of them.
		require.Len(t, u.last, distinctKeyMod)
		require.Empty(t, q.pending)
		require.Empty(t, q.queuedOrUploading)
	}
}

func TestNilOperationQueueIsNoOp(t *testing.T) {
	// Recording is disabled: newOperationQueue returns nil and every method is a
	// no-op, so Apply does not have to branch.
	q := newOperationQueue(t.Context(), nil)
	require.Nil(t, q)
	q.RecordOperation(t.Context(), "resources.jobs.foo", deployplan.Create, "id-1", nil)
	require.NoError(t, q.close())
}
