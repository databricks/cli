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

// operationSink uploads recorded operations from one background goroutine, so a
// deploy never waits on the CreateOperation round trip and the service sees one
// request at a time. Two rules shape it:
//
//   - Newest write wins. Each carries the resource's full state, so a write waiting
//     behind an upload is replaced rather than queued ("coalesced"); see coalesce for
//     the one field that is carried over instead.
//   - Uploads happen while apply runs. firstErr is what stops the deploy: DMS becomes
//     the source of truth (see dstate.readDMSState), so a missing record would make
//     the next deploy create a resource that already exists.
type operationSink struct {
	uploader operationUploader

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the one operation waiting per resource key. No key means nothing
	// is waiting; a resource can write state more than once in a deploy (a recreate
	// drops the entry, then saves the new resource) and the later write replaces the
	// earlier one here.
	pending map[string]recordedOperation

	// closed stops the uploader once everything recorded before close has gone up.
	closed bool

	err error

	// wake reports that pending may have work. Buffered so recording never blocks on
	// the uploader, and only ever holds one token: a full buffer already means "look
	// again", which is all the uploader needs to know.
	wake chan struct{}

	// done is closed when the uploader has drained pending and returned.
	done chan struct{}

	// signalClose tells the uploader to stop once pending is empty. Wrapped so close
	// can be deferred and still checked at a specific point.
	signalClose func()
}

// newOperationSink starts the uploader, returning nil when uploader is nil (recording
// off; every method is a no-op on a nil sink). ctx must outlive close.
func newOperationSink(ctx context.Context, uploader operationUploader) *operationSink {
	if uploader == nil {
		return nil
	}

	s := &operationSink{
		uploader: uploader,
		pending:  make(map[string]recordedOperation),
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	s.signalClose = sync.OnceFunc(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()

		// Wake the uploader so it notices, in case it is waiting on an empty pending.
		s.notify()
	})

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
		// The deploy already persisted this write locally, so failing it here would
		// report an error about history for a resource that deployed fine.
		log.Warnf(ctx, "Not recording operation for %s: %s", resourceKey, err)
		return
	}

	s.record(resourceKey, op)
}

// recordFailure records that applying a resource failed, so the deployment history
// explains the failure instead of omitting the resource. It returns nothing: the
// deploy is already failing, and a second error would mask the one the user needs.
func (s *operationSink) recordFailure(ctx context.Context, resourceKey string, action deployplan.ActionType, resourceID string, priorState json.RawMessage, cause error) {
	if s == nil {
		return
	}

	op, err := newFailedOperation(action, resourceID, priorState, cause)
	if err != nil {
		log.Warnf(ctx, "Not recording failure for %s: %s", resourceKey, err)
		return
	}

	s.record(resourceKey, op)
}

// record makes op the operation waiting for resourceKey and wakes the uploader.
func (s *operationSink) record(resourceKey string, op recordedOperation) {
	s.mu.Lock()
	if waiting, ok := s.pending[resourceKey]; ok {
		op = coalesce(waiting, op)
	}
	s.pending[resourceKey] = op
	s.mu.Unlock()

	s.notify()
}

// notify reports that there may be work, without ever blocking the caller. A dropped
// send means a token is already buffered, which the uploader has yet to consume - and
// it re-reads pending before it waits again, so it sees this operation either way.
func (s *operationSink) notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// coalesce folds a write that never got uploaded into the one replacing it. The newer
// write describes the resource as it now stands, so its fields win.
//
// A failure is the exception: it carries the resource's state only when there was a
// pre-deploy record to carry (see newFailedOperation), so a create that wrote state
// and then failed would replace that state with nothing and drop the resource from the
// deployment. Inherit what the superseded write recorded instead, since that is the
// resource the failure is reporting on. The same reasoning applies on the wire when
// the write did get uploaded; see failureFields.
func coalesce(older, newer recordedOperation) recordedOperation {
	if newer.isFailure() && newer.state == nil && older.state != nil {
		newer.state = older.state
		newer.resourceID = older.resourceID
	}
	return newer
}

// take claims the operation waiting for one resource. Which resource comes first is
// unspecified: a resource has at most one operation waiting, so order matters only
// within a resource, and there coalesce has already settled it.
func (s *operationSink) take() (string, recordedOperation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for resourceKey, op := range s.pending {
		delete(s.pending, resourceKey)
		return resourceKey, op, true
	}
	return "", recordedOperation{}, false
}

func (s *operationSink) run(ctx context.Context) {
	defer close(s.done)

	for {
		resourceKey, op, ok := s.take()
		if ok {
			// Keep going after a failure, so one bad upload does not drop the records
			// for every resource behind it.
			if err := s.uploader.upload(ctx, resourceKey, op); err != nil {
				s.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
			}
			continue
		}

		// Nothing waiting. Exiting only once pending is empty is what makes close a
		// drain: everything recorded before it has been uploaded by the time it
		// returns.
		if s.isClosed() {
			return
		}
		<-s.wake
	}
}

// close drains the pending operations and returns the first upload error. Every
// record caller must have returned first; calling close twice is safe, so it can be
// deferred and still checked at a specific point.
func (s *operationSink) close() error {
	if s == nil {
		return nil
	}

	s.signalClose()
	<-s.done
	return s.firstErr()
}

func (s *operationSink) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// setErr keeps the first upload error; later ones are dropped because one failure
// is enough to fail the deploy.
func (s *operationSink) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err == nil {
		s.err = err
	}
}

// firstErr returns the first upload error, or nil if every upload so far succeeded.
// A nil sink (recording disabled) never errors.
func (s *operationSink) firstErr() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}
