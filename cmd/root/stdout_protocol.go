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

// writesToStdout reports whether w is the process's stdout. It compares the open file, not a path: on macOS,
// os.Stat of /dev/stdout does not match stdout when stdout is a pipe, but the opened /dev/stdout does. stderr is
// never stdout here, even when both share a terminal.
func writesToStdout(w io.Writer) (bool, error) {
	f, ok := w.(*os.File)
	if !ok || f == os.Stderr {
		return false, nil
	}
	out, err := os.Stdout.Stat()
	if err != nil {
		return false, err
	}
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	return os.SameFile(fi, out), nil
}
