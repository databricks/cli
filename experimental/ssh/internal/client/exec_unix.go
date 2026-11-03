//go:build !windows

package client

import "syscall"

// execProcess replaces the current process (execve(2)) so the ssh PTY connects
// straight to the agent and its exit code propagates.
func execProcess(argv0 string, argv, env []string) error {
	return syscall.Exec(argv0, argv, env)
}
