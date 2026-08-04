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

// record serializes an operation and hands it to the upload workers. The upload
// itself happens on a worker, so an error returned here is either a failure to
// turn the applied resource into a payload, or an earlier upload's error
// resurfaced (see below).
//
// Recording a resource that is still waiting replaces the waiting operation
// outright, since the newer one carries the resource's full state.
func (q *operationQueue) record(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	if q == nil {
		return nil
	}

	// Report an earlier upload failure to the apply worker that is about to record
	// the next resource, so the deploy stops instead of running to completion and
	// only failing at close. That matters because a successfully completed version
	// makes DMS the source of truth for resource state (see dstate.readDMSState):
	// deploying everything while its records are missing leaves resources the next
	// deploy would create a second time.
	//
	// This refuses new work only. Operations already recorded still upload - close
	// drains them - so the records DMS does end up with match the resources that
	// were actually applied. Resources already mid-apply also finish, so the deploy
	// stops shortly after the first failure rather than exactly at it.
	if err := q.firstErr(); err != nil {
		return err
	}

	op, err := newRecordedOperation(action, resourceID, state, dependsOn)
	if err != nil {
		return err
	}

	return q.enqueue(ctx, resourceKey, op)
}

// recordFailure queues a FAILED operation carrying the error that a resource hit
// while applying, so DMS records why the deployment version failed. Unlike
// record it does not short-circuit on a prior upload error: the failure is the
// reason the deploy is stopping and must still be recorded.
func (q *operationQueue) recordFailure(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID, errorMessage string) error {
	if q == nil {
		return nil
	}

	op, err := newFailedOperation(action, resourceID, errorMessage)
	if err != nil {
		return err
	}

	return q.enqueue(ctx, resourceKey, op)
}

// enqueue stores op for resourceKey and hands the key to a worker, coalescing
// with any operation already queued for the same resource.
func (q *operationQueue) enqueue(ctx context.Context, resourceKey string, op recordedOperation) error {
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
//
// Unlike the other methods this one takes no lock. It runs on one goroutine after
// every apply worker has returned, so nothing else touches the queue by then, and
// the wg.Wait below orders the workers' writes to err before it is read.
func (q *operationQueue) close() error {
	if q == nil {
		return nil
	}

	if !q.closed {
		q.closed = true
		close(q.queue)
		q.wg.Wait()
	}

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

	// The key stays in queuedOrUploading: the worker keeps coming back here until
	// nothing is pending for it, so anything recorded while this operation uploads
	// is still picked up. The mark is only cleared above, once there is nothing
	// left - which is also what stops a second worker from taking the key and
	// uploading the same resource concurrently.
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

// firstErr returns the first upload error, or nil if every upload so far
// succeeded. A nil queue (recording disabled) never errors.
func (q *operationQueue) firstErr() error {
	if q == nil {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	return q.err
}
