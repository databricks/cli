package deployment

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bind KEY RESOURCE_ID",
		Short:   "Bind bundle-defined resources to existing resources",
		Args:    cobra.ExactArgs(2),
		PreRunE: utils.ConfigureBundleWithVariables,
	}

	var autoApprove bool
	var forceLock bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the binding")
	cmd.Flags().BoolVar(&forceLock, "force-lock", false, "Force acquisition of deployment lock.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())
		r := b.Config.Resources
		resource, err := r.FindResourceByConfigKey(args[0])
		if err != nil {
			return err
		}

		w := b.WorkspaceClient()
		ctx := cmd.Context()
		if !resource.Exists(ctx, w, args[1]) {
			return fmt.Errorf("%s with an id '%s' is not found", resource.Type(), args[1])
		}

		if !autoApprove {
			answer, err := cmdio.AskYesOrNo(ctx, "Binding to existing resource means that the resource will be managed by the bundle which can lead to changes in the resource. Do you want to continue?")
			if err != nil {
				return err
			}
			if !answer {
				return nil
			}
		}

		b.Config.Bundle.Lock.Force = forceLock
		return bundle.Apply(cmd.Context(), b, bundle.Seq(
			phases.Initialize(),
			phases.Bind(&terraform.BindOptions{
				ResourceType: resource.Type(),
				ResourceKey:  args[0],
				ResourceId:   args[1],
			}),
		))
	}

	return cmd
}
