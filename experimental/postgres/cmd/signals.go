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
// Returns a stop-and-cancel function that uninstalls the handlers (signal.Stop
// prevents future OS deliveries) and cancels the parent context so the
// goroutine wakes promptly. The caller must defer it. The channel is
// 1-buffered and GC'd on return; no explicit drain is needed.
//
// On Windows, Go maps Ctrl+C to os.Interrupt via the console-control-handler,
// so the same code path covers Windows.
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
