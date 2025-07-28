// Copied from cmd/bundle/deploy.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
	"github.com/spf13/cobra"
)

func formatOSSTemplateWarningMessage(d diag.Diagnostic) string {
	fileName := "a pipeline YAML file"
	if len(d.Locations) > 0 && d.Locations[0].File != "" {
		fileName = d.Locations[0].File
	}

	return fmt.Sprintf(`Detected %s is formatted for OSS Spark pipelines.
The "definitions" field is not supported in the Pipelines CLI.

Use the Databricks Lakeflow Declarative Pipelines format instead.
For more information, see: https://docs.databricks.com/aws/en/dlt

Example of a Pipelines CLI supported format for a pipeline YAML file:
resources:
  pipelines:
    my_project_pipeline:
      name: my_project_pipeline
      serverless: true
      catalog: ${var.catalog}
      schema: ${var.schema}
      root_path: "."
      libraries:
        - glob:
            include: transformations/**`, fileName)
}

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
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		// Enable collection of diagnostics to check for OSS template warning in ConfigureBundleWithVariables
		logdiag.SetCollect(ctx, true)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		diags := logdiag.FlushCollected(ctx)
		for _, d := range diags {
			if d.Severity == diag.Warning && strings.Contains(d.Summary, "unknown field: definitions") {
				return errors.New(formatOSSTemplateWarningMessage(d))
			}
		}

		bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) {
			b.Config.Bundle.Deployment.Lock.Force = forceLock
			b.AutoApprove = autoApprove

			if cmd.Flag("fail-on-active-runs").Changed {
				b.Config.Bundle.Deployment.FailOnActiveRuns = failOnActiveRuns
			}
		})

		var outputHandler sync.OutputHandler
		if verbose {
			outputHandler = func(ctx context.Context, c <-chan sync.Event) {
				sync.TextOutput(ctx, c, cmd.OutOrStdout())
			}
		}

		phases.Initialize(ctx, b)

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		bundle.ApplyContext(ctx, b, validate.FastValidate())

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.Build(ctx, b)

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.Deploy(ctx, b, outputHandler)

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}
	return cmd
}
