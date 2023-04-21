package ui

import (
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var Interactive = !color.NoColor

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
