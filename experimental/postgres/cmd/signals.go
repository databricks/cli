package postgrescmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// watchInterruptSignals installs handlers for SIGINT and SIGTERM that call
// cancel when the user hits Ctrl+C or the process gets a SIGTERM.
//
// Returns a stop function that uninstalls the handlers; the caller must defer
// it. Calling stop drains the signal channel so a queued signal that arrived
// during shutdown does not leak.
//
// On Windows, Go maps Ctrl+C to os.Interrupt via the console-control-handler.
// The same code path therefore works for the Windows runner; the integration
// test pins this expectation.
func watchInterruptSignals(ctx context.Context, cancel context.CancelFunc) func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
		close(done)
	}()

	return func() {
		signal.Stop(sigCh)
		// Wake the goroutine in case neither sigCh nor ctx.Done has fired.
		cancel()
		<-done
	}
}
