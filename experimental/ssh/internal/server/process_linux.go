//go:build linux

package server

import "syscall"

// sshdSysProcAttr terminates sshd in the kernel when the SSH server exits.
func sshdSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}
