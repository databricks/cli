package ucm

import (
	"github.com/databricks/cli/cmd/ucm/debug"
	"github.com/spf13/cobra"
)

// newDebugCommand returns the `ucm debug` group. Delegates to the
// cmd/ucm/debug subpackage so each debug subcommand lives in its own file.
func newDebugCommand() *cobra.Command {
	return debug.New()
}
