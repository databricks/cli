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
  lakebox auth login                              # authenticate to Databricks
  lakebox ssh                                     # SSH to your default lakebox
  lakebox ssh my-project                          # SSH to a named lakebox
  lakebox list                                    # list your lakeboxes
  lakebox create                                  # create a new lakebox
  lakebox delete my-project                       # delete a lakebox
  lakebox status my-project                       # show lakebox status
  lakebox register-key --public-key-file ~/.ssh/id_rsa.pub  # register SSH key

The CLI manages your ~/.ssh/config so you can also connect directly:
  ssh my-project                                  # after 'lakebox ssh'
`,
	}

	cmd.AddCommand(newSSHCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newSetDefaultCommand())
	cmd.AddCommand(newRegisterKeyCommand())

	return cmd
}
