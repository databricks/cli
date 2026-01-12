//go:build windows

package internal

import (
	"time"

	"golang.org/x/sys/windows"
)

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
		if err != nil {
			return true
		}
		var exitCode uint32
		err = windows.GetExitCodeProcess(handle, &exitCode)
		windows.CloseHandle(handle)
		if err != nil || exitCode != uint32(windows.STATUS_PENDING) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
