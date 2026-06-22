package dbconnect

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	libsdbconnect "github.com/databricks/cli/libs/dbconnect"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// renderResult renders the pipeline result to the command's output.
// In JSON mode it renders the full structured result (even on error).
// In text mode it prints phase headers and a summary, then returns the error.
func renderResult(ctx context.Context, cmd *cobra.Command, res *libsdbconnect.Result, pipelineErr error) error {
	// Guard against a nil result (e.g. pipeline failed before constructing one).
	// Always emit a structured object in JSON mode so callers can rely on the schema.
	if res == nil {
		res = &libsdbconnect.Result{}
		if pipelineErr != nil {
			if pe, ok := errors.AsType[*libsdbconnect.PipelineError](pipelineErr); ok {
				res.Error = pe
			} else {
				res.Error = libsdbconnect.NewError(libsdbconnect.ErrProvisionFailed, pipelineErr, "%s", pipelineErr.Error())
			}
		}
	}

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

	// Text mode: print phase headers.
	for i, phase := range res.Phases {
		cmdio.LogString(ctx, fmt.Sprintf("=== Phase %d: %s ===", i+1, phase.Name))
		if phase.Detail != "" {
			cmdio.LogString(ctx, fmt.Sprintf("    status=%s  %s", phase.Status, phase.Detail))
		} else {
			cmdio.LogString(ctx, "    status="+phase.Status)
		}
	}

	if pipelineErr != nil {
		cmdio.LogString(ctx, "For more detail, re-run with --debug, or --output json to share a structured report.")
		return pipelineErr
	}

	// Print a final success / check summary.
	if res.Check {
		if res.Plan != nil {
			cmdio.LogString(ctx, "Plan: "+filepath.ToSlash(res.Plan.PyprojectPath))
			if len(res.Plan.ChangedRegions) > 0 {
				for _, region := range res.Plan.ChangedRegions {
					cmdio.LogString(ctx, "  changed region: "+region)
				}
			}
		}
		cmdio.LogString(ctx, "Check complete. No files were modified.")
		return nil
	}

	if res.Result != nil {
		cmdio.LogString(ctx, fmt.Sprintf("Success: python=%s databricks-connect=%s venv=%s",
			res.Result.PythonVersion,
			res.Result.DatabricksConnectInstalled,
			filepath.ToSlash(res.Result.VenvPath),
		))
	}
	return nil
}
