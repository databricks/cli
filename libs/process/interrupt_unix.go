//go:build unix

package process

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// raiseWait bounds how long Raise waits for the signal to end the process.
const raiseWait = time.Second

// Raise ends this process by sig, a signal it caught, so that its parent sees
// it end as it would have had sig not been caught: a shell then stops the
// script, loop or command list that ran it. The caller must have stopped
// catching sig, which restores its default disposition.
//
// The signal ends the process only some time after kill returns, so Raise
// waits for it: a caller that went on could start more work, or exit normally
// before the signal lands. Raise returns only when sig does not end the
// process, because it was ignored when the process started (SIGHUP under
// nohup) or is still caught elsewhere.
func Raise(sig os.Signal) {
	_ = syscall.Kill(os.Getpid(), sig.(syscall.Signal))
	time.Sleep(raiseWait)
}

// BackgroundInGroup runs args as Background does, with WithProcessGroup, so a
// cancelled ctx stops the child's descendants too.
//
// The new process group takes the child out of the terminal's foreground
// group, so neither Ctrl-C nor the terminal closing reaches it. While the
// child runs, the first SIGINT, SIGTERM or SIGHUP is caught instead: it stops
// the group, and is then raised again in this process (see Raise), which
// ends as it would have without the group. The default
// disposition is restored before the group is stopped, so a second signal ends
// this process at once.
//
// The child also leads a new session, so it has no controlling terminal. In a
// background group of the terminal's session, a child that reads the terminal
// once it got SIGTERM (Claude Code does) is stopped by SIGTTIN until the
// SIGKILL at the end of the grace period. Without a controlling terminal its
// open of /dev/tty fails instead.
func BackgroundInGroup(ctx context.Context, args []string, opts ...execOption) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
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

	out, err := Background(ctx, args, append(opts, WithProcessGroup(), withNewSession())...)
	close(done)
	signal.Stop(sigCh)
	if sig, ok := <-caught; ok {
		Raise(sig)
	}
	return out, err
}

// withNewSession makes the child of WithProcessGroup, which it must follow, the
// leader of a new session instead. setpgid fails for a session leader, so it
// clears Setpgid; a session leader's group ID is its PID, so the group signals
// of WithProcessGroup still reach the child and its descendants.
func withNewSession() execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.SysProcAttr.Setpgid = false
		c.SysProcAttr.Setsid = true
		return nil
	}
}
