package autodeploy

import (
	"github.com/spf13/cobra"
)

func NewAutoDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-deploy",
		Short: "Deploy a bundle target on every push to a Git branch",
		Long: `Set up and inspect auto-deploy for a bundle target.

Auto-deploy deploys a bundle target whenever commits are pushed to a branch of
the bundle's Git repository. Run these commands from the bundle directory. The
trigger is identified by the bundle's origin URL, its path in the repository
and the target (-t).

Example:
  databricks bundle auto-deploy create -t prod --branch main`,

		// Hidden until the git-webhooks API is public.
		Hidden: true,
	}

	cmd.AddCommand(newCreateCommand())
	return cmd
}
