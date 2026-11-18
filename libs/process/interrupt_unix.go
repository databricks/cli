//go:build unix

package process

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// BackgroundInGroup runs args as Background does, with WithProcessGroup, so a
// cancelled ctx stops the child's descendants too.
//
// The new process group takes the child out of the terminal's foreground
// group, so Ctrl-C no longer reaches it. While the child runs, the first SIGINT
// or SIGTERM is caught instead: it stops the group, and is then raised again
// in this process, which ends as it would have without the group. The default
// disposition is restored before the group is stopped, so a second signal ends
// this process at once.
func BackgroundInGroup(ctx context.Context, args []string, opts ...execOption) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	caught := make(chan os.Signal, 1)
	done := make(chan struct{})
	go func() {
		defer close(caught)
		select {
		case sig := <-sigCh:
			signal.Stop(sigCh)
			caught <- sig
			cancel()
		case <-done:
		}
	}()

	out, err := Background(ctx, args, append(opts, WithProcessGroup())...)
	close(done)
	signal.Stop(sigCh)
	if sig, ok := <-caught; ok {
		_ = syscall.Kill(os.Getpid(), sig.(syscall.Signal))
	}
	return out, err
}
