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

	// Text mode. The internal phase log is intentionally NOT printed on success:
	// it read as noise in the M5 bug bash (DECO-27977). Per-phase progress is shown
	// live via the spinner reporter (see cmd/environments/progress.go) and the full
	// phase list remains in --output json and --debug.
	for _, w := range res.Warnings {
		cmdio.LogString(ctx, "warning: "+w.Message)
	}

	if pipelineErr != nil {
		if res.Error != nil && res.Error.Code == libslocalenv.ErrCanceled {
			cmdio.LogString(ctx, "✗ Setup canceled.")
		} else {
			cmdio.LogString(ctx, "✗ Setup failed "+failureClause(res)+".")
			if res.Error != nil {
				cmdio.LogString(ctx, "")
				cmdio.LogString(ctx, "  "+res.Error.Error())
			}
		}
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Re-run with --debug for details, or --output json for a structured report.")
		// The failing message is already surfaced above; ErrAlreadyPrinted exits
		// non-zero without root re-printing "Error: ...".
		return root.ErrAlreadyPrinted
	}

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

	renderSuccess(ctx, res)
	return nil
}

// renderSuccess prints the friendly post-provision summary (DECO-27977).
func renderSuccess(ctx context.Context, res *libslocalenv.Result) {
	cmdio.LogString(ctx, "✔ Local environment ready")
	cmdio.LogString(ctx, "")

	if res.Compute != nil {
		cmdio.LogString(ctx, fmt.Sprintf("  %-20s%s", "Compute target", res.Compute.Label()))
	}
	if res.Resolved != nil {
		cmdio.LogString(ctx, fmt.Sprintf("  %-20s%s", "Python", res.Resolved.PythonVersion))
		if res.Resolved.DBConnectVersion != "" {
			cmdio.LogString(ctx, fmt.Sprintf("  %-20s%s", "databricks-connect", res.Resolved.DBConnectVersion))
		}
	}
	if res.VenvPath != "" {
		cmdio.LogString(ctx, fmt.Sprintf("  %-20s%s", "Virtual env", res.VenvPath))
	}
	// pyproject.toml was created (greenfield) or updated in place (with a backup).
	pyprojectDetail := "updated"
	if res.Greenfield {
		pyprojectDetail = "created"
	} else if res.BackupPath != "" {
		pyprojectDetail = "updated (backup: " + res.BackupPath + ")"
	}
	cmdio.LogString(ctx, fmt.Sprintf("  %-20s%s", "pyproject.toml", pyprojectDetail))

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Next steps:")
	if res.VenvPath != "" {
		cmdio.LogString(ctx, "  • Activate it:  source "+res.VenvPath+"/bin/activate")
		cmdio.LogString(ctx, "  • Or select "+res.VenvPath+" as the Python interpreter in VS Code / Cursor")
	} else {
		cmdio.LogString(ctx, "  • Activate the .venv it created and select it as your interpreter")
	}
}

// failureClause maps the failing phase to a human clause for the failure line,
// e.g. "while fetching constraints". Keyed off the recorded FailurePhase so text
// output stays in step with the --output json error object.
func failureClause(res *libslocalenv.Result) string {
	if res.Error == nil {
		return "during setup"
	}
	switch res.Error.FailurePhase {
	case libslocalenv.PhasePreflight:
		return "during preflight checks"
	case libslocalenv.PhaseResolve:
		return "while resolving your compute target"
	case libslocalenv.PhaseFetch:
		return "while fetching constraints"
	case libslocalenv.PhaseMerge:
		return "while updating pyproject.toml"
	case libslocalenv.PhaseProvision:
		return "while provisioning the virtual environment"
	case libslocalenv.PhaseValidate:
		return "while validating the environment"
	default:
		return "during setup"
	}
}
