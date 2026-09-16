package phases

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

func startDeploymentHeartbeat(ctx context.Context, b *bundle.Bundle) (context.Context, func()) {
	db := &b.DeploymentBundle.StateDB
	name := fmt.Sprintf("deployments/%s/versions/%d", db.DeploymentID, db.VersionID)
	heartbeat := func(ctx context.Context) error {
		_, err := db.DmsClient().Service.Heartbeat(ctx, bundledeployments.HeartbeatRequest{Name: name})
		return err
	}
	ticker := time.NewTicker(30 * time.Second)
	runCtx, stop := keepDeploymentAlive(ctx, ticker.C, heartbeat)
	return runCtx, func() {
		ticker.Stop()
		stop()
	}
}

// A failed renewal cancels uploads/apply rather than continuing without a lease.
// Bound each RPC well below the two-minute lease, including SDK retries.
func keepDeploymentAlive(ctx context.Context, ticks <-chan time.Time, heartbeat func(context.Context) error) (context.Context, func()) {
	runCtx, cancelRun := context.WithCancel(ctx)
	heartbeatCtx, cancelHeartbeat := context.WithCancel(runCtx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticks:
				rpcCtx, cancel := context.WithTimeout(heartbeatCtx, 15*time.Second)
				err := heartbeat(rpcCtx)
				cancel()
				if heartbeatCtx.Err() != nil {
					return
				}
				if err != nil {
					logdiag.LogError(ctx, fmt.Errorf("renewing deployment version lease: %w", err))
					cancelRun()
					return
				}
			}
		}
	}()
	return runCtx, func() {
		cancelHeartbeat()
		<-done
		cancelRun()
	}
}
