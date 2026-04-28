package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

func newDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Apply ucm configuration to the target Databricks account/workspace.",
		Long: `Apply ucm configuration to the target Databricks account/workspace.

Runs the full deploy sequence: initialize → build → terraform init →
terraform apply → state push. A failure mid-apply leaves the remote state on
the previous seq; re-running the command will re-attempt from a fresh pull.

Common invocations:
  databricks ucm deploy                  # Deploy the default target
  databricks ucm deploy --target prod    # Deploy a specific target`,
		Args: root.NoArgs,
	}

	var autoApprove bool
	var forceLock bool
	var verbose bool
	var readPlanPath string
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for destructive actions.")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output.")
	cmd.Flags().StringVar(&readPlanPath, "plan", "", "Path to a JSON plan file to apply instead of planning (direct engine only).")
	// Verbose flag is parity with bundle; UCM has no file sync today so the
	// flag is currently a no-op. Hidden until file sync lands.
	cmd.Flags().MarkHidden("verbose")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{
			InitFunc: func(u *ucm.Ucm) {
				u.ForceLock = forceLock
				u.AutoApprove = autoApprove
			},
			Verbose:      verbose,
			AlwaysPull:   true,
			FastValidate: true,
			Build:        true,
			ReadPlanPath: readPlanPath,
		})
		ctx := cmd.Context()
		if err != nil {
			return err
		}
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// UCM's phases.Deploy needs a Backend + TerraformFactory that ProcessUcm
		// does not yet plumb (tracked in #103). Until then the verb assembles
		// phases.Options here and runs Deploy directly.
		// (Build: true above is safe — phases.BuildArtifacts is a no-op stub
		// today per #101. Deploy: true would trigger phases.Deploy with
		// zero-value Options that lacks Backend; we run Deploy ourselves below
		// until #103 plumbs Backend through ProcessOptions.)
		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve deploy options: %w", err)
		}
		opts.ForceLock = u.ForceLock
		opts.AutoApprove = u.AutoApprove

		phases.Deploy(ctx, u, opts)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Deploy OK!")
		return nil
	}

	return cmd
}
