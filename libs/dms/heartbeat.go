package dms

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// heartbeatInterval is how long a version may go without an update before a heartbeat is sent.
// The UI shows a version as interrupted once its last update is over 2 minutes old.
const heartbeatInterval = 30 * time.Second

// heartbeatRetryDelays are the quick retries after a failed heartbeat, before waiting a full interval.
var heartbeatRetryDelays = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}

// Heartbeat keeps an open deployment version's last update time fresh, which is how the UI tells
// a running deploy from an interrupted one. CreateVersion and every successful write update it
// too, so a heartbeat is sent only after an interval without one. Best effort: a failure can only
// make the deploy show as interrupted, so it never fails the deploy.
type Heartbeat struct {
	send func(ctx context.Context) error

	renewed  chan struct{}
	stop     chan struct{}
	done     chan struct{}
	stopOnce func()
}

// StartHeartbeat starts heartbeating the version the caller just created. CreateVersion counts
// as the first update, so the first heartbeat goes out an interval later.
func StartHeartbeat(ctx context.Context, deploymentID string, versionNum int) *Heartbeat {
	name := VersionName(deploymentID, versionNum)
	send := func(ctx context.Context) error {
		_, err := cmdctx.WorkspaceClient(ctx).BundleDeployments.Heartbeat(ctx, bundledeployments.HeartbeatRequest{Name: name})
		return err
	}
	return startHeartbeat(ctx, send)
}

func startHeartbeat(ctx context.Context, send func(ctx context.Context) error) *Heartbeat {
	h := &Heartbeat{
		send:    send,
		renewed: make(chan struct{}, 1),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	h.stopOnce = sync.OnceFunc(func() {
		close(h.stop)
		<-h.done
	})
	go h.run(ctx)
	return h
}

// renew records that a successful write updated the version, which the service counts as a
// heartbeat, so the next heartbeat is pushed back a full interval.
func (h *Heartbeat) renew() {
	select {
	case h.renewed <- struct{}{}:
	default:
	}
}

func (h *Heartbeat) run(ctx context.Context) {
	defer close(h.done)

	next := time.NewTimer(heartbeatInterval)
	defer next.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stop:
			return
		case <-h.renewed:
			next.Reset(heartbeatInterval)
		case <-next.C:
			h.beat(ctx)
			next.Reset(heartbeatInterval)
		}
	}
}

// beat sends a heartbeat, retrying quickly a few times before leaving it to the next interval.
func (h *Heartbeat) beat(ctx context.Context) {
	err := h.send(ctx)
	for _, delay := range heartbeatRetryDelays {
		if err == nil || ctx.Err() != nil {
			break
		}
		log.Warnf(ctx, "Failed to send deployment heartbeat, retrying in %s: %s", delay, err)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		case <-h.stop:
			return
		}
		err = h.send(ctx)
	}

	// A cancelled run needs no warning.
	if err != nil && ctx.Err() == nil {
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Failed to send deployment heartbeat, so the deployment may show as interrupted; retrying in %s: %s", heartbeatInterval, err),
		})
	}
}

// Stop ends the heartbeat and waits for it. Safe to call twice.
func (h *Heartbeat) Stop() {
	h.stopOnce()
}
