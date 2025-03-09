package bundle

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/render"
	"github.com/databricks/cli/clis"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sync"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDeployCommand(cliType clis.CLIType) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy to a workspace",
		Args:  root.NoArgs,
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
	if cliType == clis.DLT {
		cmd.Flags().MarkHidden("compute-id")
	}
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	cmd.Flags().MarkHidden("verbose")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)

		if !diags.HasError() {
			bundle.ApplyFunc(ctx, b, func(context.Context, *bundle.Bundle) diag.Diagnostics {
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

				return nil
			})

			var outputHandler sync.OutputHandler
			if verbose {
				outputHandler = func(ctx context.Context, c <-chan sync.Event) {
					sync.TextOutput(ctx, c, cmd.OutOrStdout())
				}
			}
			if cliType == clis.DLT {
				// DLT CLI has very minimalistic output
				cmdio.LogString(ctx, fmt.Sprintf("Deploying to target '%s'...", b.Config.Bundle.Target))
			}

			diags = diags.Extend(phases.Initialize(ctx, b))

			if !diags.HasError() {
				diags = diags.Extend(bundle.Apply(ctx, b, validate.FastValidate()))
			}

			if !diags.HasError() {
				diags = diags.Extend(phases.Build(ctx, b))
			}

			if !diags.HasError() {
				diags = diags.Extend(phases.Deploy(ctx, b, outputHandler, cliType))
			}

			if cliType == clis.DLT {
				if len(b.Config.Resources.Pipelines) == 1 {
					diags = diags.Extend(bundle.ApplySeq(ctx, b, mutator.InitializeURLs()))
					for _, pipeline := range b.Config.Resources.Pipelines {
						fmt.Println("Deployed to " + color.CyanString(pipeline.URL))
						break
					}
				} else {
					fmt.Println("TIP: Use the 'dlt.run' command to see all deployed resources.")
				}
			}
		}

		renderOpts := render.RenderOptions{RenderSummaryTable: false}
		err := render.RenderDiagnostics(cmd.OutOrStdout(), b, diags, renderOpts)
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
