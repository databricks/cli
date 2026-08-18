package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/databricks/cli/libs/dms"
)

// operationSinkQueueSize is how many resources may be waiting to be recorded before the
// next write has to wait. Without a cap a deploy could finish far ahead of the service,
// and DMS is what the next plan reads.
const operationSinkQueueSize = 10

// operationSink writes operations one at a time on a background goroutine, so a deploy never
// waits on a round trip. queue holds bundle state keys, converted where they go on the wire,
// and pending the newest update per key, so a second write for a resource replaces the first.
type operationSink struct {
	writer dms.OperationWriter

	// queue holds the keys that have something waiting. One slot per resource, so a full
	// queue means the deploy is that many resources ahead and the next write waits. Record
	// outside the state DB lock, or that wait blocks every other resource too.
	queue chan string

	// done is closed once the writer has drained the queue and returned.
	done chan struct{}

	// stopQueue closes the queue, wrapped so close can safely run twice.
	stopQueue func()

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest update per resource key, absent once the writer takes it.
	pending map[string]dms.OperationUpdate

	err error
}

// newOperationSink starts the writer. It returns nil when recording is off, and every
// method is a no-op on a nil sink. ctx must outlive close.
func newOperationSink(ctx context.Context, writer dms.OperationWriter) *operationSink {
	if writer == nil {
		return nil
	}

	s := &operationSink{
		writer:  writer,
		queue:   make(chan string, operationSinkQueueSize),
		done:    make(chan struct{}),
		pending: make(map[string]dms.OperationUpdate),
	}
	s.stopQueue = sync.OnceFunc(func() { close(s.queue) })

	go s.run(ctx)
	return s
}

// RecordOperation implements dstate.OperationSink, turning every state write into an
// operation so DMS mirrors the local state. state is the serialized envelope, and nil for
// a delete. An earlier failure does not stop it: keep recording, best effort.
func (s *operationSink) RecordOperation(ctx context.Context, resourceKey string, inProgress bool, resourceID string, state json.RawMessage) {
	if s == nil {
		return
	}

	update, err := dms.NewStateUpdate(resourceID, state, inProgress)
	if err != nil {
		s.setErr(fmt.Errorf("recording operation for %s: %w", resourceKey, err))
		return
	}

	s.record(resourceKey, update)
}

// recordFailure records that a resource did not apply, so the history says why rather
// than leaving the resource out.
func (s *operationSink) recordFailure(resourceKey, resourceID string, cause error) {
	if s == nil {
		return
	}

	s.record(resourceKey, dms.NewFailureUpdate(resourceID, cause))
}

// record makes update the one waiting for resourceKey, waiting itself while the queue is
// full. Recording after close panics, so every caller must return before close.
func (s *operationSink) record(resourceKey string, update dms.OperationUpdate) {
	s.mu.Lock()
	waiting, queued := s.pending[resourceKey]
	if queued {
		update = waiting.Merge(update)
	}
	s.pending[resourceKey] = update
	s.mu.Unlock()

	// Already queued: the writer reads the map when it gets to the key, so it picks up
	// what was just stored. No second slot, and no waiting.
	if !queued {
		s.queue <- resourceKey
	}
}

// take claims the update waiting for resourceKey.
func (s *operationSink) take(resourceKey string) (dms.OperationUpdate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	update, ok := s.pending[resourceKey]
	delete(s.pending, resourceKey)
	return update, ok
}

func (s *operationSink) run(ctx context.Context) {
	defer close(s.done)

	for resourceKey := range s.queue {
		update, ok := s.take(resourceKey)
		if !ok {
			// Unreachable: a key is queued only when nothing was waiting for it. Guard so
			// a stray key could never write a zero-valued update.
			continue
		}

		// Keep going after a failure, so one bad write does not drop everything behind it.
		if err := s.writer.Write(ctx, dms.KeyFromState(resourceKey), update); err != nil {
			s.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
		}
	}
}

// close drains what is waiting and returns the first write error, which fails the deploy:
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
