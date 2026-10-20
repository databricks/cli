package process

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNotifyContextStopsWithoutSignal covers the no-signal path: stop() must
// return (joining its goroutine) rather than leaving it parked on the channel
// for the life of the process, and must be safe to call again, as a caller that
// stops early and also defers stop does.
func TestNotifyContextStopsWithoutSignal(t *testing.T) {
	ctx, stop := NotifyContext(t.Context(), os.Interrupt)

	returned := make(chan struct{})
	go func() {
		stop()
		stop()
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("stop() blocked: the signal goroutine was never joined")
	}
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	var sigErr *SignalError
	assert.NotErrorAs(t, context.Cause(ctx), &sigErr, "stop must not look like a signal")
}

// TestNotifyContextParentCancel checks that cancelling the parent cancels the
// returned context and that stop still returns afterwards.
func TestNotifyContextParentCancel(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	ctx, stop := NotifyContext(parent, os.Interrupt)
	cancel()
	<-ctx.Done()
	stop()
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}
