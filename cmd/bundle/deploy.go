// Copied to cmd/pipelines/deploy.go and adapted for pipelines use.
// Consider if changes made here should be made to the pipelines counterpart as well.
package bundle

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy bundle",
		Long: `Deploy bundle.

Common patterns:
  databricks bundle deploy                  # Deploy to default target (dev)
  databricks bundle deploy --target dev     # Deploy to development
  databricks bundle deploy --target prod    # Deploy to production

See https://docs.databricks.com/en/dev-tools/bundles/index.html for more information.`,
		Args: root.NoArgs,
	}

	var force bool
	var forceLock bool
	var failOnActiveRuns bool
	var clusterId string
	var autoApprove bool
	var verbose bool
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation.")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")
	cmd.Flags().BoolVar(&failOnActiveRuns, "fail-on-active-runs", false, "Fail if there are running jobs or pipelines in the deployment.")
	cmd.Flags().StringVar(&clusterId, "compute-id", "", "Override cluster in the deployment with the given compute ID.")
	cmd.Flags().StringVarP(&clusterId, "cluster-id", "c", "", "Override cluster in the deployment with the given cluster ID.")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals that might be required for deployment.")
	cmd.Flags().MarkDeprecated("compute-id", "use --cluster-id instead")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	// Verbose flag currently only affects file sync output, it's used by the vscode extension
	cmd.Flags().MarkHidden("verbose")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			InitFunc: func(b *bundle.Bundle) {
				b.Config.Bundle.Force = force
				b.Config.Bundle.Deployment.Lock.Force = forceLock
				b.AutoApprove = autoApprove

				if cmd.Flag("compute-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}

				if cmd.Flag("cluster-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}
				if cmd.Flag("fail-on-active-runs").Changed {
					b.Config.Bundle.Deployment.FailOnActiveRuns = failOnActiveRuns
				}
			},
			Verbose:         verbose,
			AlwaysPull:      true,
			FastValidate:    true,
			Build:           true,
			PreDeployChecks: true,
			Deploy:          true,
		})

		return err
	}

	return cmd
}
