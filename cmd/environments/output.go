package environments

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/spf13/cobra"
)

// renderResult renders the pipeline result to the command's output.
// In JSON mode it renders the full structured result (even on error).
// In text mode it prints a friendly success/failure summary (per-phase progress
// is shown live via the spinner while the run is in flight), then returns the
// error.
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

	// Text mode. The internal phase log is intentionally NOT printed on success:
	// it read as noise in the M5 bug bash (DECO-27977). Per-phase progress is shown
	// live via the spinner reporter (see cmd/environments/progress.go); the full
	// phase list remains in --output json, and --debug logs each phase as it is
	// entered.
	for _, w := range res.Warnings {
		cmdio.LogString(ctx, "warning: "+w.Message)
	}

	if pipelineErr != nil {
		if res.Error != nil && res.Error.Code == libslocalenv.ErrCanceled {
			cmdio.LogString(ctx, "✗ Setup canceled.")
		} else {
			cmdio.LogString(ctx, "✗ Setup failed"+failureClause(res)+".")
			if res.Error != nil {
				cmdio.LogString(ctx, "")
				// Indent every line: uv-driven failures fold uv's (often multi-line)
				// stderr into the message, and a flush-left continuation reads as
				// unrelated output rather than part of the reason.
				cmdio.LogString(ctx, indent(res.Error.Error(), "  "))
			}
			// Only on an actual failure, not a user interrupt: after Ctrl-C the
			// fallback note reads as a contradictory postscript to "Setup canceled".
			renderInstalledPythonFallback(ctx, res)
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

// renderSuccess prints the friendly post-provision summary.
//
// It runs only on a non-dry-run success (renderResult returns earlier for JSON,
// failures, and dry runs), so res.VenvPath is always set: the validate phase — the
// last thing a successful run does — assigns it unconditionally (see Pipeline.validate).
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
	cmdio.LogString(ctx, fmt.Sprintf("  %-20s%s", "Virtual env", res.VenvPath))
	// pyproject.toml was created (greenfield) or updated in place (with a backup).
	pyprojectDetail := "updated"
	if res.Greenfield {
		pyprojectDetail = "created"
	} else if res.BackupPath != "" {
		pyprojectDetail = "updated (backup: " + res.BackupPath + ")"
	}
	cmdio.LogString(ctx, fmt.Sprintf("  %-20s%s", "pyproject.toml", pyprojectDetail))
	renderInstalledPythonFallback(ctx, res)

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Next steps:")
	cmdio.LogString(ctx, "  • Activate it:  "+activateHint(res.VenvPath))
	cmdio.LogString(ctx, "  • Or select "+res.VenvPath+" as the Python interpreter in VS Code / Cursor")
}

// renderInstalledPythonFallback explains the non-default resolution path on
// both success and failure, so a later provisioning error is diagnosable.
func renderInstalledPythonFallback(ctx context.Context, res *libslocalenv.Result) {
	if res.PythonResolution == libslocalenv.PythonResolutionInstalledFallback {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Python download failed; using the installed interpreter "+res.PythonInterpreter+" instead.")
	}
}

// activateHint returns the shell command to activate the virtual environment,
// matching the running OS. uv lays the venv out as Scripts\activate on Windows
// and bin/activate on Unix (see venvPython in libs/localenv/uv.go, which branches
// the same way); "source" is a POSIX-shell builtin, so Windows gets the bare
// path instead. Printing the Unix form on Windows would hand the user a command
// that fails with "source is not recognized" and a path that does not exist.
func activateHint(venvPath string) string {
	if runtime.GOOS == "windows" {
		return venvPath + `\Scripts\activate`
	}
	return "source " + venvPath + "/bin/activate"
}

// failureClause maps the failing phase to a human clause for the failure line,
// e.g. " while fetching constraints" (note the leading space). Keyed off the
// recorded FailurePhase so text output stays in step with the --output json
// error object. Returns "" for an unknown or missing phase, so the caller reads
// "Setup failed." rather than the redundant "Setup failed during setup."
func failureClause(res *libslocalenv.Result) string {
	if res.Error == nil {
		return ""
	}
	switch res.Error.FailurePhase {
	case libslocalenv.PhasePreflight:
		return " during preflight checks"
	case libslocalenv.PhaseResolve:
		return " while resolving your compute target"
	case libslocalenv.PhaseFetch:
		return " while fetching constraints"
	case libslocalenv.PhaseMerge:
		return " while updating pyproject.toml"
	case libslocalenv.PhaseProvision:
		return " while provisioning the virtual environment"
	case libslocalenv.PhaseValidate:
		return " while validating the environment"
	default:
		return ""
	}
}

// indent prefixes every line of s with prefix. Used so a multi-line failure
// message (uv stderr folded in) stays visually grouped under the failure line.
func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
