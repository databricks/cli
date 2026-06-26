package experimental

import (
	accessexplaincmd "github.com/databricks/cli/experimental/accessexplain/cmd"
	aitoolscmd "github.com/databricks/cli/experimental/aitools/cmd"
	authdoctorcmd "github.com/databricks/cli/experimental/authdoctor/cmd"
	geniecmd "github.com/databricks/cli/experimental/genie/cmd"
	postgrescmd "github.com/databricks/cli/experimental/postgres/cmd"
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

	cmd.AddCommand(accessexplaincmd.New())
	cmd.AddCommand(aitoolscmd.NewAitoolsCmd())
	cmd.AddCommand(authdoctorcmd.New())
	cmd.AddCommand(geniecmd.NewGenieCmd())
	cmd.AddCommand(postgrescmd.New())
	cmd.AddCommand(newWorkspaceOpenCommand())

	return cmd
}
