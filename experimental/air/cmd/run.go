package aircmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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
		permissions    []string
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
	cmd.Flags().StringArrayVar(&permissions, "permissions", nil, `Grant a permission on the job, e.g. "CAN_VIEW=group_name:users" or "CAN_MANAGE=user_name:a@b.com" (repeatable). Merged with the config's permissions block.`)
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

		// --permissions flags append to the config's permissions block, then the whole
		// set is re-validated so flag and YAML grants are held to the same rules.
		if len(permissions) > 0 {
			parsed, err := parsePermissions(permissions)
			if err != nil {
				return err
			}
			cfg.Permissions = append(cfg.Permissions, parsed...)
			for i := range cfg.Permissions {
				if err := cfg.Permissions[i].validate(); err != nil {
					return err
				}
			}
		}

		if dryRun {
			// Show the generated bundle without deploying.
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

// parsePermissions parses --permissions flag values of the form
// "LEVEL=principal_type:name", e.g. "CAN_VIEW=group_name:users". principal_type is
// one of user_name, group_name, service_principal_name. The result is validated by
// the caller alongside any config permissions.
func parsePermissions(specs []string) ([]permission, error) {
	out := make([]permission, 0, len(specs))
	for _, spec := range specs {
		level, principal, ok := strings.Cut(spec, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --permissions %q: expected LEVEL=principal_type:name (e.g. CAN_VIEW=group_name:users)", spec)
		}
		kind, name, ok := strings.Cut(principal, ":")
		if !ok {
			return nil, fmt.Errorf("invalid --permissions %q: principal must be principal_type:name (e.g. group_name:users)", spec)
		}
		p := permission{Level: strings.TrimSpace(level)}
		switch strings.TrimSpace(kind) {
		case "user_name":
			p.UserName = &name
		case "group_name":
			p.GroupName = &name
		case "service_principal_name":
			p.ServicePrincipalName = &name
		default:
			return nil, fmt.Errorf("invalid --permissions %q: principal type %q must be user_name, group_name, or service_principal_name", spec, kind)
		}
		out = append(out, p)
	}
	return out, nil
}
