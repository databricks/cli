package sandbox

import (
	"github.com/spf13/cobra"
)

// New returns the root command for the sandbox subcommand.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sandbox",
		Short:   "Manage Databricks Sandbox environments",
		GroupID: "development",
		Hidden:  true,
		Long: `Manage Databricks Sandbox environments.

Sandbox provides SSH-accessible development environments backed by
microVM isolation. Each sandbox is a personal sandbox with pre-installed
tooling (Python, Node.js, Rust, Databricks CLI) and persistent storage.

Getting started:
  databricks auth login --host https://...   # authenticate to a Databricks workspace
  databricks sandbox register                # generate and register an SSH key
  databricks sandbox create                  # provision a sandbox (becomes your default)
  databricks sandbox ssh                     # SSH to your default sandbox

Common workflows:
  databricks sandbox ssh                     # SSH to your default sandbox
  databricks sandbox ssh my-project          # SSH to a named sandbox
  databricks sandbox list                    # list your sandboxes
  databricks sandbox create                  # create a new sandbox
  databricks sandbox delete my-project       # delete a sandbox
  databricks sandbox status my-project       # show sandbox status
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
