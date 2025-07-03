// Copied from cmd/bundle/deploy.go and adapted for pipelines use.
package pipelines

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sync"
	"github.com/spf13/cobra"
)

func Deploy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy pipelines",
		Args:  root.NoArgs,
	}

	var forceLock bool
	var autoApprove bool
	var verbose bool
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals that might be required for deployment.")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	// Verbose flag currently only affects file sync output, it's used by the vscode extension
	cmd.Flags().MarkHidden("verbose")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)

		if !diags.HasError() {
			bundle.ApplyFunc(ctx, b, func(context.Context, *bundle.Bundle) diag.Diagnostics {
				b.Config.Bundle.Deployment.Lock.Force = forceLock
				b.AutoApprove = autoApprove
				return nil
			})

			var outputHandler sync.OutputHandler
			if verbose {
				outputHandler = func(ctx context.Context, c <-chan sync.Event) {
					sync.TextOutput(ctx, c, cmd.OutOrStdout())
				}
			}

			diags = diags.Extend(phases.Initialize(ctx, b))

			if !diags.HasError() {
				diags = diags.Extend(bundle.Apply(ctx, b, validate.FastValidate()))
			}

			if !diags.HasError() {
				diags = diags.Extend(phases.Build(ctx, b))
			}

			if !diags.HasError() {
				diags = diags.Extend(phases.Deploy(ctx, b, outputHandler))
			}
		}

		return RenderDiagnostics(cmd.OutOrStdout(), b, diags)
	}
	return cmd
}
