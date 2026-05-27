package ssh

import (
	"github.com/spf13/cobra"
)

const disclaimer = `WARNING! This is an experimental feature:
- The product is in preview and not intended to be used in production;
- The product may change or may never be released;
- While we will not charge separately for this product right now, we may charge for it in the future. You will still incur charges for DBUs;
- There's no formal support or SLAs for the preview - so please reach out to your account or other contact with any questions or feedback;`

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh",
		Short:  "Connect to your Databricks compute and workspace via SSH",
		Hidden: true,
		Long: `Connect to your Databricks compute and workspace via SSH.

Common workflows:
  databricks ssh connect --ide=cursor                         # connect to serverless through Cursor
  databricks ssh setup --name=<name> --cluster=<cluster-id>   # update ~/.ssh/config so you can reconnect to a dedicated cluster
  ssh <name>                                                  # connect to dedicated cluster after setup

Use ` + "`databricks ssh connect --help`" + ` to see all available flags.

` + disclaimer,
	}

	cmd.AddCommand(newSetupCommand())
	cmd.AddCommand(newConnectCommand())
	cmd.AddCommand(newServerCommand())

	return cmd
}
