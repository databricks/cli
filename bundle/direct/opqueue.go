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
// worker does not wait for the CreateOperation round trip.
//
// At most one upload is in flight per resource key, so the last operation
// recorded for a resource is also the last one the service sees.
//
// Uploads are not fire-and-forget: close returns the first failure and fails the
// deploy. A dropped operation would leave DMS with an incomplete resource set,
// and since DMS then becomes the source of truth (see dstate.overlayDMSState),
// the next deploy would recreate resources that already exist.
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

	// owned holds the resource keys that are already queued or being uploaded.
	// Such a key is never queued a second time; recording writes to pending
	// instead, which the owning worker re-checks after each upload. This is what
	// keeps uploads for one resource sequential.
	owned map[string]bool

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
		owned:    make(map[string]bool),
	}

	q.wg.Add(operationUploadWorkers)
	for range operationUploadWorkers {
		go q.work(ctx)
	}

	return q
}

// record serializes an operation and queues it for upload. It makes no API call,
// so upload failures surface from close; an error here only means the applied
// resource could not be turned into a payload.
//
// An operation for a resource that is still waiting replaces it rather than
// queueing again: DMS keeps one state per key, so one upload records both. The
// merge keeps a queued create's action (see mergeAction). Best effort — only
// operations no worker has picked up yet are collapsed.
func (q *operationQueue) record(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	if q == nil {
		return nil
	}

	op, err := newRecordedOperation(action, resourceID, state, dependsOn)
	if err != nil {
		return err
	}

	q.mu.Lock()
	queued, waiting := q.pending[resourceKey]
	if waiting {
		op.action = mergeAction(queued.action, op.action)
	}
	q.pending[resourceKey] = op
	owned := q.owned[resourceKey]
	q.owned[resourceKey] = true
	q.mu.Unlock()

	if waiting {
		log.Debugf(ctx, "Coalescing queued deployment operation for %s", resourceKey)
	}

	// Someone already owns this key, so pending is enough: a worker will pick the
	// operation up. Queueing again would upload the resource twice in parallel.
	if owned {
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

// take claims the operation waiting for resourceKey. It reports false and gives
// up ownership when nothing is waiting, which is what lets the next record
// queue the key again.
func (q *operationQueue) take(resourceKey string) (recordedOperation, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	op, ok := q.pending[resourceKey]
	if !ok {
		delete(q.owned, resourceKey)
		return recordedOperation{}, false
	}

	delete(q.pending, resourceKey)
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
