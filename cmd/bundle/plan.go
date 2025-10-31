package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
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
			return errors.New("the -o/--output flag is not supported for this command. Use an experimental 'databricks bundle debug plan' command instead")
		}
		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts := utils.ProcessOptions{
			InitFunc: func(b *bundle.Bundle) {
				b.Config.Bundle.Force = force

				if cmd.Flag("compute-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}

				if cmd.Flag("cluster-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}
			},
			AlwaysPull:   true,
			FastValidate: true,
			Build:        true,
		}

		b, isDirectEngine, err := utils.ProcessBundleRet(cmd, &opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		plan := phases.Plan(ctx, b, isDirectEngine)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// Count actions by type and collect formatted actions
		createCount := 0
		updateCount := 0
		deleteCount := 0
		unchangedCount := 0

		for _, change := range plan.GetActions() {
			switch change.ActionType {
			case deployplan.ActionTypeCreate:
				createCount++
			case deployplan.ActionTypeUpdate, deployplan.ActionTypeUpdateWithID, deployplan.ActionTypeResize:
				updateCount++
			case deployplan.ActionTypeDelete:
				deleteCount++
			case deployplan.ActionTypeRecreate:
				// A recreate counts as both a delete and a create
				deleteCount++
				createCount++
			case deployplan.ActionTypeSkip, deployplan.ActionTypeUndefined:
				unchangedCount++
			}
		}

		out := cmd.OutOrStdout()

		switch root.OutputType(cmd) {
		case flags.OutputText:
			// Print summary line and actions to stdout
			totalChanges := createCount + updateCount + deleteCount
			if totalChanges > 0 {
				// Print all actions in the order they were processed
				for _, action := range plan.GetActions() {
					if action.ActionType == deployplan.ActionTypeSkip {
						continue
					}
					key := strings.TrimPrefix(action.ResourceKey, "resources.")
					fmt.Fprintf(out, "%s %s\n", action.ActionType.StringShort(), key)
				}
				fmt.Fprintln(out)
			}
			fmt.Fprintf(out, "Plan: %d to add, %d to change, %d to delete, %d unchanged\n", createCount, updateCount, deleteCount, unchangedCount)
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
