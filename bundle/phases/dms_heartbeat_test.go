package phases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentHeartbeatRenewsAndStops(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	ticks := make(chan time.Time)
	called := make(chan time.Time, 2)
	runCtx, stop := keepDeploymentAlive(ctx, ticks, func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("heartbeat has no deadline")
		}
		called <- deadline
		return nil
	})
	defer stop()
	for range 2 {
		ticks <- time.Now()
		select {
		case deadline := <-called:
			assert.WithinDuration(t, time.Now().Add(15*time.Second), deadline, time.Second)
		case <-time.After(5 * time.Second):
			t.Fatal("heartbeat did not run")
		}
	}
	require.NoError(t, runCtx.Err())
	stop()
	assert.ErrorIs(t, runCtx.Err(), context.Canceled)
	assert.False(t, logdiag.HasError(ctx))
}

func TestDeploymentHeartbeatFailureCancelsWork(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ticks := make(chan time.Time, 1)
	runCtx, stop := keepDeploymentAlive(ctx, ticks, func(context.Context) error {
		return errors.New("lease lost")
	})
	defer stop()
	ticks <- time.Now()
	select {
	case <-runCtx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("work was not canceled after losing the lease")
	}
	stop()
	assert.Contains(t, logdiag.GetFirstErrorSummary(ctx), "lease lost")
}

func TestDeploymentHeartbeatStopCancelsInflightRequest(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	ticks := make(chan time.Time, 1)
	started := make(chan struct{})
	_, stop := keepDeploymentAlive(ctx, ticks, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	defer stop()
	ticks <- time.Now()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("heartbeat did not start")
	}
	stop()
	assert.False(t, logdiag.HasError(ctx), "normal shutdown must not fail a successful deploy")
}
