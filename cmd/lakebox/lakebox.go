package lakebox

import (
	"github.com/databricks/cli/cmd/root"
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

Getting started:
  lakebox auth login --host https://...   # authenticate to Databricks workspace and lakebox service
  lakebox ssh                             # SSH to your default lakebox

Common workflows:
  lakebox ssh                             # SSH to your default lakebox
  lakebox ssh my-project                  # SSH to a named lakebox
  lakebox list                            # list your lakeboxes
  lakebox create                          # create a new lakebox
  lakebox delete my-project               # delete a lakebox
  lakebox status my-project               # show lakebox status

The CLI manages your ~/.ssh/config so you can also connect directly:
  ssh my-project                          # after 'lakebox ssh'
`,
	}

	cmd.AddCommand(newRegisterCommand())
	cmd.AddCommand(newSetDefaultCommand())
	cmd.AddCommand(newSSHCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newStatusCommand())

	return cmd
}

// mustWorkspaceClient applies the saved last-login profile when the user
// hasn't explicitly set --profile, then delegates to root.MustWorkspaceClient.
func mustWorkspaceClient(cmd *cobra.Command, args []string) error {
	profileFlag := cmd.Flag("profile")
	if profileFlag != nil && !profileFlag.Changed {
		if last := GetLastProfile(); last != "" {
			_ = profileFlag.Value.Set(last)
		}
	}
	return root.MustWorkspaceClient(cmd, args)
}
