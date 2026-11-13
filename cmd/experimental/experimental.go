package experimental

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	aitoolscmd "github.com/databricks/cli/experimental/aitools/cmd"
	postgrescmd "github.com/databricks/cli/experimental/postgres/cmd"
	"github.com/spf13/cobra"
)

// Spec declares the experimental command.
var Spec = root.CommandSpec{
	Use:    "experimental",
	Short:  "Experimental commands that may change in future versions",
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Experimental commands that may change in future versions.

╔══════════════════════════════════════════════════════════════════╗
║  🚧  EXPERIMENTAL: These commands may change in future versions  ║
╚══════════════════════════════════════════════════════════════════╝

These commands provide early access to new features that are still under
development. They may change or be removed in future versions without notice.`

	cmd.AddCommand(newAirCmd())
	cmd.AddCommand(aitoolscmd.NewAitoolsCmd())
	cmd.AddCommand(newGenieCmd())
	cmd.AddCommand(postgrescmd.New())
	cmd.AddCommand(newWorkspaceOpenCommand())
}
