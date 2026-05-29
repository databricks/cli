package lock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/log"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// asyncReporterBufferSize matches direct.defaultParallelism so that, in the
// worst case, a hard process crash drops at most ~10 unsent operation events.
// The send blocks when the buffer is full, applying backpressure to the apply
// worker pool.
const asyncReporterBufferSize = 10

// operationEvent is one resource-apply outcome enqueued for delivery to DMS.
type operationEvent struct {
	resourceKey  string
	resourceID   string
	action       deployplan.ActionType
	operationErr error
	state        json.RawMessage
}

// asyncReporter ships operationEvents to DMS from a single sender goroutine
// fed by a buffered channel. Apply workers push events and continue; the
// sender drains the channel and logs any DMS-side failures without
// interrupting the deploy. Close blocks until the sender drains.
//
// We send asynchronously because operation reporting is on the hot path of
// every Create/Update/Delete and the DMS call is a network round-trip we
// don't want to serialize the deploy on. The buffer size matches the apply
// parallelism so a single batch of operations can be enqueued without
// blocking; once full, sends apply backpressure to the worker pool.
type asyncReporter struct {
	ch     chan operationEvent
	done   chan struct{}
	sendFn func(ctx context.Context, ev operationEvent) error
	ctx    context.Context
}

// newAsyncReporter starts the sender goroutine. The given ctx is used for all
// DMS API calls; it must outlive individual worker contexts (callers pass the
// deploy-level context for this reason).
func newAsyncReporter(ctx context.Context, sendFn func(context.Context, operationEvent) error) *asyncReporter {
	r := &asyncReporter{
		ch:     make(chan operationEvent, asyncReporterBufferSize),
		done:   make(chan struct{}),
		sendFn: sendFn,
		ctx:    ctx,
	}
	go r.run()
	return r
}

func (r *asyncReporter) run() {
	defer close(r.done)
	for ev := range r.ch {
		if err := r.sendFn(r.ctx, ev); err != nil {
			// Reporting failures are intentionally non-fatal: the deploy
			// already succeeded (or failed independently), and we don't
			// want a DMS hiccup to surface as a deploy error. Matches the
			// heartbeat behaviour established in step 3.
			log.Warnf(r.ctx, "Failed to report %s operation for %s to DMS: %v", ev.action, ev.resourceKey, err)
		}
	}
}

// Reporter returns a direct.OperationReporter that enqueues onto the channel.
// The returned function is safe to call from multiple goroutines and returns
// quickly unless the buffer is full.
func (r *asyncReporter) Reporter() direct.OperationReporter {
	return func(
		ctx context.Context,
		resourceKey, resourceID string,
		action deployplan.ActionType,
		operationErr error,
		state json.RawMessage,
	) error {
		select {
		case r.ch <- operationEvent{
			resourceKey:  resourceKey,
			resourceID:   resourceID,
			action:       action,
			operationErr: operationErr,
			state:        state,
		}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Close signals end-of-input and waits for the sender to drain.
func (r *asyncReporter) Close() {
	close(r.ch)
	<-r.done
}

// makeOperationSender returns the synchronous "send one event to DMS" function
// consumed by asyncReporter. Skip actions short-circuit to nil; mapping errors
// and API errors are returned to the caller (asyncReporter logs and continues).
func makeOperationSender(svc sdkbundle.BundleInterface, deploymentID, versionID string) func(context.Context, operationEvent) error {
	return func(ctx context.Context, ev operationEvent) error {
		// Internal state uses fully-qualified keys like "resources.jobs.foo".
		// DMS-side resource keys omit the "resources." prefix.
		apiKey := strings.TrimPrefix(ev.resourceKey, "resources.")

		actionType, err := planActionToOperationAction(ev.action)
		if err != nil {
			return fmt.Errorf("mapping action for resource %s: %w", ev.resourceKey, err)
		}
		if actionType == "" {
			// Skip actions are not reported — there is nothing for DMS to record.
			return nil
		}

		status := sdkbundle.OperationStatusOperationStatusSucceeded
		var errorMessage string
		if ev.operationErr != nil {
			status = sdkbundle.OperationStatusOperationStatusFailed
			errorMessage = ev.operationErr.Error()
		}

		op := sdkbundle.Operation{
			ResourceKey:  apiKey,
			ResourceId:   ev.resourceID,
			Status:       status,
			ActionType:   actionType,
			ErrorMessage: errorMessage,
		}
		if len(ev.state) > 0 {
			s := ev.state
			op.State = &s
		}

		_, err = svc.CreateOperation(ctx, sdkbundle.CreateOperationRequest{
			Parent:      fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID),
			ResourceKey: apiKey,
			Operation:   op,
		})
		if err != nil {
			return fmt.Errorf("reporting operation for resource %s: %w", ev.resourceKey, err)
		}
		return nil
	}
}

// planActionToOperationAction maps a deploy-plan action onto the DMS
// OperationActionType enum. The Skip action is mapped to the empty string so
// the caller can drop it (DMS has no concept of a no-op operation).
//
// Bind / BindAndUpdate / InitialRegister are not currently produced by the
// direct planner, so they are intentionally not mapped here — adding new plan
// actions in the planner without updating this mapping will fail loud at
// runtime (default branch returns an error).
func planActionToOperationAction(action deployplan.ActionType) (sdkbundle.OperationActionType, error) {
	switch action {
	case deployplan.Skip:
		return "", nil
	case deployplan.Create:
		return sdkbundle.OperationActionTypeOperationActionTypeCreate, nil
	case deployplan.Update:
		return sdkbundle.OperationActionTypeOperationActionTypeUpdate, nil
	case deployplan.UpdateWithID:
		return sdkbundle.OperationActionTypeOperationActionTypeUpdateWithId, nil
	case deployplan.Delete:
		return sdkbundle.OperationActionTypeOperationActionTypeDelete, nil
	case deployplan.Recreate:
		return sdkbundle.OperationActionTypeOperationActionTypeRecreate, nil
	case deployplan.Resize:
		return sdkbundle.OperationActionTypeOperationActionTypeResize, nil
	default:
		return "", fmt.Errorf("unsupported operation action type: %s", action)
	}
}
