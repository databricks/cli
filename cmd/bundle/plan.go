package bundle

import (
	"encoding/json"
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
	var quiet bool
	var clusterId string
	var selectResources []string
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation.")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Only print the summary line, not the per-resource actions.")
	cmd.Flags().StringVar(&clusterId, "compute-id", "", "Override cluster in the deployment with the given compute ID.")
	cmd.Flags().StringVarP(&clusterId, "cluster-id", "c", "", "Override cluster in the deployment with the given cluster ID.")
	cmd.Flags().MarkDeprecated("compute-id", "use --cluster-id instead")
	cmd.Flags().StringSliceVar(&selectResources, "select", nil, "Plan only the specified resource (e.g. 'my_job' or 'jobs.my_job'). Can be repeated or comma-separated.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts := utils.ProcessOptions{
			AlwaysPull:      true,
			FastValidate:    true,
			Build:           true,
			PreDeployChecks: true,
			InitFunc: func(b *bundle.Bundle) {
				b.Config.Bundle.Force = force
				b.Select = selectResources

				if cmd.Flag("compute-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}

				if cmd.Flag("cluster-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}
			},
		}

		b, stateDesc, err := utils.ProcessBundleRet(cmd, opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		plan := phases.RunPlan(ctx, b, stateDesc.Engine)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		counts := plan.CountActions()

		out := cmd.OutOrStdout()

		switch root.OutputType(cmd) {
		case flags.OutputText:
			// Print summary line and actions to stdout
			totalChanges := counts.Create + counts.Change + counts.Delete
			if totalChanges > 0 && !quiet {
				// Print all actions in the order they were processed
				for _, action := range plan.GetActions() {
					if action.ActionType == deployplan.Skip {
						continue
					}
					key := strings.TrimPrefix(action.ResourceKey, "resources.")
					fmt.Fprintf(out, "%s %s\n", action.ActionType.StringShort(), key)
				}
				fmt.Fprintln(out)
			}
			// Note, this string should not be changed, "bundle deployment migrate" depends on this format:
			fmt.Fprintf(out, "Plan: %d to add, %d to change, %d to delete, %d unchanged", counts.Create, counts.Change, counts.Delete, counts.Unchanged)
			if len(selectResources) > 0 {
				fmt.Fprintf(out, ", %d not selected", plan.NotSelected)
			}
			fmt.Fprintln(out)
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
