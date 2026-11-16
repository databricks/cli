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
// MSYS=disable_pcon, or mintty's ConPTY option), not a property of the
// terminal, shell or environment variables like MSYSTEM. It is on by default
// in MSYS2 since 2022 and in Git for Windows since 2.47 (MSYS2 runtime 3.5).
// Older Git for Windows installers offered it as an experimental option that
// was off by default and wrote MSYS=disable_pcon, so the pipe case remains
// common on machines with an older Git for Windows. Terminals that host the
// shell through ConPTY themselves (Windows Terminal, the VS Code terminal)
// always produce the console case.
//
// Hence the check is on the handle itself: see isMSYSPipe. To reproduce the
// pipe case, start mintty with MSYS=disable_pcon in its environment.

// isMSYSPipe reports whether v is an MSYS2/Cygwin pty pipe rather than a real
// console, i.e. the "without ConPTY" case described above. It is always false
// outside Windows, and for anything that is not an *os.File.
func isMSYSPipe(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsCygwinTerminal(fd) && !isatty.IsTerminal(fd) //nolint:forbidigo
}
