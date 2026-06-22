package dbconnect

import (
	"context"
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
func renderResult(cmd *cobra.Command, ctx context.Context, res *libsdbconnect.Result, pipelineErr error) error {
	if root.OutputType(cmd) == flags.OutputJSON {
		return cmdio.Render(ctx, res)
	}

	// Text mode: print phase headers.
	for i, phase := range res.Phases {
		cmdio.LogString(ctx, fmt.Sprintf("=== Phase %d: %s ===", i, phase.Name))
		if phase.Detail != "" {
			cmdio.LogString(ctx, fmt.Sprintf("    status=%s  %s", phase.Status, phase.Detail))
		} else {
			cmdio.LogString(ctx, fmt.Sprintf("    status=%s", phase.Status))
		}
	}

	if pipelineErr != nil {
		return pipelineErr
	}

	// Print a final success / check summary.
	if res.Check {
		if res.Plan != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Plan: %s", filepath.ToSlash(res.Plan.PyprojectPath)))
			if len(res.Plan.ChangedRegions) > 0 {
				for _, region := range res.Plan.ChangedRegions {
					cmdio.LogString(ctx, fmt.Sprintf("  changed region: %s", region))
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
