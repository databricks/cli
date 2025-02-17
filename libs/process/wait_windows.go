//go:build windows

package process

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

func waitForPid(pid int) error {
	handle, err := windows.OpenProcess(
		windows.SYNCHRONIZE|windows.PROCESS_QUERY_INFORMATION,
		false,
		uint32(pid),
	)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return ErrProcessDoesNotExist{Pid: pid}
	}
	if err != nil {
		return fmt.Errorf("OpenProcess failed: %v", err)
	}
	defer windows.CloseHandle(handle)

	// Wait forever for the process to exit. Wait for 5 minutes max.
	ret, err := windows.WaitForSingleObject(handle, uint32(5*time.Minute.Milliseconds()))
	if err != nil {
		return fmt.Errorf("Wait failed: %v", err)
	}

	switch ret {
	case windows.WAIT_OBJECT_0:
		return nil // Process exited
	case 0x00000102:
		// Standard library does not have have a constant defined for this
		// so we use the hex value directly. This is the WAIT_TIMEOUT value.
		// ref: https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-waitforsingleobject#return-value
		return fmt.Errorf("process wait timed out")
	default:
		return fmt.Errorf("unexpected process wait return value: %d", ret)
	}
}
