package dms

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// bufferedOperations caps how far ahead of the service a deploy may get; DMS is what the next
// plan reads.
const bufferedOperations = 10

const historyDrainTimeoutPerOperation = 5 * time.Second

// BufferOptions selects optional deployment history shadow recording behavior.
type BufferOptions struct {
	BestEffort bool
	// OperationCount counts staged resource keys. All recorded keys must be staged.
	// File history uses it for queue capacity and the drain timeout.
	OperationCount int
}

// OperationBufferStats describes one deployment version's history uploads.
type OperationBufferStats struct {
	OperationCount          int
	MaxQueueSize            int
	UndrainedOperationCount int
}

// OperationBuffer records each state write with DMS for one deployment version, off the apply
// path: writes are queued and sent on one background goroutine. It exists only while a bundle
// records deployment history; callers hold a nil buffer otherwise and must not call it.
type OperationBuffer struct {
	deploymentID string
	versionNum   int

	// The buffer. queue holds bundle state keys, and pending the newest update per key, so a
	// second write for a resource replaces the first.
	queue     chan string
	done      chan struct{}
	stopQueue func()

	// cancel stops in-flight history uploads when the drain deadline expires.
	cancel       context.CancelFunc
	bestEffort   bool
	drainTimeout time.Duration

	// sequenceIDs holds the token the last update for a resource returned. A missing key reads as
	// 0, which is what staging leaves. Unguarded: run is the only goroutine that writes, one update at a time.
	//
	// TODO: revisit and possibly deprecate. Sequence ids guard against an earlier update
	// overwriting a later one, which cannot happen here - updates for a resource are sent one at a
	// time from a single goroutine. UpdateOperation requires them today, so the client cannot
	// simply stop sending them.
	sequenceIDs map[string]int64

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest update per resource key, absent once the writer takes it.
	pending            map[string]OperationUpdate
	inFlight           string
	stats              OperationBufferStats
	drainStatsRecorded bool

	// latestState is the newest state-bearing update per resource key, kept for the buffer's
	// whole life - unlike pending, which the writer removes on take. RecordFailure merges its
	// status and message onto this so a failure keeps the state the last write recorded (which
	// the service may already hold), not the pre-deploy state, which would overwrite it.
	latestState map[string]OperationUpdate

	err error
}

// StartOperationBuffer opens the buffer for the version the caller just created. The version
// must already exist: operations record under it, and nothing here creates it.
func StartOperationBuffer(ctx context.Context, deploymentID string, versionNum int, options BufferOptions) *OperationBuffer {
	// History can stop independently without canceling the deployment.
	ctx, cancel := context.WithCancel(ctx)
	capacity := bufferedOperations
	if options.BestEffort {
		// Updates coalesce, so each staged key occupies at most one queue slot.
		// The worker frees that slot before the key can be queued again.
		// One slot per staged key therefore avoids deployment backpressure.
		capacity = options.OperationCount
	}
	b := &OperationBuffer{
		deploymentID: deploymentID,
		versionNum:   versionNum,
		queue:        make(chan string, capacity),
		done:         make(chan struct{}),
		cancel:       cancel,
		bestEffort:   options.BestEffort,
		drainTimeout: time.Duration(options.OperationCount) * historyDrainTimeoutPerOperation,
		pending:      make(map[string]OperationUpdate),
		latestState:  make(map[string]OperationUpdate),
		sequenceIDs:  make(map[string]int64),
		stats:        OperationBufferStats{OperationCount: options.OperationCount},
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

	b.record(resourceKey, update, false, nil)
}

// RecordFailure records that a resource did not apply, so the history says why. state is a
// fallback used only when no write recorded state yet; otherwise record keeps the last state.
func (b *OperationBuffer) RecordFailure(resourceKey, resourceID string, state json.RawMessage, cause error) {
	// The passed update is the fallback used only when no write recorded state yet; record
	// otherwise merges the failure onto the state the last write recorded (see below).
	b.record(resourceKey, NewFailureUpdate(resourceID, state, cause), true, cause)
}

// record queues update for resourceKey. keepExistingUpdateIfAny (a failure) merges onto the
// resource's last recorded state if one exists; otherwise (a state write) update is saved as
// that latest state.
func (b *OperationBuffer) record(resourceKey string, update OperationUpdate, keepExistingUpdateIfAny bool, cause error) {
	b.mu.Lock()
	if keepExistingUpdateIfAny {
		if latest, recorded := b.latestState[resourceKey]; recorded {
			update = latest.Merge(failureUpdate(cause))
		}
	} else {
		b.latestState[resourceKey] = update
	}
	waiting, queued := b.pending[resourceKey]
	if queued {
		update = waiting.Merge(update)
	}
	b.pending[resourceKey] = update
	b.stats.MaxQueueSize = max(b.stats.MaxQueueSize, len(b.pending))
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
	if ok {
		b.inFlight = resourceKey
	}
	return update, ok
}

func (b *OperationBuffer) run(ctx context.Context) {
	defer close(b.done)
	defer b.cancel()

	for {
		var resourceKey string
		var ok bool
		if b.bestEffort {
			select {
			case resourceKey, ok = <-b.queue:
			case <-ctx.Done():
				// Drain calls cancel on timeout. Parent cancellation can also stop history.
				b.setErr(ctx.Err())
				return
			}
		} else {
			resourceKey, ok = <-b.queue
		}
		if !ok {
			return
		}
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
		b.mu.Lock()
		b.inFlight = ""
		b.mu.Unlock()
	}
}

// write sends one update, at the sequence id the resource is at.
func (b *OperationBuffer) write(ctx context.Context, key string, update OperationUpdate) error {
	w := cmdctx.WorkspaceClient(ctx)
	result, err := w.BundleDeployments.UpdateOperation(ctx, bundledeployments.UpdateOperationRequest{
		Name:       OperationName(b.deploymentID, b.versionNum, key),
		Operation:  update.operation(b.sequenceIDs[key]),
		UpdateMask: update.Fields.Mask(),
	})
	if err != nil {
		return err
	}

	// The next write for this resource echoes the sequence id this one earned.
	b.sequenceIDs[key] = result.SequenceId
	return nil
}

// Drain waits for buffered uploads before version completion.
//   - WritingDeploymentHistoryOnly: close the channel and wait for queued writes.
//   - WritingDeploymentHistoryShadow: wait up to five seconds per staged operation.
//
// Shadow history cancels outstanding uploads when the timeout or ctx expires.
// It returns the first recording error and is safe to call more than once.
func (b *OperationBuffer) Drain(ctx context.Context) error {
	b.stopQueue()
	if !b.bestEffort {
		// DMS is authoritative: wait for all queued writes and return any recording error.
		<-b.done
		b.recordDrainStats()
		return b.Err()
	}

	// File state is already saved. Budget five seconds per staged operation,
	// including operations already uploaded; caller cancellation can shorten the wait.
	ctx, cancel := context.WithTimeout(ctx, b.drainTimeout)
	defer cancel()
	select {
	case <-b.done:
		// Report the first upload error even if later uploads succeeded.
		b.recordDrainStats()
		return b.Err()
	case <-ctx.Done():
		// Deadline or cancellation: stop the upload so deployment can finish promptly.
		b.recordDrainStats()
		b.setErr(ctx.Err())
		b.cancel()
		return b.Err()
	}
}

// recordDrainStats freezes outstanding resource counts before cancellation can clear an upload.
func (b *OperationBuffer) recordDrainStats() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.drainStatsRecorded {
		return
	}
	b.drainStatsRecorded = true
	b.stats.UndrainedOperationCount = len(b.pending)
	if _, queued := b.pending[b.inFlight]; b.inFlight != "" && !queued {
		b.stats.UndrainedOperationCount++
	}
}

// Stats returns a snapshot; undrained operations are counted once when Drain ends.
func (b *OperationBuffer) Stats() OperationBufferStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stats
}

// setErr keeps the first recording error for Drain to report.
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
