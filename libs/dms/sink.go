package dms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// operationSinkQueueSize is how many resources may be waiting to be recorded before the
// next write has to wait. Without a cap a deploy could finish far ahead of the service,
// and DMS is what the next plan reads.
const operationSinkQueueSize = 10

// stagedSequenceID is what CreateVersion leaves on every operation it stages, and so the
// precondition for the first update of a resource.
const stagedSequenceID = "0"

// OperationSink fills in the operations one version staged. It writes them one at a time on a
// background goroutine, so a deploy never waits on a round trip. queue holds bundle state keys,
// converted where they go on the wire, and pending the newest update per key, so a second write
// for a resource replaces the first.
type OperationSink struct {
	client       *Client
	deploymentID string
	version      int64

	// queue holds the keys that have something waiting. One slot per resource, so a full
	// queue means the deploy is that many resources ahead and the next write waits. Record
	// outside the state DB lock, or that wait blocks every other resource too.
	queue chan string

	// done is closed once the writer has drained the queue and returned.
	done chan struct{}

	// stopQueue closes the queue, wrapped so Close can safely run twice.
	stopQueue func()

	// sequenceIDs holds the token the last update for a resource returned. A resource absent
	// from it has only what staging left, so its first update sends that. Unguarded: run is
	// the only goroutine that writes, one update at a time.
	sequenceIDs map[ResourceKey]string

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest update per resource key, absent once the writer takes it.
	pending map[string]OperationUpdate

	err error
}

// newOperationSink starts the writer. ctx must outlive Close.
func newOperationSink(ctx context.Context, client *Client, deploymentID string, version int64) *OperationSink {
	s := &OperationSink{
		client:       client,
		deploymentID: deploymentID,
		version:      version,
		queue:        make(chan string, operationSinkQueueSize),
		done:         make(chan struct{}),
		sequenceIDs:  make(map[ResourceKey]string),
		pending:      make(map[string]OperationUpdate),
	}
	s.stopQueue = sync.OnceFunc(func() { close(s.queue) })

	go s.run(ctx)
	return s
}

// RecordOperation turns a state write into an operation so DMS mirrors the local state.
// resourceKey is the bundle state key, state the serialized envelope, and nil for a delete.
// An earlier failure does not stop it: keep recording, best effort.
func (s *OperationSink) RecordOperation(ctx context.Context, resourceKey string, inProgress bool, resourceID string, state json.RawMessage) {
	update, err := NewStateUpdate(resourceID, state, inProgress)
	if err != nil {
		s.setErr(fmt.Errorf("recording operation for %s: %w", resourceKey, err))
		return
	}

	s.record(resourceKey, update)
}

// RecordFailure records that a resource did not apply, so the history says why rather
// than leaving the resource out.
func (s *OperationSink) RecordFailure(resourceKey, resourceID string, cause error) {
	s.record(resourceKey, NewFailureUpdate(resourceID, cause))
}

// record makes update the one waiting for resourceKey, waiting itself while the queue is
// full. Recording after Close panics, so every caller must return before Close.
func (s *OperationSink) record(resourceKey string, update OperationUpdate) {
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
func (s *OperationSink) take(resourceKey string) (OperationUpdate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	update, ok := s.pending[resourceKey]
	delete(s.pending, resourceKey)
	return update, ok
}

func (s *OperationSink) run(ctx context.Context) {
	defer close(s.done)

	for resourceKey := range s.queue {
		update, ok := s.take(resourceKey)
		if !ok {
			// Unreachable: a key is queued only when nothing was waiting for it. Guard so
			// a stray key could never write a zero-valued update.
			continue
		}

		// Keep going after a failure, so one bad write does not drop everything behind it.
		if err := s.write(ctx, KeyFromState(resourceKey), update); err != nil {
			s.setErr(fmt.Errorf("recording operation for %s with the deployment metadata service: %w", resourceKey, err))
		}
	}
}

// write sends one update, at the sequence id the resource is at.
func (s *OperationSink) write(ctx context.Context, key ResourceKey, update OperationUpdate) error {
	sequenceID, written := s.sequenceIDs[key]
	if !written {
		sequenceID = stagedSequenceID
	}

	next, err := s.client.UpdateOperation(ctx, s.deploymentID, s.version, key, sequenceID, update)
	if err != nil {
		return err
	}

	// The next write for this resource echoes the sequence id this one earned.
	s.sequenceIDs[key] = next

	return nil
}

// Close drains what is waiting and returns the first write error, which fails the deploy:
// DMS is the source of truth, so a missing record would have the next deploy create a
// resource that already exists. Safe to call twice.
func (s *OperationSink) Close() error {
	s.stopQueue()
	<-s.done
	return s.FirstErr()
}

// setErr keeps the first error; one failure is enough to fail the deploy.
func (s *OperationSink) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err == nil {
		s.err = err
	}
}

// FirstErr returns the first recording error, or nil.
func (s *OperationSink) FirstErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}
