package ucm

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/logdiag"
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
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for destructive actions.")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx := cmd.Context()
		if err != nil {
			return err
		}
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve deploy options: %w", err)
		}
		opts.ForceLock = forceLock
		opts.AutoApprove = autoApprove

		phases.Deploy(ctx, u, opts)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Deploy OK!")
		return nil
	}

	return cmd
}
