//go:build windows

package agentconfig

import (
	"errors"

	"golang.org/x/sys/windows"
)

// fileInUse reports whether opening a file failed because another process has it open without sharing, as during a
// rename over it.
func fileInUse(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
