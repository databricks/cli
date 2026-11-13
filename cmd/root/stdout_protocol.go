package root

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// StdoutCarriesProtocol is the cobra annotation of a command whose stdout carries a protocol, such as MCP JSON-RPC
// messages. Root refuses a log file on stdout for such a command and skips the stale-version warning, which an agent
// would show in every session.
const StdoutCarriesProtocol = "stdout-carries-protocol"

// carriesProtocol reports whether cmd is annotated with StdoutCarriesProtocol.
func carriesProtocol(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations[StdoutCarriesProtocol]
	return ok
}

// writesToStdout reports whether w is the open stdout file. It compares files, not paths: on macOS a stat of
// /dev/stdout does not match a stdout pipe, but the opened /dev/stdout does. The streams are parameters so tests can
// pass their own, since under `go test -json` os.Stderr is os.Stdout.
//
// A file that is also stderr is not stdout, because one terminal file carries both diagnostics and protocol.
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
