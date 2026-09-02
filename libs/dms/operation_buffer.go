package dms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// bufferedOperations caps how far ahead of the service a deploy may get; DMS is what the next
// plan reads.
const bufferedOperations = 10

// stagedSequenceID is what version creation leaves on a staged operation, so a resource's first
// update sends it as the precondition.
const stagedSequenceID = "0"

// OperationBuffer records each state write with DMS for one deployment version, off the apply
// path: writes are queued and sent on one background goroutine. It exists only while a bundle
// records deployment history; callers hold a nil buffer otherwise and must not call it.
type OperationBuffer struct {
	client       *Client
	deploymentID string
	versionNum   int64

	// The buffer. queue holds bundle state keys, and pending the newest update per key, so a
	// second write for a resource replaces the first.
	queue     chan string
	done      chan struct{}
	stopQueue func()

	// sequenceIDs holds the token the last update for a resource returned. A resource absent
	// from it has only what staging left, so its first update sends that. Unguarded: run is the
	// only goroutine that writes, one update at a time.
	sequenceIDs map[string]string

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest update per resource key, absent once the writer takes it.
	pending map[string]OperationUpdate

	err error
}

// StartOperationBuffer opens the buffer for the version the caller just created. The version
// must already exist: operations record under it, and nothing here creates it.
func StartOperationBuffer(ctx context.Context, client *Client, deploymentID string, versionNum int64) *OperationBuffer {
	b := &OperationBuffer{
		client:       client,
		deploymentID: deploymentID,
		versionNum:   versionNum,
		queue:        make(chan string, bufferedOperations),
		done:         make(chan struct{}),
		pending:      make(map[string]OperationUpdate),
		sequenceIDs:  make(map[string]string),
	}
	b.stopQueue = sync.OnceFunc(func() { close(b.queue) })
	go b.run(ctx)
	return b
}

// RecordOperation records a state write. state is the serialized envelope, nil for a delete.
// An earlier failure does not stop it: keep recording, best effort.
func (b *OperationBuffer) RecordOperation(ctx context.Context, resourceKey string, inProgress bool, resourceID string, state json.RawMessage) {
	update, err := NewStateUpdate(resourceID, state, inProgress)
	if err != nil {
		b.setErr(fmt.Errorf("recording operation for %s: %w", resourceKey, err))
		return
	}

	b.record(resourceKey, update)
}

// RecordFailure records that a resource did not apply, so the history says why rather
// than leaving the resource out.
func (b *OperationBuffer) RecordFailure(resourceKey, resourceID string, state json.RawMessage, cause error) {
	b.record(resourceKey, NewFailureUpdate(resourceID, state, cause))
}

// record makes update the one waiting for resourceKey, waiting itself while the queue is
// full. Recording after Drain panics, so every caller must return before Drain.
func (b *OperationBuffer) record(resourceKey string, update OperationUpdate) {
	b.mu.Lock()
	waiting, queued := b.pending[resourceKey]
	if queued {
		update = waiting.Merge(update)
	}
	b.pending[resourceKey] = update
	b.mu.Unlock()

	// when already queued: the writer reads the map when it gets to the key,
	// so it picks up what was just stored.
	//
	// Only when the resource is not already queued, enqueue it.
	if !queued {
		b.queue <- resourceKey
	}
}

// take claims the update waiting for resourceKey.
func (b *OperationBuffer) take(resourceKey string) (OperationUpdate, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	update, ok := b.pending[resourceKey]
	delete(b.pending, resourceKey)
	return update, ok
}

func (b *OperationBuffer) run(ctx context.Context) {
	defer close(b.done)

	for resourceKey := range b.queue {
		update, ok := b.take(resourceKey)
		if !ok {
			// Unreachable: a key is queued only when nothing was waiting for it. Guard so
			// a stray key could never write a zero-valued update.
			continue
		}

		// Keep going after a failure, so one bad write does not drop everything behind it.
		if err := b.write(ctx, resourceKey, update); err != nil {
			b.setErr(fmt.Errorf("recording operation for %s: %w", resourceKey, err))
		}
	}
}

// write sends one update, at the sequence id the resource is at.
func (b *OperationBuffer) write(ctx context.Context, key string, update OperationUpdate) error {
	sequenceID, written := b.sequenceIDs[key]
	if !written {
		sequenceID = stagedSequenceID
	}

	next, err := b.client.UpdateOperation(ctx, b.deploymentID, b.versionNum, key, sequenceID, update)
	if err != nil {
		return err
	}

	// The next write for this resource echoes the sequence id this one earned.
	b.sequenceIDs[key] = next

	return nil
}

// Drain waits for everything buffered to reach the service and returns the first write error,
// which fails the deploy: DMS is the source of truth for what exists. Safe to call twice.
func (b *OperationBuffer) Drain() error {
	b.stopQueue()
	<-b.done
	return b.Err()
}

// setErr keeps the first error; one failure is enough to fail the deploy.
func (b *OperationBuffer) setErr(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.err == nil {
		b.err = err
	}
}

// Err returns the first recording error, or nil.
func (b *OperationBuffer) Err() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.err
}
