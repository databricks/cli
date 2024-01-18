package bundle

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func newBindCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bind KEY RESOURCE_ID",
		Short:   "Bind bundle-defined resources to existing resources",
		PreRunE: ConfigureBundleWithVariables,
		Args:    cobra.ExactArgs(2),
	}

	var autoApprove bool
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Automatically approve the binding")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())
		r := b.Config.Resources
		resourceType, err := r.FindResourceByConfigKey(args[0])
		if err != nil {
			return err
		}

		w := b.WorkspaceClient()
		err = checkResourceExists(cmd.Context(), w, resourceType, args[1])
		if err != nil {
			return fmt.Errorf("%s with an id '%s' is not found, err: %w", resourceType, args[1], err)
		}

		ctx := cmd.Context()
		if !autoApprove {
			answer, err := cmdio.AskYesOrNo(ctx, "Binding to existing resource means that the resource will be managed by the bundle which can lead to changes in the resource. Do you want to continue?")
			if err != nil {
				return err
			}
			if !answer {
				return nil
			}
		}

		return bundle.Apply(cmd.Context(), b, bundle.Seq(
			phases.Initialize(),
			phases.Bind(&phases.BindOptions{
				ResourceType: resourceType,
				ResourceKey:  args[0],
				ResourceId:   args[1],
			}),
		))
	}

	return cmd
}

func checkResourceExists(ctx context.Context, w *databricks.WorkspaceClient, resourceType string, resourceId string) error {
	switch resourceType {
	case "databricks_job":
		id, err := strconv.Atoi(resourceId)
		if err != nil {
			return err
		}
		_, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
			JobId: int64(id),
		})
		return err
	case "databricks_pipeline":
		_, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{
			PipelineId: resourceId,
		})
		return err
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}
