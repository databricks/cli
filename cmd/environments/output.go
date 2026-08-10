package environments

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/spf13/cobra"
)

// renderResult renders the pipeline result to the command's output.
// In JSON mode it renders the full structured result (even on error).
// In text mode it prints phase headers and a summary, then returns the error.
//
// res is always non-nil: Pipeline.Run constructs and returns a fully-populated
// Result (with the canonical phase list and error object) on every path,
// including failures, so no nil guard is needed here.
func renderResult(ctx context.Context, cmd *cobra.Command, res *libslocalenv.Result, pipelineErr error) error {
	// Emit one telemetry event per real run (success, failure, or cancel);
	// dry-run runs are not recorded. Best-effort: never affects output.
	logSetupLocalEvent(ctx, res)

	if root.OutputType(cmd) == flags.OutputJSON {
		if err := cmdio.Render(ctx, res); err != nil {
			return err
		}
		// The JSON object is the only thing written to stdout. On failure we still
		// need a non-zero exit, but returning pipelineErr would make the root print
		// "Error: ..." to stderr. ErrAlreadyPrinted exits non-zero without that.
		if pipelineErr != nil {
			return root.ErrAlreadyPrinted
		}
		return nil
	}

	// Text mode: print each phase in execution order.
	for _, phase := range res.Phases {
		if phase.Detail != "" {
			cmdio.LogString(ctx, fmt.Sprintf("%-10s %s  %s", phase.Phase, phase.Status, phase.Detail))
		} else {
			cmdio.LogString(ctx, fmt.Sprintf("%-10s %s", phase.Phase, phase.Status))
		}
	}

	for _, w := range res.Warnings {
		cmdio.LogString(ctx, "warning: "+w.Message)
	}

	if pipelineErr != nil {
		cmdio.LogString(ctx, "For more detail, re-run with --debug, or --output json to share a structured report.")
		// The failing phase's message was already printed by the phase loop above
		// (Pipeline.fail sets the errored phase's Detail to the error text).
		// Returning pipelineErr would make root print "Error: ..." with the same
		// message again, since PipelineError.Unwrap yields the cause, not the
		// ErrAlreadyPrinted sentinel. Signal already-printed to exit non-zero once.
		return root.ErrAlreadyPrinted
	}

	// Print a final success / check summary.
	if res.DryRun {
		if res.Plan != nil {
			cmdio.LogString(ctx, "Plan: "+res.Plan.WouldWrite)
			for _, region := range res.Plan.ChangedRegions {
				cmdio.LogString(ctx, "  changed region: "+region)
			}
		}
		cmdio.LogString(ctx, "Check complete. No files were modified.")
		return nil
	}

	if res.Resolved != nil {
		summary := "Success: python=" + res.Resolved.PythonVersion
		if res.Resolved.DBConnectVersion != "" {
			summary += " databricks-connect=" + res.Resolved.DBConnectVersion
		}
		if res.VenvPath != "" {
			summary += " venv=" + res.VenvPath
		}
		cmdio.LogString(ctx, summary)
	}
	return nil
}
