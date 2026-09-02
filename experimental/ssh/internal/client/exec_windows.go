//go:build windows

package client

import "errors"

// execProcess is unsupported on Windows (no syscall.Exec); the shim only runs on the Linux driver.
func execProcess(argv0 string, argv, env []string) error {
	return errors.New("process exec is not supported on Windows")
}
