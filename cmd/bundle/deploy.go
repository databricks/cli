package bundle

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/spf13/cobra"
)

func newDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy bundle",
		Args:  root.NoArgs,
	}

	var force bool
	var forceLock bool
	var failOnActiveRuns bool
	var computeID string
	var autoApprove bool
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation.")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")
	cmd.Flags().BoolVar(&failOnActiveRuns, "fail-on-active-runs", false, "Fail if there are running jobs or pipelines in the deployment.")
	cmd.Flags().StringVarP(&computeID, "compute-id", "c", "", "Override compute in the deployment with the given compute ID.")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals that might be required for deployment.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)

		if !diags.HasError() {
			bundle.ApplyFunc(ctx, b, func(context.Context, *bundle.Bundle) diag.Diagnostics {
				b.Config.Bundle.Force = force
				b.Config.Bundle.Deployment.Lock.Force = forceLock
				b.AutoApprove = autoApprove

				if cmd.Flag("compute-id").Changed {
					b.Config.Bundle.ComputeID = computeID
				}
				if cmd.Flag("fail-on-active-runs").Changed {
					b.Config.Bundle.Deployment.FailOnActiveRuns = failOnActiveRuns
				}

				return nil
			})

			diags = diags.Extend(
				bundle.Apply(ctx, b, bundle.Seq(
					phases.Initialize(),
					phases.Build(),
					phases.Deploy(),
				)),
			)
		}

		renderOpts := render.RenderOptions{RenderSummaryTable: false}
		err := render.RenderTextOutput(cmd.OutOrStdout(), b, diags, renderOpts)
		if err != nil {
			return fmt.Errorf("failed to render output: %w", err)
		}

		if diags.HasError() {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
}
