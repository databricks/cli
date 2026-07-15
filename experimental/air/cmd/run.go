package aircmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// runResult is the JSON payload for `air run`.
type runResult struct {
	Status       string `json:"status"`
	DryRun       bool   `json:"dry_run,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	DashboardURL string `json:"dashboard_url,omitempty"`
	// Bundle is the generated databricks.yml, included in a --dry-run so the user
	// can see exactly what would be deployed on their behalf.
	Bundle string `json:"bundle,omitempty"`
}

func newRunCommand() *cobra.Command {
	var (
		file           string
		watch          bool
		overrides      []string
		dryRun         bool
		idempotencyKey string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Args:  root.NoArgs,
		Short: "Submit a training workload from a YAML config",
		Long: `Submit a training workload to Databricks serverless GPU compute.

The workload is described by a YAML config file (see --file).`,
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to the workload YAML config")
	cmd.Flags().BoolVar(&watch, "watch", false, "Stream logs until the run completes")
	cmd.Flags().StringArrayVar(&overrides, "override", nil, "Override a YAML field, e.g. compute.num_accelerators=8 (repeatable)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate the config and show the generated bundle without submitting")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Return the existing run if this key was already used")
	_ = cmd.MarkFlagRequired("file")

	// --dry-run only validates the config locally, so it needs no workspace.
	// Submission requires an authenticated client.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if dryRun {
			return nil
		}
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// These flags' pipelines are not ported yet; reject rather than silently
		// ignore them.
		if len(overrides) > 0 {
			return errors.New("--override is not yet supported")
		}
		if watch {
			return errors.New("--watch is not yet supported")
		}

		cfg, err := loadRunConfig(file)
		if err != nil {
			return err
		}

		if dryRun {
			// A dry run shows the generated bundle so the user can see the artifact
			// we'd deploy on their behalf (transparency); it does not deploy.
			bundleYAML, err := renderBundle(cfg, file)
			if err != nil {
				return err
			}
			if root.OutputType(cmd) == flags.OutputText {
				cmdio.LogString(ctx, fmt.Sprintf("Dry run: %q would deploy as this bundle (not deploying):\n\n%s", cfg.ExperimentName, bundleYAML))
				return nil
			}
			return renderEnvelope(ctx, runResult{Status: "DRY_RUN_OK", DryRun: true, Bundle: bundleYAML})
		}

		w := cmdctx.WorkspaceClient(ctx)

		// submitWorkload converts the config to a bundle, deploys it, and triggers a
		// run in-process (see rundabs.go). It is the only submit path.
		runID, dashboardURL, err := submitWorkload(ctx, w, cfg, file, idempotencyKey)
		if err != nil {
			return err
		}

		runIDStr := strconv.FormatInt(runID, 10)
		if root.OutputType(cmd) == flags.OutputText {
			cmdio.LogString(ctx, "Submitted run "+runIDStr)
			cmdio.LogString(ctx, "View at: "+dashboardURL)
			return nil
		}
		return renderEnvelope(ctx, runResult{Status: "SUBMITTED", RunID: runIDStr, DashboardURL: dashboardURL})
	}

	return cmd
}
