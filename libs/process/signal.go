package process

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

// NotifyContext is like signal.NotifyContext, but stops relaying as soon as
// the first signal arrives, so a second one is handled as it was before the
// call: for a signal the process doesn't otherwise handle, it terminates.
//
// signal.Notify overrides the previous disposition for as long as the channel
// stays registered, so with signal.NotifyContext a second Ctrl-C is merely
// buffered and dropped, leaving the user no way to abort a step that ignores
// cancellation or a slow teardown. Calling signal.Stop before cancel restores
// the previous disposition first, so by the time the cancellation is
// observable a second signal gets it. Note that Notify also stops ignoring
// SIGHUP or SIGINT if the process started with it ignored (e.g. under nohup);
// signal.Stop ignores it again.
//
// A signal cancels the context with a *SignalError cause, so a caller can tell
// it apart from its own call to stop.
//
// The returned stop uninstalls the handler, waits for the handler goroutine to
// exit, and cancels the context. A signal relayed before stop uninstalls the
// handler is still reported as the cause rather than dropped, so after stop
// returns the caller can check whether one came in without racing a late
// signal. It is safe to call more than once; the caller must defer it.
func NotifyContext(parent context.Context, sigs ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(parent)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sigs...)

	stopping := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Selecting on ctx.Done() and stopping too lets the goroutine exit on the
		// normal (no signal) path: signal.Stop unregisters the channel but never
		// closes it.
		select {
		case sig := <-sigCh:
			signal.Stop(sigCh)
			cancel(&SignalError{Signal: sig})
		case <-ctx.Done():
			signal.Stop(sigCh)
		case <-stopping:
			// stop has already called signal.Stop, so a signal that arrived
			// before then is buffered in sigCh. Report it rather than drop it.
			select {
			case sig := <-sigCh:
				cancel(&SignalError{Signal: sig})
			default:
			}
		}
	}()

	return ctx, sync.OnceFunc(func() {
		signal.Stop(sigCh)
		close(stopping)
		<-done
		cancel(nil)
	})
}

// SignalError is the cause NotifyContext cancels its context with when a
// signal arrives.
type SignalError struct {
	Signal os.Signal
}

func (e *SignalError) Error() string {
	return e.Signal.String() + " signal received"
}
