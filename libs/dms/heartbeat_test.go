package dms

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Timing is not reachable from acceptance tests, so the heartbeat is tested here on synctest's
// fake clock, against a send that counts calls instead of the service.

type fakeSend struct {
	calls atomic.Int32
	err   error
}

func (f *fakeSend) send(context.Context) error {
	f.calls.Add(1)
	return f.err
}

func heartbeatContext(t *testing.T) context.Context {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	return ctx
}

func TestHeartbeatBeatsWhileIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeSend{}
		h := startHeartbeat(heartbeatContext(t), f.send)
		defer h.Stop()

		// CreateVersion counts as the first update, so nothing goes out before the interval.
		time.Sleep(heartbeatInterval - time.Second)
		synctest.Wait()
		assert.Equal(t, int32(0), f.calls.Load())

		time.Sleep(2 * time.Second)
		synctest.Wait()
		assert.Equal(t, int32(1), f.calls.Load())

		time.Sleep(heartbeatInterval)
		synctest.Wait()
		assert.Equal(t, int32(2), f.calls.Load())
	})
}

func TestHeartbeatSkippedAfterWrite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeSend{}
		h := startHeartbeat(heartbeatContext(t), f.send)
		defer h.Stop()

		time.Sleep(20 * time.Second)
		h.renew()

		// The write counts as an update, so the beat due at 30s moves to 50s.
		time.Sleep(25 * time.Second)
		synctest.Wait()
		assert.Equal(t, int32(0), f.calls.Load())

		time.Sleep(10 * time.Second)
		synctest.Wait()
		assert.Equal(t, int32(1), f.calls.Load())
	})
}

func TestHeartbeatRetriesThenWaitsForInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := heartbeatContext(t)
		f := &fakeSend{err: errors.New("connection refused")}
		h := startHeartbeat(ctx, f.send)
		defer h.Stop()

		// The beat at 30s and its retries 1s, 2s and 4s apart.
		time.Sleep(heartbeatInterval + 8*time.Second)
		synctest.Wait()
		assert.Equal(t, int32(1+len(heartbeatRetryDelays)), f.calls.Load())

		diags := logdiag.FlushCollected(ctx)
		require.Len(t, diags, 1)
		assert.Equal(t, "Failed to send deployment heartbeat, so the deployment may show as interrupted; retrying in 30s: connection refused", diags[0].Summary)

		// Then nothing more until a full interval has passed.
		time.Sleep(heartbeatInterval - 2*time.Second)
		synctest.Wait()
		assert.Equal(t, int32(1+len(heartbeatRetryDelays)), f.calls.Load())

		// Checked before its first retry, 1s later.
		time.Sleep(1500 * time.Millisecond)
		synctest.Wait()
		assert.Equal(t, int32(2+len(heartbeatRetryDelays)), f.calls.Load())
	})
}

func TestHeartbeatStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeSend{}
		h := startHeartbeat(heartbeatContext(t), f.send)

		h.Stop()
		h.Stop()
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		assert.Equal(t, int32(0), f.calls.Load())
	})
}
