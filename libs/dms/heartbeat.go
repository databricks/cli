package dms

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// heartbeatInterval is how long a version may go without an update before a heartbeat is sent.
// The UI shows a version as interrupted once its last update is over 2 minutes old.
const heartbeatInterval = 30 * time.Second

// Heartbeat keeps an open deployment version's last update time fresh, which is how the UI tells
// a running deploy from an interrupted one. CreateVersion and every successful write update it
// too, so a heartbeat is sent only after an interval without one. Best effort: a failure can only
// make the deploy show as interrupted, so it is reported and the next interval tries again.
type Heartbeat struct {
	renewed chan struct{}
	cancel  context.CancelFunc
}

// StartHeartbeat starts heartbeating the version the caller just created. CreateVersion counts
// as the first update, so the first heartbeat goes out an interval later.
func StartHeartbeat(ctx context.Context, deploymentID string, versionNum int) *Heartbeat {
	name := VersionName(deploymentID, versionNum)
	return startHeartbeat(ctx, func(ctx context.Context) error {
		_, err := cmdctx.WorkspaceClient(ctx).BundleDeployments.Heartbeat(ctx, bundledeployments.HeartbeatRequest{Name: name})
		return err
	})
}

// startHeartbeat takes the send call so tests can count calls instead of reaching a server.
func startHeartbeat(ctx context.Context, send func(context.Context) error) *Heartbeat {
	ctx, cancel := context.WithCancel(ctx)
	h := &Heartbeat{renewed: make(chan struct{}, 1), cancel: cancel}
	go h.run(ctx, send)
	return h
}

func (h *Heartbeat) run(ctx context.Context, send func(context.Context) error) {
	timer := time.NewTimer(heartbeatInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.renewed:
		case <-timer.C:
			// A request cut short by Stop is not a failure.
			if err := send(ctx); err != nil && ctx.Err() == nil {
				logdiag.LogDiag(ctx, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("Failed to send deployment heartbeat, so the deployment may show as interrupted; retrying in %s: %s", heartbeatInterval, err),
				})
			}
		}
		timer.Reset(heartbeatInterval)
	}
}

// renew records that a successful write updated the version, which the service counts as a
// heartbeat, so the next heartbeat is pushed back a full interval. Non-blocking: one pending
// signal is enough.
func (h *Heartbeat) renew() {
	select {
	case h.renewed <- struct{}{}:
	default:
	}
}

// Stop ends the heartbeat, cutting short one in flight.
func (h *Heartbeat) Stop() {
	h.cancel()
}
