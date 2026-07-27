package direct

import (
	"context"
	"fmt"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
)

const (
	// operationQueueSize bounds how many recorded operations wait for upload.
	// Apply deploys at most defaultParallelism resources at a time, so a queue
	// this deep means an apply worker practically never blocks on a free slot.
	operationQueueSize = 10

	// operationUploadWorkers is how many uploads run at a time. It is below
	// operationQueueSize so a burst of operations is absorbed by the queue rather
	// than by one request per resource.
	operationUploadWorkers = 4
)

// operationQueue uploads recorded operations from background workers, so an apply
// worker does not wait for the CreateOperation round trip before moving on to the
// next resource.
//
// It guarantees at most one upload in flight per resource key: within a key the
// worker that owns it uploads sequentially, so the last operation recorded for a
// resource is also the last one the service sees.
//
// Uploads are not fire-and-forget: close drains the queue and returns the first
// failure, which fails the deploy. That matters because a successfully completed
// version makes DMS the source of truth for resource state (see
// dstate.overlayDMSState); silently dropping an operation would leave DMS with an
// incomplete resource set, and the next deploy would plan to create resources
// that already exist.
type operationQueue struct {
	uploader operationUploader

	// queue carries resource keys, not the operations themselves: a worker looks
	// the operation up in pending when it picks the key up, which is what lets
	// record collapse repeated writes to the same resource.
	queue chan string
	wg    sync.WaitGroup

	// mu guards the fields below.
	mu sync.Mutex

	// pending is the latest operation recorded per resource key that no worker has
	// picked up yet.
	pending map[string]recordedOperation

	// inflight holds the resource keys a worker currently owns. A key that is
	// in flight is not queued again: the owning worker re-checks pending after its
	// upload and picks up anything recorded in the meantime.
	inflight map[string]bool

	err    error
	closed bool
}

// newOperationQueue starts the upload workers. It returns nil when uploader is
// nil (recording disabled), and every method is a no-op on a nil queue so callers
// do not have to branch.
//
// ctx is used for the uploads, so it must stay valid until close returns.
func newOperationQueue(ctx context.Context, uploader operationUploader) *operationQueue {
	if uploader == nil {
		return nil
	}

	q := &operationQueue{
		uploader: uploader,
		queue:    make(chan string, operationQueueSize),
		pending:  make(map[string]recordedOperation),
		inflight: make(map[string]bool),
	}

	q.wg.Add(operationUploadWorkers)
	for range operationUploadWorkers {
		go q.work(ctx)
	}

	return q
}

// record serializes an operation and queues it for upload. It performs no API
// call, so upload failures surface from close rather than here; the error
// returned is only about turning the applied resource into a payload.
//
// When an operation for the same resource is already waiting it is replaced
// instead of queued again: DMS keeps one state per resource key, so the later
// operation supersedes the earlier one and a single upload records both. This is
// best effort - only operations that have not been picked up yet are collapsed.
func (q *operationQueue) record(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any) error {
	if q == nil {
		return nil
	}

	op, err := newRecordedOperation(action, resourceID, state)
	if err != nil {
		return err
	}

	q.mu.Lock()
	_, waiting := q.pending[resourceKey]
	owned := waiting || q.inflight[resourceKey]
	q.pending[resourceKey] = op
	q.mu.Unlock()

	if owned {
		log.Debugf(ctx, "Coalescing queued deployment operation for %s", resourceKey)
		return nil
	}

	q.queue <- resourceKey
	return nil
}

// close drains the queue and returns the first upload error. All callers of
// record must have returned first: record on a closed queue panics. Calling close
// more than once is safe, so callers can defer it and still check the error at a
// specific point.
func (q *operationQueue) close() error {
	if q == nil {
		return nil
	}

	q.mu.Lock()
	closed := q.closed
	q.closed = true
	q.mu.Unlock()

	if !closed {
		close(q.queue)
		q.wg.Wait()
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	return q.err
}

func (q *operationQueue) work(ctx context.Context) {
	defer q.wg.Done()

	for resourceKey := range q.queue {
		// Keep uploading this key until nothing new was recorded for it, instead of
		// putting it back on the queue: a worker sending to the channel it consumes
		// from can deadlock once the queue is full.
		for {
			op, ok := q.take(resourceKey)
			if !ok {
				break
			}

			if err := q.uploader.upload(ctx, resourceKey, op); err != nil {
				q.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
			}
		}
	}
}

// take claims the operation waiting for resourceKey, marking the key in flight so
// record does not queue it a second time. It reports false, and releases the key,
// when nothing is waiting.
func (q *operationQueue) take(resourceKey string) (recordedOperation, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	op, ok := q.pending[resourceKey]
	if !ok {
		delete(q.inflight, resourceKey)
		return recordedOperation{}, false
	}

	delete(q.pending, resourceKey)
	q.inflight[resourceKey] = true
	return op, true
}

// setErr keeps the first upload error; later ones are dropped because one failure
// is enough to fail the deploy.
func (q *operationQueue) setErr(err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.err == nil {
		q.err = err
	}
}
