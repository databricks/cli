package bundle

import (
	"context"
	"encoding/json"
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
		Args:  root.NoArgs,

		// Output format may change without notice; main use case is in acceptance tests.
		// Today, this command also uploads libraries, which is not the intent here. We need to refactor
		// libraries.Upload() mutator to separate config mutation with actual upload.
		Hidden: true,
	}

	var force bool
	var clusterId string
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation.")
	cmd.Flags().StringVar(&clusterId, "compute-id", "", "Override cluster in the deployment with the given compute ID.")
	cmd.Flags().StringVarP(&clusterId, "cluster-id", "c", "", "Override cluster in the deployment with the given cluster ID.")
	cmd.Flags().MarkDeprecated("compute-id", "use --cluster-id instead")

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

		// Count actions by type and collect formatted actions
		createCount := 0
		updateCount := 0
		deleteCount := 0
		changed := make(map[string]bool)

		for _, change := range plan.GetActions() {
			changed[change.ResourceKey] = true
			switch change.ActionType.StringShort() {
			case "create":
				createCount++
			case "update":
				updateCount++
			case "delete":
				deleteCount++
			case "recreate":
				// A recreate counts as both a delete and a create
				deleteCount++
				createCount++
			}
		}

		// Calculate number of all unchanged resources
		unchanged := 0
		for _, group := range b.Config.Resources.AllResources() {
			for rKey := range group.Resources {
				resourceKey := "resources." + group.Description.PluralName + "." + rKey
				if _, ok := changed[resourceKey]; !ok {
					unchanged++
				}
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
					key := strings.TrimPrefix(action.ResourceKey, "resources.")
					fmt.Fprintf(out, "%s %s\n", action.ActionType.StringShort(), key)
				}
			}
			fmt.Fprintf(out, "Plan: %d to add, %d to change, %d to delete, %d unchanged\n", createCount, updateCount, deleteCount, unchanged)
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
