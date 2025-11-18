package experimental

import (
	"github.com/databricks/cli/experimental/aitools"
	mcp "github.com/databricks/cli/experimental/apps-mcp/cmd"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "experimental",
		Short:  "Experimental commands that may change in future versions",
		Hidden: true,
		Long: `Experimental commands that may change in future versions.

╔════════════════════════════════════════════════════════════════╗
║  ⚠️  EXPERIMENTAL: These commands may change in future versions ║
╚════════════════════════════════════════════════════════════════╝

These commands provide early access to new features that are still under
development. They may change or be removed in future versions without notice.`,
	}

	cmd.AddCommand(aitools.New())
	cmd.AddCommand(mcp.NewMcpCmd())

	return cmd
}
