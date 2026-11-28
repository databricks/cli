package cmdio

import (
	"os"

	"golang.org/x/sys/windows"
)

// discardInput discards what was typed at the console f but not read yet.
func discardInput(f *os.File) error {
	return windows.FlushConsoleInputBuffer(windows.Handle(f.Fd()))
}
