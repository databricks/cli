//go:build !linux

package server

import "syscall"

// sshdSysProcAttr is a no-op on platforms without Linux parent-death signaling.
func sshdSysProcAttr() *syscall.SysProcAttr {
	return nil
}
