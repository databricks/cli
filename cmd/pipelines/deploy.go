// Copied from cmd/bundle/deploy.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	libsutils "github.com/databricks/cli/libs/utils"
	"github.com/spf13/cobra"
)

func deployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy pipelines",
		Long:  `Deploy pipelines by uploading all files defined in the project to the target workspace, and creating or updating the pipelines defined in the workspace.`,
		Args:  root.NoArgs,
	}

	var forceLock bool
	var failOnActiveRuns bool
	var autoApprove bool
	var verbose bool
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")
	cmd.Flags().BoolVar(&failOnActiveRuns, "fail-on-active-runs", false, "Fail if there are running pipelines in the deployment.")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals that might be required for deployment.")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	// Verbose flag currently only affects file sync output, it's used by the vscode extension
	cmd.Flags().MarkHidden("verbose")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			InitFunc: func(b *bundle.Bundle) {
				b.Config.Bundle.Deployment.Lock.Force = forceLock
				b.AutoApprove = autoApprove

				if cmd.Flag("fail-on-active-runs").Changed {
					b.Config.Bundle.Deployment.FailOnActiveRuns = failOnActiveRuns
				}
			},
			Verbose:         verbose,
			SkipInitContext: false,
			AlwaysPull:      true,
			FastValidate:    true,
			Build:           true,
			Deploy:          true,
			IsPipelinesCLI:  true,
		})
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		bundle.ApplyContext(ctx, b, mutator.InitializeURLs())
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		for _, group := range b.Config.Resources.AllResources() {
			for _, resourceKey := range libsutils.SortedKeys(group.Resources) {
				resource := group.Resources[resourceKey]
				cmdio.LogString(ctx, fmt.Sprintf("View your %s %s here: %s", resource.ResourceDescription().SingularName, resourceKey, resource.GetURL()))
			}
		}

		return nil
	}
	return cmd
}
