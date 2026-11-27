package cmdio

import (
	"os"

	"golang.org/x/sys/windows"
)

// inForeground reports whether this process may read the console f. Windows
// consoles have no background jobs, so it always may.
func inForeground(*os.File) bool {
	return true
}

// discardInput discards what was typed at the console f but not read yet.
func discardInput(f *os.File) error {
	return windows.FlushConsoleInputBuffer(windows.Handle(f.Fd()))
}
