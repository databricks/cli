package cmdio

import (
	"os"

	"github.com/mattn/go-isatty"
)

// Native Windows programs started from an MSYS2/Cygwin terminal (mintty and
// the shells built on it) see one of two kinds of standard handles, depending
// on whether the MSYS2 runtime uses ConPTY, the Windows pseudo console:
//
//   - With ConPTY, the handles are a real Windows console. isatty.IsTerminal
//     is true, and everything works as in conhost or Windows Terminal,
//     including bubbletea prompts and pagers.
//
//   - Without ConPTY, the handles are named pipes that the MSYS2 runtime uses
//     to emulate a pty (\\.\pipe\msys-<id>-pty<N>-{from,to}-master).
//     isatty.IsTerminal is false and isatty.IsCygwinTerminal is true. Output
//     is still rendered by a terminal, so colors, spinners and line-based
//     input work. Reading raw keystrokes does not: bubbletea's input reader
//     calls GetConsoleMode on stdin and fails with
//     "get console mode: The handle is invalid". Prompts and pagers must not
//     run there.
//
// Whether ConPTY is used is a runtime setting (MSYS=enable_pcon or
// MSYS=disable_pcon, or mintty's ConPTY option), not something MSYSTEM or the
// shell reveals. It is on by default since Git for Windows 2.47; older
// installers wrote MSYS=disable_pcon by default. It only matters in mintty: a
// shell running directly in a console (conhost, Windows Terminal, the VS Code
// terminal) passes that console on to native programs.
//
// Hence the check is on the handle itself: see isMSYSPipe. To reproduce the
// pipe case, start mintty with MSYS=disable_pcon in its environment.
//
// References:
//   - https://github.com/mintty/mintty/wiki/Tips#inputoutput-interaction-with-alien-programs
//   - https://github.com/git-for-windows/build-extra/blob/main/installer/release.sh
//   - https://github.com/git-for-windows/build-extra/blob/main/ReleaseNotes.md
//   - https://github.com/mattn/go-isatty/blob/master/isatty_windows.go
//   - https://code.visualstudio.com/docs/terminal/advanced#_windows-and-conpty

// isMSYSPipe reports whether v is an MSYS2/Cygwin pty pipe rather than a real
// console, i.e. the "without ConPTY" case described above. It is always false
// outside Windows, and for anything that is not an *os.File.
func isMSYSPipe(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	// A Cygwin pty is a named pipe, so this never matches a console handle.
	return isatty.IsCygwinTerminal(fd) //nolint:forbidigo
}
