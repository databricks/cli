//go:build unix

package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// WithProcessGroup makes the child the leader of a new process group and, when
// the context is cancelled, signals the entire group rather than just the child.
//
// exec.CommandContext's default cancellation only SIGKILLs the direct child, so
// a tool that fans out to its own subprocesses (e.g. `uv sync` spawning Python
// and build backends) leaves those grandchildren running as orphans when the CLI
// receives SIGINT/SIGTERM. Putting the child in its own group and signalling the
// group (negative PID) delivers SIGTERM to every descendant at once, giving them
// a chance to exit cleanly.
//
// Two backstops handle a member that ignores SIGTERM: WaitDelay bounds how long
// Wait blocks on a hung leader (Go then SIGKILLs the leader and closes the pipes
// so Wait returns), and reapProcessGroup sends a final group-wide SIGKILL after
// Wait returns. The second is necessary because Go's WaitDelay escalation targets
// the leader PID only, not the group — without it a grandchild that outlives a
// SIGKILLed leader would be re-orphaned.
func WithProcessGroup() execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		if c.SysProcAttr == nil {
			c.SysProcAttr = &syscall.SysProcAttr{}
		}
		c.SysProcAttr.Setpgid = true

		c.WaitDelay = processGroupGracePeriod
		c.Cancel = func() error {
			// With Setpgid and Pgid unset, the child's group ID equals its PID;
			// a negative PID targets the whole group. Map "no such process" to
			// os.ErrProcessDone so a benign exit/cancel race is not surfaced as a
			// Wait error.
			err := syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
			if errors.Is(err, syscall.ESRCH) {
				return os.ErrProcessDone
			}
			return err
		}
		return nil
	}
}

// reapProcessGroup sends a final SIGKILL to the child's process group after the
// command has been waited on, closing the gap left by Go's WaitDelay escalation
// (which SIGKILLs only the leader PID). It runs only for a WithProcessGroup child
// whose context was cancelled — the escalation path — so a normally-exited
// command is never signalled.
//
// It is safe against PID reuse: this runs synchronously after Wait has returned
// (the leader is reaped), not on a delayed timer. The group ID stays reserved by
// the kernel while any member is alive, so kill(-pgid) hits surviving descendants
// or returns ESRCH on an already-empty group; there is no 10s window in which the
// PGID could be reused by an unrelated group before the signal is sent.
func reapProcessGroup(ctx context.Context, c *exec.Cmd) {
	if c.SysProcAttr == nil || !c.SysProcAttr.Setpgid {
		return
	}
	if c.Process == nil || ctx.Err() == nil {
		return
	}
	_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
}
