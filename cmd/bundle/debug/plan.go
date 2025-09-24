package debug

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func NewPlanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show deployment plan",
		Long:  "Show the deployment plan for the current bundle configuration. This command is experimental and may change without notice.",
		Args:  root.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := logdiag.InitContext(cmd.Context())
			cmd.SetContext(ctx)
			b := utils.ConfigureBundleWithVariables(cmd)
			if b == nil || logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
			plan, err := utils.GetPlan(ctx, b)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()

			buf, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(buf))
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
			return nil
		},
	}
}
