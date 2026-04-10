package lakebox

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lakebox",
		Short: "Manage Databricks Lakebox environments",
		Long: `Manage Databricks Lakebox environments.

Lakebox provides SSH-accessible development environments backed by
microVM isolation. Each lakebox is a personal sandbox with pre-installed
tooling (Python, Node.js, Rust, Databricks CLI) and persistent storage.

Common workflows:
  databricks lakebox login                        # authenticate to Databricks
  databricks lakebox ssh                          # SSH to your default lakebox
  databricks lakebox ssh my-project               # SSH to a named lakebox
  databricks lakebox list                         # list your lakeboxes
  databricks lakebox create --name my-project     # create a new lakebox
  databricks lakebox delete my-project            # delete a lakebox
  databricks lakebox status                       # show current lakebox status

The CLI manages your ~/.ssh/config so you can also connect directly:
  ssh my-project                                  # after 'lakebox ssh --setup'
`,
	}

	cmd.AddCommand(newLoginCommand())
	cmd.AddCommand(newSSHCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newSetDefaultCommand())

	return cmd
}
