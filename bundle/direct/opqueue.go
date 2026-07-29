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

// operationQueue hands recorded operations to background workers, so an apply
// worker does not wait for the CreateOperation round trip before deploying the
// next resource.
//
// Two rules shape the design:
//
//   - Uploads for one resource never overlap. DMS stores one state per resource
//     key, so concurrent uploads could land out of order and leave stale state.
//   - Only the newest operation for a resource matters. Each operation carries the
//     resource's full state, not a delta, so a newer one entirely supersedes an
//     older one. When both are still waiting, the older is dropped ("coalesced")
//     and one upload records the result.
//
// Uploads are not fire-and-forget: close returns the first failure and fails the
// deploy. A dropped operation would leave DMS with an incomplete resource set,
// and since DMS then becomes the source of truth (see dstate.readDMSState), the
// next deploy would recreate resources that already exist.
type operationQueue struct {
	uploader operationUploader

	// queue carries resource keys, not operations. A worker looks the operation up
	// when it picks the key up, so recording again before then just overwrites the
	// entry in pending - that is what makes coalescing work.
	queue chan string
	wg    sync.WaitGroup

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest operation per resource key that no worker has taken
	// yet. Empty for a key means everything recorded for it has been uploaded.
	pending map[string]recordedOperation

	// queuedOrUploading marks keys that are already in the queue channel or being
	// uploaded right now. Such a key must not be queued again, or two workers could
	// upload the same resource at once; recording writes to pending instead, and
	// the worker handling the key picks it up when its current upload finishes.
	//
	// No single worker "owns" a key for the whole time it is marked: a key can be
	// handled by one worker, released, and later picked up by another. The mark only
	// means "some worker will get to this", which is all record needs to know.
	queuedOrUploading map[string]bool

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
		uploader:          uploader,
		queue:             make(chan string, operationQueueSize),
		pending:           make(map[string]recordedOperation),
		queuedOrUploading: make(map[string]bool),
	}

	q.wg.Add(operationUploadWorkers)
	for range operationUploadWorkers {
		go q.work(ctx)
	}

	return q
}

// record serializes an operation and hands it to the upload workers. It makes no
// API call, so upload failures surface from close; an error here only means the
// applied resource could not be turned into a payload.
//
// Recording a resource that is still waiting replaces the waiting operation
// outright, since the newer one carries the resource's full state.
func (q *operationQueue) record(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	if q == nil {
		return nil
	}

	op, err := newRecordedOperation(action, resourceID, state, dependsOn)
	if err != nil {
		return err
	}

	q.mu.Lock()
	_, replaced := q.pending[resourceKey]
	q.pending[resourceKey] = op
	alreadyHandled := q.queuedOrUploading[resourceKey]
	q.queuedOrUploading[resourceKey] = true
	q.mu.Unlock()

	if replaced {
		log.Debugf(ctx, "Coalescing queued deployment operation for %s", resourceKey)
	}

	// A worker is already going to handle this key, and it re-reads pending before
	// finishing, so it will see the operation written above. Queueing the key again
	// would let a second worker upload the same resource concurrently.
	if alreadyHandled {
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
		// Keep uploading this key until nothing new was recorded for it, rather than
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

// take claims the operation waiting for resourceKey. It reports false and clears
// the queuedOrUploading mark when nothing is waiting, which is what lets the next
// record queue the key again.
//
// Clearing the mark and observing pending empty happen under one lock, so record
// can never skip queueing a key that no worker is going to look at again.
func (q *operationQueue) take(resourceKey string) (recordedOperation, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	op, ok := q.pending[resourceKey]
	if !ok {
		delete(q.queuedOrUploading, resourceKey)
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
