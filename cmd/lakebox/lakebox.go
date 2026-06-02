package lakebox

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lakebox",
		Short:   "Manage Databricks Lakebox environments",
		GroupID: "development",
		Hidden:  true,
		Long: `Manage Databricks Lakebox environments.

Lakebox provides SSH-accessible development environments backed by
microVM isolation. Each lakebox is a personal sandbox with pre-installed
tooling (Python, Node.js, Rust, Databricks CLI) and persistent storage.

Getting started:
  databricks auth login --host https://...   # authenticate to a Databricks workspace
  databricks lakebox register                # generate and register an SSH key
  databricks lakebox create                  # provision a lakebox (becomes your default)
  databricks lakebox ssh                     # SSH to your default lakebox

Common workflows:
  databricks lakebox ssh                     # SSH to your default lakebox
  databricks lakebox ssh my-project          # SSH to a named lakebox
  databricks lakebox list                    # list your lakeboxes
  databricks lakebox create                  # create a new lakebox
  databricks lakebox delete my-project       # delete a lakebox
  databricks lakebox status my-project       # show lakebox status
`,
	}

	cmd.AddCommand(newRegisterCommand())
	cmd.AddCommand(newSSHKeyCommand())
	cmd.AddCommand(newSetDefaultCommand())
	cmd.AddCommand(newSSHCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newStopCommand())
	cmd.AddCommand(newStartCommand())
	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newConfigCommand())

	return cmd
}
