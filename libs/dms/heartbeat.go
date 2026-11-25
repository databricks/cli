package dms

import (
	"context"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// heartbeatInterval is the maximum time between consecutive updates.
const heartbeatInterval = 30 * time.Second

// HeartbeatIntervalMsVariable overrides heartbeatInterval in milliseconds. Acceptance tests that
// record requests set it high so a slow cloud deploy does not add a heartbeat at a random point.
const HeartbeatIntervalMsVariable = "DATABRICKS_BUNDLE_DMS_HEARTBEAT_INTERVAL_MS"

// interval returns heartbeatInterval, or the override if it is set to a positive number.
func interval(ctx context.Context) time.Duration {
	ms, err := strconv.Atoi(env.Get(ctx, HeartbeatIntervalMsVariable))
	if err != nil || ms <= 0 {
		return heartbeatInterval
	}
	return time.Duration(ms) * time.Millisecond
}

// Heartbeat updates the update_time of an open deployment version. This is also updated by
// CreateVersion and every successful write.
type Heartbeat struct {
	renewed chan struct{}
	cancel  context.CancelFunc
}

// startVersionHeartbeat starts the heartbeat. We count CreateVersion as the first heartbeat.
func startVersionHeartbeat(ctx context.Context, deploymentID string, versionNum int) *Heartbeat {
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
	go h.run(ctx, send, interval(ctx))
	return h
}

func (h *Heartbeat) run(ctx context.Context, send func(context.Context) error, interval time.Duration) {
	timer := time.NewTimer(interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.renewed:
		case <-timer.C:
			err := send(ctx)
			if err == nil {
				break
			}
			// We only get an error if Stop was called
			if ctx.Err() != nil {
				return
			}
			log.Infof(ctx, "Failed to send deployment heartbeat, so the deployment may show as interrupted; retrying in %s: %s", interval, err)
		}
		timer.Reset(interval)
	}
}

// renew records that a write started, so the next heartbeat waits a full interval.
func (h *Heartbeat) renew() {
	select {
	case h.renewed <- struct{}{}:
	default:
	}
}

// Stop ends the heartbeat.
func (h *Heartbeat) Stop() {
	h.cancel()
}
