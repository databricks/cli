//go:build !unix

package process

import (
	"context"
	"os/exec"
)

// WithProcessGroup sets a WaitDelay so a cancelled command does not block
// indefinitely on a stuck child.
//
// Unlike the Unix build, this does not reap the child's descendants: killing a
// whole process tree on Windows requires a Job Object (CreateJobObject +
// AssignProcessToJobObject with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE), which is
// out of scope here. The direct child is still terminated by the default
// cancellation, and the caller's signal handler still fires; only grandchildren
// spawned by the child may outlive it.
func WithProcessGroup() execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.WaitDelay = processGroupGracePeriod
		return nil
	}
}

// reapProcessGroup is a no-op on non-unix builds: there is no process group to
// sweep (WithProcessGroup does not set Setpgid here). See the unix build for the
// group-SIGKILL escalation this closes.
func reapProcessGroup(_ context.Context, _ *exec.Cmd) {}
