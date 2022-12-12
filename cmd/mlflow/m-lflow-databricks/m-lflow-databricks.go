// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package m_lflow_databricks

import (
	"fmt"

	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "m-lflow-databricks",
	Short: `These endpoints are modified versions of the MLflow API that accept additional input parameters or return additional information.`,
	Long: `These endpoints are modified versions of the MLflow API that accept additional
  input parameters or return additional information.`,
}

// start get command

var getReq mlflow.GetMLflowDatabrickRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get model.`,
	Long: `Get model.
  
  Get the details of a model. This is a Databricks Workspace version of the
  [MLflow endpoint] that also returns the model's Databricks Workspace ID and
  the permission level of the requesting user on the model.
  
  [MLflow endpoint]: https://www.mlflow.org/docs/latest/rest-api.html#get-registeredmodel`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getReq.Name = args[0]

		response, err := w.MLflowDatabricks.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start transition-stage command

var transitionStageReq mlflow.TransitionModelVersionStageDatabricks

func init() {
	Cmd.AddCommand(transitionStageCmd)
	// TODO: short flags

	transitionStageCmd.Flags().StringVar(&transitionStageReq.Comment, "comment", transitionStageReq.Comment, `User-provided comment on the action.`)

}

var transitionStageCmd = &cobra.Command{
	Use:   "transition-stage NAME VERSION STAGE ARCHIVE_EXISTING_VERSIONS",
	Short: `Transition a stage.`,
	Long: `Transition a stage.
  
  Transition a model version's stage. This is a Databricks Workspace version of
  the [MLflow endpoint] that also accepts a comment associated with the
  transition to be recorded.",
  
  [MLflow endpoint]: https://www.mlflow.org/docs/latest/rest-api.html#transition-modelversion-stage`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(4),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		transitionStageReq.Name = args[0]
		transitionStageReq.Version = args[1]
		_, err = fmt.Sscan(args[2], &transitionStageReq.Stage)
		if err != nil {
			return fmt.Errorf("invalid STAGE: %s", args[2])
		}
		_, err = fmt.Sscan(args[3], &transitionStageReq.ArchiveExistingVersions)
		if err != nil {
			return fmt.Errorf("invalid ARCHIVE_EXISTING_VERSIONS: %s", args[3])
		}

		response, err := w.MLflowDatabricks.TransitionStage(ctx, transitionStageReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service MLflowDatabricks
