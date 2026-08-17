package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
)

// operationSinkQueueSize is how many resources may be waiting to be recorded before the
// next write has to wait. Without a cap a deploy could finish far ahead of the service,
// and DMS is what the next plan reads.
const operationSinkQueueSize = 10

// operationSink uploads operations one at a time on a background goroutine, so a deploy
// never waits on a round trip. queue holds resource keys and pending holds the newest
// operation per key, so a second write for a resource simply replaces the first.
type operationSink struct {
	uploader operationUploader

	// queue holds the keys that have something waiting. One slot per resource, so a full
	// queue means the deploy is that many resources ahead and the next write waits. Record
	// outside the state DB lock, or that wait blocks every other resource too.
	queue chan string

	// done is closed once the uploader has drained the queue and returned.
	done chan struct{}

	// stopQueue closes the queue, wrapped so close can safely run twice.
	stopQueue func()

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest operation per resource key, absent once the uploader takes it.
	pending map[string]recordedOperation

	err error
}

// newOperationSink starts the uploader. It returns nil when recording is off, and every
// method is a no-op on a nil sink. ctx must outlive close.
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

// RecordOperation implements dstate.OperationSink, turning every state write into an
// operation so DMS mirrors the local state. state is the serialized envelope, and nil for
// a delete. An earlier failure does not stop it: keep recording, best effort.
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

// recordFailure records that a resource did not apply, so the history says why rather
// than leaving the resource out.
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

// record makes op the one waiting for resourceKey, waiting itself while the queue is
// full. Recording after close panics, so every caller must return before close.
func (s *operationSink) record(resourceKey string, op recordedOperation) {
	s.mu.Lock()
	waiting, queued := s.pending[resourceKey]
	if queued {
		op = coalesce(waiting, op)
	}
	s.pending[resourceKey] = op
	s.mu.Unlock()

	// Already queued: the uploader reads the map when it gets to the key, so it picks up
	// what was just stored. No second slot, and no waiting.
	if !queued {
		s.queue <- resourceKey
	}
}

// coalesce merges a write that superseded another still waiting. The newer one wins,
// except a failure, which only carries pre-deploy state: it takes the superseded write's
// state and mask instead - absent state included, since a recreate's delete did remove it.
func coalesce(older, newer recordedOperation) recordedOperation {
	if newer.isFailure() {
		newer.state = older.state
		newer.resourceID = older.resourceID
		newer.updateFields = older.updateFields
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
			// Unreachable: a key is queued only when nothing was waiting for it. Guard so
			// a stray key could never upload a zero-valued operation.
			continue
		}

		// Keep going after a failure, so one bad upload does not drop everything behind it.
		if err := s.uploader.upload(ctx, resourceKey, op); err != nil {
			s.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
		}
	}
}

// close drains what is waiting and returns the first upload error, which fails the deploy:
// DMS is the source of truth, so a missing record would have the next deploy create a
// resource that already exists. Safe to call twice.
func (s *operationSink) close() error {
	if s == nil {
		return nil
	}

	s.stopQueue()
	<-s.done
	return s.firstErr()
}

// setErr keeps the first error; one failure is enough to fail the deploy.
func (s *operationSink) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err == nil {
		s.err = err
	}
}

// firstErr returns the first recording error, or nil. A nil sink never errors.
func (s *operationSink) firstErr() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}
