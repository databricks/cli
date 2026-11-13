package root

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// StdoutCarriesProtocol is the cobra annotation of a command whose stdout carries a protocol, such as MCP JSON-RPC
// messages, so nothing else may be written there. Root refuses a log file on stdout for such a command, and it skips
// the stale-version warning. That warning goes to stderr, but an agent shows the server's stderr each time it starts
// the server, so the warning would repeat in every session.
const StdoutCarriesProtocol = "stdout-carries-protocol"

// carriesProtocol reports whether cmd is annotated with StdoutCarriesProtocol.
func carriesProtocol(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations[StdoutCarriesProtocol]
	return ok
}

// writesToStdout reports whether w is stdout, given the process's stdout and stderr. It compares the open file, not a
// path: on macOS, os.Stat of /dev/stdout does not match stdout when stdout is a pipe, but the opened /dev/stdout does.
// The streams are parameters because under `go test -json` the testing package sets os.Stderr to os.Stdout.
//
// A file that is also stderr is not stdout. When stdout and stderr are one file, as on a terminal, the protocol
// already shares it with the diagnostics, so the check cannot help and the log file is allowed.
func writesToStdout(w io.Writer, stdout, stderr *os.File) bool {
	f, ok := w.(*os.File)
	if !ok || f == stderr {
		return false
	}
	return sameFile(f, stdout) && !sameFile(f, stderr)
}

// sameFile reports whether a and b are the same open file. A stream that cannot be stat'ed, such as a closed one, is
// not the same file as anything.
func sameFile(a, b *os.File) bool {
	if a == b {
		return true
	}
	ai, err := a.Stat()
	if err != nil {
		return false
	}
	bi, err := b.Stat()
	if err != nil {
		return false
	}
	return sameFileInfo(ai, bi)
}
