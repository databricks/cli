package postgrescmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithStatementTimeout_ZeroIsPassthrough(t *testing.T) {
	parent := t.Context()
	got, cancel := withStatementTimeout(parent, 0)
	defer cancel()
	// Parent and got should compare equal: zero timeout returns the parent
	// unchanged (and a no-op cancel).
	deadline, ok := got.Deadline()
	assert.False(t, ok, "deadline should not be set when timeout is 0")
	assert.True(t, deadline.IsZero())
}

func TestWithStatementTimeout_AppliesDeadline(t *testing.T) {
	parent := t.Context()
	got, cancel := withStatementTimeout(parent, time.Second)
	defer cancel()
	deadline, ok := got.Deadline()
	assert.True(t, ok)
	assert.False(t, deadline.IsZero())
}

func TestReportCancellation_SignalCanceled(t *testing.T) {
	signalCtx, signalCancel := context.WithCancel(t.Context())
	signalCancel()
	stmtCtx := signalCtx
	got := reportCancellation(signalCtx, stmtCtx, errors.New("anything"), 0)
	assert.Equal(t, "Query cancelled.", got)
}

func TestReportCancellation_TimeoutFired(t *testing.T) {
	signalCtx := t.Context()
	stmtCtx, stmtCancel := context.WithDeadline(signalCtx, time.Now().Add(-time.Second))
	defer stmtCancel()
	// Wait for the deadline to be surfaced.
	<-stmtCtx.Done()
	got := reportCancellation(signalCtx, stmtCtx, errors.New("query failed"), 5*time.Second)
	assert.Equal(t, "Query timed out after 5s.", got)
}

func TestReportCancellation_GenericError(t *testing.T) {
	signalCtx := t.Context()
	stmtCtx := signalCtx
	got := reportCancellation(signalCtx, stmtCtx, errors.New("syntax error"), 0)
	assert.Equal(t, "syntax error", got)
}

func TestWatchInterruptSignals_CancelOnStop(t *testing.T) {
	// stop should cancel the parent context as a side-effect so the goroutine
	// terminates promptly. We don't actually send a SIGINT here (it would
	// also kill the test runner); we just verify stop cleans up.
	parent, parentCancel := context.WithCancel(t.Context())
	defer parentCancel()

	cancelled := false
	cancel := func() {
		cancelled = true
		parentCancel()
	}

	stop := watchInterruptSignals(parent, cancel)
	stop()
	assert.True(t, cancelled, "stop should call cancel to wake the goroutine")
}
