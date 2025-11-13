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
		Short:  "Connect to Databricks compute with ssh",
		Hidden: true,
		Long: `Connect to Databricks compute with ssh.

SSH commands let you setup and establish ssh connections to Databricks compute.

Common workflows:
  databricks experimental ssh connect --cluster=<cluster-id> --profile=<profile-name>  # connect to a cluster without any setup
  databricks experimental ssh setup --name=my-compute --cluster=<cluster-id>           # update local ssh config
  ssh my-compute                                                                       # connect to the compute using ssh client

` + disclaimer,
	}

	cmd.AddCommand(newSetupCommand())
	cmd.AddCommand(newConnectCommand())
	cmd.AddCommand(newServerCommand())

	return cmd
}
