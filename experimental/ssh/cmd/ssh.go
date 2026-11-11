package ssh

import (
	"context"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// Spec declares the ssh command.
var Spec = root.CommandSpec{
	Use:    "ssh",
	Short:  "Connect to your Databricks compute and workspace via SSH",
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Connect to your Databricks compute and workspace via SSH.

Common workflows:
  databricks ssh connect --ide=cursor                         # connect to serverless through Cursor
  databricks ssh setup --name=<name> --cluster=<cluster-id>   # update ~/.ssh/config so you can reconnect to a dedicated cluster
  ssh <name>                                                  # connect to dedicated cluster after setup

Use ` + "`databricks ssh connect --help`" + ` to see all available flags.`

	cmd.AddCommand(newSetupCommand())
	cmd.AddCommand(newConnectCommand())
	cmd.AddCommand(newServerCommand())
	cmd.AddCommand(newAgentShimCommand())
}
