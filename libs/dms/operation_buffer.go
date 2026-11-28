package dms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// bufferedOperations caps queued resource keys in authoritative mode. The writer can also
// hold a batch of up to maxBatchOperations updates and one carried update.
const bufferedOperations = 10

const historyDrainTimeoutPerOperation = 5 * time.Second

// maxBatchOperations and maxBatchBytes cap one BatchUpdateOperations request: the service takes
// 1-20 updates and asks callers to stay under 1 MiB in all.
const (
	maxBatchOperations = 20
	maxBatchBytes      = 1 << 20
)

type recordingState uint8

const (
	recordingPending recordingState = iota
	recordingSucceeded
	recordingFailed
)

type bufferedUpdate struct {
	update OperationUpdate
	state  recordingState
}

// BufferOptions selects optional deployment history shadow recording behavior.
type BufferOptions struct {
	BestEffort bool
	// StagedOperationCount counts staged resource keys. All recorded keys must be staged.
	// File history uses it for queue capacity and the drain timeout.
	StagedOperationCount int
}

// OperationBufferStats describes one deployment version's history uploads.
type OperationBufferStats struct {
	StagedOperationCount         int
	PeakPendingUpdateCount       int
	QueuedUpdateCountAtTimeout   int
	RecordingAttemptFailureCount int
	// Successful history uploads. Intermediate and repeated updates count separately.
	RecordingAttemptSuccessCount int
	// Resource operations whose latest final history update uploaded successfully.
	OperationRecordingSuccessCount int
	// Resource operations whose latest final history update failed to build or upload.
	OperationRecordingFailureCount int
	// Staged resource operations whose latest final history update has not completed,
	// including queued, in-flight, and never-queued updates.
	OperationRecordingPendingCount int
}

// OperationBuffer records each state write with DMS for one deployment version, off the apply
// path: writes are queued and sent on one background goroutine. It exists only while a bundle
// records deployment history; callers hold a nil buffer otherwise and must not call it.
type OperationBuffer struct {
	deploymentID string
	versionNum   int

	// heartbeat keeps the version's update_time fresh until Stop, past Drain. Each write renews
	// it as it starts, since the service counts a write as a heartbeat.
	heartbeat *Heartbeat

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
	// 0, which is what staging leaves. Unguarded: only run accesses it, and batches are sent serially.
	//
	// TODO: revisit and possibly deprecate. Sequence ids guard against an earlier update
	// overwriting a later one, which cannot happen here - updates for a resource are sent in order
	// from a single goroutine. The service requires them today, so the client cannot
	// simply stop sending them.
	sequenceIDs map[string]int64

	// apiClient sends the batches. Built by run on its first write, and only run uses it.
	apiClient *databricksclient.DatabricksClient

	// mu guards the fields below.
	mu sync.Mutex

	// pending holds the newest update per resource key, absent once the writer takes it.
	pending map[string]*bufferedUpdate
	stats   OperationBufferStats
	// Tracks the recording outcome of each resource's latest update.
	operations map[string]*bufferedUpdate

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
	// Started on ctx, not the writer's context below, so it keeps running after Drain.
	heartbeat := startVersionHeartbeat(ctx, deploymentID, versionNum)
	// History can stop independently without canceling the deployment.
	ctx, cancel := context.WithCancel(ctx)
	capacity := bufferedOperations
	if options.BestEffort {
		// Updates coalesce, so each staged key occupies at most one queue slot.
		// The worker frees that slot before the key can be queued again.
		// One slot per staged key therefore avoids deployment backpressure.
		capacity = options.StagedOperationCount
	}
	b := &OperationBuffer{
		deploymentID: deploymentID,
		versionNum:   versionNum,
		heartbeat:    heartbeat,
		queue:        make(chan string, capacity),
		done:         make(chan struct{}),
		cancel:       cancel,
		bestEffort:   options.BestEffort,
		drainTimeout: time.Duration(options.StagedOperationCount) * historyDrainTimeoutPerOperation,
		pending:      make(map[string]*bufferedUpdate),
		latestState:  make(map[string]OperationUpdate),
		sequenceIDs:  make(map[string]int64),
		stats:        OperationBufferStats{StagedOperationCount: options.StagedOperationCount},
		operations:   make(map[string]*bufferedUpdate, options.StagedOperationCount),
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
		b.mu.Lock()
		b.stats.RecordingAttemptFailureCount++
		if !inProgress {
			b.operations[resourceKey] = &bufferedUpdate{state: recordingFailed}
		}
		b.mu.Unlock()
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
		update = waiting.update.Merge(update)
	}
	entry := &bufferedUpdate{update: update, state: recordingPending}
	b.pending[resourceKey] = entry
	b.operations[resourceKey] = entry
	b.stats.PeakPendingUpdateCount = max(b.stats.PeakPendingUpdateCount, len(b.pending))
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
func (b *OperationBuffer) take(resourceKey string) (*bufferedUpdate, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry, ok := b.pending[resourceKey]
	delete(b.pending, resourceKey)
	return entry, ok
}

// queuedWrite is an update the writer took off the queue.
type queuedWrite struct {
	key   string
	entry *bufferedUpdate
}

// batchEntry is a queued write with its request entry, encoded once so its size is known.
type batchEntry struct {
	queuedWrite
	body json.RawMessage
}

// The BatchUpdateOperations wire shapes. Each entry names its operation in the body.
type batchUpdateOperationsRequest struct {
	Requests []json.RawMessage `json:"requests"`
}

type operationUpdateRequest struct {
	Operation  bundledeployments.Operation `json:"operation"`
	UpdateMask fieldmask.FieldMask         `json:"update_mask"`
}

type batchUpdateOperationsResponse struct {
	Operations []bundledeployments.Operation `json:"operations"`
}

// batchRequestOverhead is the size of a request with no entries.
const batchRequestOverhead = len(`{"requests":[]}`)

// run sends the queued updates in batches, each holding what was already waiting when it was
// built. One batch is in flight at a time.
func (b *OperationBuffer) run(ctx context.Context) {
	defer close(b.done)
	defer b.cancel()

	// carried was taken off the queue but could not join the previous batch.
	var carried *queuedWrite
	for {
		if carried == nil {
			w, ok := b.next(ctx, true)
			if !ok {
				return
			}
			carried = &w
		}
		var batch []batchEntry
		batch, carried = b.collect(ctx, *carried)

		// Keep going after a failure, so one bad write does not drop everything behind it.
		if err := b.write(ctx, batch); err != nil {
			b.setErr(fmt.Errorf("recording operation for %s: %w", batchKeys(batch), err))
		}
	}
}

// next takes the next queued update. With wait it blocks for one; without, it reports false when
// none is queued. It also reports false once the queue is closed and drained, and, for best-effort
// history, once ctx is done.
func (b *OperationBuffer) next(ctx context.Context, wait bool) (queuedWrite, bool) {
	for {
		var resourceKey string
		var ok bool
		switch {
		case !wait:
			select {
			case resourceKey, ok = <-b.queue:
			default:
				return queuedWrite{}, false
			}
		case b.bestEffort:
			select {
			case resourceKey, ok = <-b.queue:
			case <-ctx.Done():
				// Drain calls cancel on timeout. Parent cancellation can also stop history.
				b.setErr(ctx.Err())
				return queuedWrite{}, false
			}
		default:
			resourceKey, ok = <-b.queue
		}
		if !ok {
			return queuedWrite{}, false
		}
		if entry, ok := b.take(resourceKey); ok {
			return queuedWrite{key: resourceKey, entry: entry}, true
		}
		// Unreachable: a key is queued only when nothing was waiting for it. Guard so
		// a stray key could never write a zero-valued update.
	}
}

// collect builds a batch from first and the updates already queued behind it, up to what one
// request may carry. It returns the update that ended the batch early: one past the size cap, or
// a resource written again after the batch took it, whose write needs the sequence id this batch
// earns.
func (b *OperationBuffer) collect(ctx context.Context, first queuedWrite) ([]batchEntry, *queuedWrite) {
	var batch []batchEntry
	size := batchRequestOverhead
	w := first
	for {
		if slices.ContainsFunc(batch, func(e batchEntry) bool { return e.key == w.key }) {
			return batch, &w
		}
		body, err := b.encode(w)
		switch {
		case err != nil:
			b.finish([]batchEntry{{queuedWrite: w}}, err)
			b.setErr(fmt.Errorf("recording operation for %s: %w", w.key, err))
		case len(batch) > 0 && size+len(body)+1 > maxBatchBytes:
			return batch, &w
		default:
			batch = append(batch, batchEntry{queuedWrite: w, body: body})
			size += len(body) + 1
		}
		if len(batch) == maxBatchOperations {
			return batch, nil
		}
		var ok bool
		if w, ok = b.next(ctx, false); !ok {
			return batch, nil
		}
	}
}

// encode builds w's request entry, at the sequence id its resource is at.
func (b *OperationBuffer) encode(w queuedWrite) (json.RawMessage, error) {
	return json.Marshal(operationUpdateRequest{
		Operation:  w.entry.update.operation(OperationName(b.deploymentID, b.versionNum, w.key), b.sequenceIDs[w.key]),
		UpdateMask: w.entry.update.Fields.Mask(),
	})
}

// write sends one batch. The service applies all of it or none of it, so every update in the
// batch shares the outcome.
func (b *OperationBuffer) write(ctx context.Context, batch []batchEntry) error {
	if len(batch) == 0 {
		return nil
	}
	// Renewed when the write starts, not when it succeeds, so a heartbeat only overlaps a write
	// that has been in flight for a full interval. Both update the version row.
	b.heartbeat.renew()

	result, err := b.send(ctx, batch)
	if err == nil && len(result.Operations) != len(batch) {
		err = fmt.Errorf("the service returned %d operations for %d updates", len(result.Operations), len(batch))
	}
	b.finish(batch, err)
	if err != nil {
		return err
	}

	// The next write for each resource echoes the sequence id this one earned. Results come back
	// in request order.
	for i, e := range batch {
		b.sequenceIDs[e.key] = result.Operations[i].SequenceId
	}
	return nil
}

// send posts batch through a raw client, since the SDK leaves BatchUpdateOperations out while
// the service stages the method DEVELOPMENT.
//
// TODO: switch to the SDK's generated method once the SDK has it.
func (b *OperationBuffer) send(ctx context.Context, batch []batchEntry) (batchUpdateOperationsResponse, error) {
	var result batchUpdateOperationsResponse
	if b.apiClient == nil {
		client, err := databricksclient.New(cmdctx.WorkspaceClient(ctx).Config)
		if err != nil {
			return result, err
		}
		b.apiClient = client
	}
	request := batchUpdateOperationsRequest{Requests: make([]json.RawMessage, len(batch))}
	for i, e := range batch {
		request.Requests[i] = e.body
	}
	path := "/api/2.0/bundle/" + VersionName(b.deploymentID, b.versionNum) + "/operations:batchUpdate"
	err := b.apiClient.Do(ctx, http.MethodPost, path, auth.WorkspaceIDHeaders(b.apiClient.Config), nil, request, &result)
	return result, err
}

// finish records the outcome of the batch's upload for each update in it.
func (b *OperationBuffer) finish(batch []batchEntry, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, e := range batch {
		if err != nil {
			b.stats.RecordingAttemptFailureCount++
			e.entry.state = recordingFailed
		} else {
			b.stats.RecordingAttemptSuccessCount++
			e.entry.state = recordingSucceeded
		}
	}
}

func batchKeys(batch []batchEntry) string {
	keys := make([]string, len(batch))
	for i, e := range batch {
		keys[i] = e.key
	}
	return strings.Join(keys, ", ")
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
		return b.Err()
	}

	// File state is already saved. Budget five seconds per staged operation,
	// including operations already uploaded; caller cancellation can shorten the wait.
	ctx, cancel := context.WithTimeout(ctx, b.drainTimeout)
	defer cancel()
	select {
	case <-b.done:
		// Report the first upload error even if later uploads succeeded.
		return b.Err()
	case <-ctx.Done():
		// Deadline or cancellation: stop the upload so deployment can finish promptly.
		b.mu.Lock()
		defer b.mu.Unlock()
		b.stats.QueuedUpdateCountAtTimeout = len(b.queue)
		if b.err == nil {
			b.err = ctx.Err()
		}
		b.cancel()
		return b.err
	}
}

// Stop ends the heartbeat, the buffer's only background work left after Drain.
func (b *OperationBuffer) Stop() {
	b.heartbeat.Stop()
}

// Stats returns a snapshot of the history measurements.
func (b *OperationBuffer) Stats() OperationBufferStats {
	b.mu.Lock()
	defer b.mu.Unlock()
	stats := b.stats
	for _, entry := range b.operations {
		// A recreation's delete update is not the operation's final result.
		if entry.update.Status == StatusPending {
			continue
		}
		switch entry.state {
		case recordingPending:
			continue
		case recordingSucceeded:
			stats.OperationRecordingSuccessCount++
		case recordingFailed:
			stats.OperationRecordingFailureCount++
		}
	}
	stats.OperationRecordingPendingCount = stats.StagedOperationCount - stats.OperationRecordingSuccessCount - stats.OperationRecordingFailureCount
	return stats
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
