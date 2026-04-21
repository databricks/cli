package ucm

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

func newDestroyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Tear down everything managed by the current target.",
		Long: `Tear down everything managed by the current target.

Runs the initialize → terraform init → terraform destroy → state push sequence
against the selected target. Operates on the already-rendered terraform config
cached from the last apply.

Common invocations:
  databricks ucm destroy --auto-approve                # Destroy default target
  databricks ucm destroy --target dev --auto-approve   # Destroy a specific target`,
		Args:    root.NoArgs,
		PreRunE: utils.MustWorkspaceClient,
	}

	var autoApprove bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip interactive approvals for deleting resources.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if !cmdio.IsPromptSupported(ctx) && !autoApprove {
			return errors.New("please specify --auto-approve since terminal does not support interactive prompts")
		}

		u := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx = cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve deploy options: %w", err)
		}

		phases.Destroy(ctx, u, opts)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Destroy OK!")
		return nil
	}

	return cmd
}
