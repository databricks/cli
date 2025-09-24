package bundle

import (
	"context"
  "encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show deployment plan",
		Long: `Show the deployment plan for the current bundle configuration.

This command builds the bundle and displays the actions which will be done on resources that would be deployed, without making any changes.
It is useful for previewing changes before running 'bundle deploy'.`,
		Args: root.NoArgs,
	}

	var force bool
	var clusterId string
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation.")
	cmd.Flags().StringVar(&clusterId, "compute-id", "", "Override cluster in the deployment with the given compute ID.")
	cmd.Flags().StringVarP(&clusterId, "cluster-id", "c", "", "Override cluster in the deployment with the given cluster ID.")
	cmd.Flags().MarkDeprecated("compute-id", "use --cluster-id instead")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if f := cmd.Flag("output"); f != nil && f.Changed {
			return errors.New("the -o/--output flag is not supported for this command")
		}
		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) {
			b.Config.Bundle.Force = force

			if cmd.Flag("compute-id").Changed {
				b.Config.Bundle.ClusterId = clusterId
			}

			if cmd.Flag("cluster-id").Changed {
				b.Config.Bundle.ClusterId = clusterId
			}
		})

		phases.Initialize(ctx, b)

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		bundle.ApplyContext(ctx, b, validate.FastValidate())

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.Build(ctx, b)

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		plan := phases.Plan(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		out := cmd.OutOrStdout()

		switch root.OutputType(cmd) {
		case flags.OutputText:
			for _, action := range plan.GetActions() {
				key := strings.TrimPrefix(action.ResourceKey, "resources.")
				fmt.Fprintf(out, "%s %s\n", action.ActionType.StringShort(), key)
			}
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(buf))
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
			return nil
		}

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
}
