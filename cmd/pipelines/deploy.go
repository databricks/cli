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
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
	"github.com/spf13/cobra"
)

func formatOSSTemplateWarningMessage(d diag.Diagnostic) string {
	fileName := "a pipeline YAML"
	if len(d.Locations) > 0 && d.Locations[0].File != "" {
		fileName = d.Locations[0].File
	}

	return fmt.Sprintf(`Detected %s file is formatted for OSS Spark pipelines.
The "definitions" field is not supported in the Pipelines CLI.
To create a new Pipelines CLI project with a supported pipeline YAML file, run:
  pipelines init`, fileName)
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
		logdiag.SetCollect(ctx, false)

		diags := logdiag.FlushCollected(ctx)
		var ossWarning *diag.Diagnostic

		for _, d := range diags {
			if d.Severity == diag.Warning && strings.Contains(d.Summary, "unknown field: definitions") && len(d.Paths) == 1 && d.Paths[0].Equal(dyn.EmptyPath) {
				ossWarning = &d
			}
			logdiag.LogDiag(ctx, d)
		}

		if ossWarning != nil {
			return errors.New(formatOSSTemplateWarningMessage(*ossWarning))
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
