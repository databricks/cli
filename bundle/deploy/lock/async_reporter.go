package lock

import (
	"context"
	"encoding/json"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/log"
)

// Matches direct.defaultParallelism so a hard process crash drops at most
// ~10 unsent operation events.
const asyncReporterBufferSize = 10

type operationEvent struct {
	resourceKey  string
	resourceID   string
	action       deployplan.ActionType
	operationErr error
	state        json.RawMessage
}

// asyncReporter dispatches DMS operation reports from a single sender
// goroutine fed by a buffered channel. CRUD workers push and continue;
// when the buffer is full the send blocks, applying backpressure to the
// worker pool. Reporting is best-effort — DMS API errors are logged and
// the sender keeps draining.
type asyncReporter struct {
	ch     chan operationEvent
	done   chan struct{}
	sendFn func(ctx context.Context, ev operationEvent) error
	ctx    context.Context
}

// newAsyncReporter starts the sender goroutine. ctx is used for all DMS API
// calls and must outlive individual worker contexts.
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
			log.Warnf(r.ctx, "Failed to report %s operation for %s to DMS: %v", ev.action, ev.resourceKey, err)
		}
	}
}

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
