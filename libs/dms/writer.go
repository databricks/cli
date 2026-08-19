package dms

import (
	"context"
	"sync"
)

// stagedSequenceID is what CreateVersion leaves on every operation it stages, and so the
// precondition for the first update of a resource.
const stagedSequenceID = "0"

// OperationWriter fills in the operations one version staged. Calls for different resources
// may run concurrently.
type OperationWriter interface {
	Write(ctx context.Context, key ResourceKey, update OperationUpdate) error
}

// operationWriter writes through the API, tracking the sequence id each resource is at.
type operationWriter struct {
	client       *Client
	deploymentID string
	version      int64

	mu sync.Mutex
	// sequenceIDs holds the token the last update for a resource returned. A resource
	// absent from it has only what staging left, so its first update sends that.
	sequenceIDs map[ResourceKey]string
}

func (w *operationWriter) Write(ctx context.Context, key ResourceKey, update OperationUpdate) error {
	w.mu.Lock()
	sequenceID, written := w.sequenceIDs[key]
	w.mu.Unlock()
	if !written {
		sequenceID = stagedSequenceID
	}

	next, err := w.client.UpdateOperation(ctx, w.deploymentID, w.version, key, sequenceID, update)
	if err != nil {
		return err
	}

	// The next write for this resource echoes the sequence id this one earned.
	w.mu.Lock()
	w.sequenceIDs[key] = next
	w.mu.Unlock()

	return nil
}
