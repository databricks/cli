package bundle

import (
	"github.com/databricks/cli/cmd/bundle/debug"
	"github.com/spf13/cobra"
)

func newDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug information about bundles",
		Long:  "Debug information about bundles",
		// This command group is currently intended for the Databricks VSCode extension only
		Hidden: true,
	}
	cmd.AddCommand(debug.NewTerraformCommand())
	return cmd
}
