package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
)

const (
	// operationQueueSize bounds how many recorded operations wait for upload. Deep
	// enough that an apply worker practically never blocks on a free slot.
	operationQueueSize = 10

	// operationUploadWorkers is how many uploads run at a time.
	operationUploadWorkers = 8
)

// operationQueue uploads recorded operations from background workers, so a deploy
// never waits on the CreateOperation round trip. Two rules shape it:
//
//   - One resource, one upload at a time. DMS keeps a single state per resource, so
//     overlapping uploads could land out of order and leave the older state.
//   - Newest operation wins. Each write carries the resource's full state, so writes
//     that pile up behind an in-flight upload are merged into one ("coalesced") and
//     the resource costs a single request instead of one per write.
//
// close reports the first upload failure, which fails the deploy: DMS becomes the
// source of truth (see dstate.readDMSState), so a missing record would make the
// next deploy create a resource that already exists.
type operationQueue struct {
	uploader operationUploader

	// queue carries resource keys, not operations: a worker looks the operation up
	// when it picks the key up, which is what makes coalescing work.
	queue chan string
	wg    sync.WaitGroup

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the one operation waiting per resource key: a resource can write
	// state more than once in a deploy (a recreate drops the entry, then saves the new
	// resource), and a write that arrives before the previous one is uploaded replaces
	// it. No key means nothing is waiting.
	pending map[string]recordedOperation

	// queuedOrUploading means "some worker will get to this key". Recording such a
	// key writes to pending only, so two workers never upload one resource at once.
	queuedOrUploading map[string]bool

	err    error
	closed bool
}

// newOperationQueue starts the upload workers, returning nil when uploader is nil
// (recording off; every method is a no-op on a nil queue). ctx must outlive close.
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

// RecordOperation implements dstate.OperationSink: every state write becomes an
// operation, so DMS mirrors the WAL. state is already the serialized envelope, and
// nil for a delete.
//
// An earlier upload failure does not stop this: every write is still recorded, best
// effort, so DMS ends up as close to reality as it can get.
func (q *operationQueue) RecordOperation(ctx context.Context, resourceKey string, info dstate.OperationInfo, resourceID string, state json.RawMessage) {
	if q == nil {
		return
	}

	op, err := newStateOperation(info, resourceID, state)
	if err != nil {
		// The deploy already persisted this write locally, so failing it here would
		// report an error about history for a resource that deployed fine.
		log.Warnf(ctx, "Not recording operation for %s: %s", resourceKey, err)
		return
	}

	q.enqueue(ctx, resourceKey, op)
}

// recordFailure records that applying a resource failed, so the deployment history
// explains the failure instead of omitting the resource. It returns nothing: the
// deploy is already failing, and a second error would mask the one the user needs.
func (q *operationQueue) recordFailure(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, priorState json.RawMessage, cause error) {
	if q == nil {
		return
	}

	op, err := newFailedOperation(action, resourceID, priorState, cause)
	if err != nil {
		log.Warnf(ctx, "Not recording failure for %s: %s", resourceKey, err)
		return
	}

	q.enqueue(ctx, resourceKey, op)
}

// enqueue makes op the operation waiting for resourceKey, replacing whatever was
// already waiting, and makes sure a worker will pick it up.
func (q *operationQueue) enqueue(ctx context.Context, resourceKey string, op recordedOperation) {
	q.mu.Lock()
	q.pending[resourceKey] = op
	alreadyHandled := q.queuedOrUploading[resourceKey]
	q.queuedOrUploading[resourceKey] = true
	q.mu.Unlock()

	// A worker will re-read pending before it finishes, so it picks up the operation
	// stored above. Queueing again would let a second worker upload the same key.
	if alreadyHandled {
		return
	}

	q.queue <- resourceKey
}

// close drains the queue and returns the first upload error. Every record caller
// must have returned first (record on a closed queue panics); calling close twice
// is safe, so it can be deferred and still checked at a specific point.
//
// It takes no lock: it runs after every apply worker returned, and wg.Wait orders
// the workers' writes to err before it is read here.
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
		// Drain this key here instead of re-queueing it: a worker sending to the
		// channel it consumes from deadlocks once the queue is full.
		for {
			op, ok := q.take(resourceKey)
			if !ok {
				break
			}

			// Keep going after a failure, so one bad upload does not drop the records
			// for every resource behind it.
			if err := q.uploader.upload(ctx, resourceKey, op); err != nil {
				q.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
			}
		}
	}
}

// take claims the operation waiting for resourceKey, reporting false and clearing the
// queuedOrUploading mark when nothing is left, which lets record queue it again. Both
// happen under one lock, so a key can never be left for no worker to pick up.
func (q *operationQueue) take(resourceKey string) (recordedOperation, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	op, ok := q.pending[resourceKey]
	if !ok {
		delete(q.queuedOrUploading, resourceKey)
		return recordedOperation{}, false
	}

	// The mark stays until the branch above clears it, so anything recorded during
	// this upload is still picked up and no second worker takes the key meanwhile.
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
