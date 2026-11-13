package root

import "github.com/spf13/cobra"

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
