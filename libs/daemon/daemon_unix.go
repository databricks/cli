//go:build linux || darwin

package daemon

import "syscall"

// References:
// 1. linux: https://go.dev/src/syscall/exec_linux.go
// 2. macos (arm): https://go.dev/src/syscall/exec_libc2.go
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// Create a new session for the child process. This ensures that the daemon
		// is not terminated when the parent session is closed. This can happen
		// for example when a ssh session is terminated.
		Setsid: true,
	}
}
