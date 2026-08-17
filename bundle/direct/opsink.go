package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
)

// operationSinkQueueSize is how many resources may have an unrecorded state write
// before the next one waits. Recording is deliberately not free: a deploy that ran far
// ahead of the service would leave a long tail of resources applied but unrecorded, and
// DMS is the source of truth for the next plan. At one waiting operation per resource
// this bounds the lag to roughly the apply parallelism.
const operationSinkQueueSize = 10

// operationSink uploads recorded operations from one background goroutine, so a deploy
// does not wait on every CreateOperation round trip and the service sees one request at
// a time.
//
// The queue carries resource keys and pending holds the operation for each, which is
// what makes coalescing fall out: a second write for a resource replaces the first in
// the map, and the key already in the queue picks up whichever operation is there when
// the uploader reaches it.
//
// close reports the first upload failure, which fails the deploy: DMS becomes the
// source of truth (see dstate.readDMSState), so a missing record would make the next
// deploy create a resource that already exists.
type operationSink struct {
	uploader operationUploader

	// queue carries the resource keys that have an operation waiting. A key is sent
	// only when nothing was waiting for it, so a resource never occupies more than one
	// slot; once the queue is full, recording waits for the uploader, which is what
	// holds the deploy back. Callers must therefore record outside the state DB lock -
	// see dstate.SaveState.
	queue chan string

	// done is closed once the uploader has drained the queue and returned.
	done chan struct{}

	// stopQueue closes the queue. Wrapped so close can be deferred and still checked
	// at a specific point.
	stopQueue func()

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the operation waiting per resource key. A key is absent once the
	// uploader has taken its operation.
	pending map[string]recordedOperation

	err error
}

// newOperationSink starts the uploader, returning nil when uploader is nil (recording
// off; every method is a no-op on a nil sink). ctx must outlive close.
func newOperationSink(ctx context.Context, uploader operationUploader) *operationSink {
	if uploader == nil {
		return nil
	}

	s := &operationSink{
		uploader: uploader,
		queue:    make(chan string, operationSinkQueueSize),
		done:     make(chan struct{}),
		pending:  make(map[string]recordedOperation),
	}
	s.stopQueue = sync.OnceFunc(func() { close(s.queue) })

	go s.run(ctx)
	return s
}

// RecordOperation implements dstate.OperationSink: every state write becomes an
// operation, so DMS mirrors the WAL. state is already the serialized envelope, and
// nil for a delete.
//
// An earlier upload failure does not stop this: every write is still recorded, best
// effort, so DMS ends up as close to reality as it can get.
func (s *operationSink) RecordOperation(ctx context.Context, resourceKey string, info dstate.OperationInfo, resourceID string, state json.RawMessage) {
	if s == nil {
		return
	}

	op, err := newStateOperation(info, resourceID, state)
	if err != nil {
		s.setErr(fmt.Errorf("recording operation for %s: %w", resourceKey, err))
		return
	}

	s.record(resourceKey, op)
}

// recordFailure records that applying a resource failed, so the deployment history
// explains the failure instead of omitting the resource.
func (s *operationSink) recordFailure(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, priorState json.RawMessage, cause error) {
	if s == nil {
		return
	}

	op, err := newFailedOperation(action, resourceID, priorState, cause)
	if err != nil {
		s.setErr(fmt.Errorf("recording failure for %s: %w", resourceKey, err))
		return
	}

	s.record(resourceKey, op)
}

// record makes op the operation waiting for resourceKey, waiting for the uploader when
// the queue is full. Recording on a closed sink panics; every caller must have returned
// before close.
func (s *operationSink) record(resourceKey string, op recordedOperation) {
	s.mu.Lock()
	waiting, queued := s.pending[resourceKey]
	if queued {
		op = coalesce(waiting, op)
	}
	s.pending[resourceKey] = op
	s.mu.Unlock()

	// Already queued: the uploader reads the map when it reaches the key, so it picks
	// up what was just stored without a second token - and without waiting, since a
	// resource that is already represented in the queue is not running ahead.
	if !queued {
		s.queue <- resourceKey
	}
}

// coalesce folds a write that never got uploaded into the one replacing it. The newer
// write describes the resource as it now stands, so its fields win.
//
// A failure is the exception: it says why a resource stopped, not what it looks like,
// so it keeps the state of the write it supersedes. Its own state comes from the
// pre-deploy record (see newFailedOperation), which is either nothing - dropping the
// resource from the deployment - or a state the write it replaces has already moved
// past. This is the same rule the wire applies once the write is uploaded, where a
// failure updates only status and error_message; see failureFields.
func coalesce(older, newer recordedOperation) recordedOperation {
	if newer.isFailure() && older.state != nil {
		newer.state = older.state
		newer.resourceID = older.resourceID
	}
	return newer
}

// take claims the operation waiting for resourceKey.
func (s *operationSink) take(resourceKey string) (recordedOperation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.pending[resourceKey]
	delete(s.pending, resourceKey)
	return op, ok
}

func (s *operationSink) run(ctx context.Context) {
	defer close(s.done)

	for resourceKey := range s.queue {
		op, ok := s.take(resourceKey)
		if !ok {
			// A key is queued only when nothing is waiting for it, so this cannot
			// happen; the check is here so a stray token could never upload a
			// zero-valued operation.
			continue
		}

		// Keep going after a failure, so one bad upload does not drop the records for
		// every resource behind it.
		if err := s.uploader.upload(ctx, resourceKey, op); err != nil {
			s.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
		}
	}
}

// close drains the recorded operations and returns the first upload error. Every
// record caller must have returned first; calling close twice is safe, so it can be
// deferred and still checked at a specific point.
func (s *operationSink) close() error {
	if s == nil {
		return nil
	}

	s.stopQueue()
	<-s.done
	return s.firstErr()
}

// setErr keeps the first recording error; later ones are dropped because one failure
// is enough to fail the deploy.
func (s *operationSink) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err == nil {
		s.err = err
	}
}

// firstErr returns the first recording error, or nil if everything so far was
// recorded. A nil sink (recording disabled) never errors.
func (s *operationSink) firstErr() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}
