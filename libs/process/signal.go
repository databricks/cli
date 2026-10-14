package process

import (
	"context"
	"os"
	"os/signal"
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
// The returned stop uninstalls the handler, cancels the context, and waits for
// the handler goroutine to exit. It is safe to call more than once; the caller
// must defer it.
func NotifyContext(parent context.Context, sigs ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sigs...)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Selecting on ctx.Done() too lets the goroutine exit on the normal (no
		// signal) path: signal.Stop unregisters the channel but never closes it.
		select {
		case <-sigCh:
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
		cancel()
	}()

	return ctx, func() {
		signal.Stop(sigCh)
		cancel()
		<-done
	}
}
