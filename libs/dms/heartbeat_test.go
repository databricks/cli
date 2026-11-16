package dms

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestHeartbeatBeatsWhileIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeSend{}
		h := startHeartbeat(t.Context(), f.send)
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
		h := startHeartbeat(t.Context(), f.send)
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

func TestHeartbeatFailureRetriesNextInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeSend{err: errors.New("connection refused")}
		h := startHeartbeat(t.Context(), f.send)
		defer h.Stop()

		time.Sleep(heartbeatInterval + time.Second)
		synctest.Wait()
		assert.Equal(t, int32(1), f.calls.Load())

		time.Sleep(heartbeatInterval)
		synctest.Wait()
		assert.Equal(t, int32(2), f.calls.Load())
	})
}

func TestHeartbeatStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &fakeSend{}
		h := startHeartbeat(t.Context(), f.send)

		h.Stop()
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		assert.Equal(t, int32(0), f.calls.Load())
	})
}
