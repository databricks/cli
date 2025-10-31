package debug

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func NewPlanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show deployment plan in JSON format (experimental)",
		Long:  "Show the deployment plan for the current bundle configuration. This command is experimental and may change without notice.",
		Args:  root.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := utils.ProcessOptions{
				AlwaysPull:   true,
				FastValidate: true,
				Build:        true,
			}

			b, isDirectEngine, err := utils.ProcessBundleRet(cmd, opts)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			plan := phases.Plan(ctx, b, isDirectEngine)
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
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
