// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package m_lflow_databricks

import (
	"github.com/databricks/bricks/lib/jsonflag"
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

	getCmd.Flags().StringVar(&getReq.Name, "name", getReq.Name, `Name of the model.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get model.`,
	Long: `Get model.
  
  Get the details of a model. This is a Databricks Workspace version of the
  [MLflow endpoint] that also returns the model's Databricks Workspace ID and
  the permission level of the requesting user on the model.
  
  [MLflow endpoint]: https://www.mlflow.org/docs/latest/rest-api.html#get-registeredmodel`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowDatabricks.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start transition-stage command

var transitionStageReq mlflow.TransitionModelVersionStageDatabricks
var transitionStageJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(transitionStageCmd)
	// TODO: short flags
	transitionStageCmd.Flags().Var(&transitionStageJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	transitionStageCmd.Flags().BoolVar(&transitionStageReq.ArchiveExistingVersions, "archive-existing-versions", transitionStageReq.ArchiveExistingVersions, `Specifies whether to archive all current model versions in the target stage.`)
	transitionStageCmd.Flags().StringVar(&transitionStageReq.Comment, "comment", transitionStageReq.Comment, `User-provided comment on the action.`)
	transitionStageCmd.Flags().StringVar(&transitionStageReq.Name, "name", transitionStageReq.Name, `Name of the model.`)
	transitionStageCmd.Flags().Var(&transitionStageReq.Stage, "stage", `Target stage of the transition.`)
	transitionStageCmd.Flags().StringVar(&transitionStageReq.Version, "version", transitionStageReq.Version, `Version of the model.`)

}

var transitionStageCmd = &cobra.Command{
	Use:   "transition-stage",
	Short: `Transition a stage.`,
	Long: `Transition a stage.
  
  Transition a model version's stage. This is a Databricks Workspace version of
  the [MLflow endpoint] that also accepts a comment associated with the
  transition to be recorded.",
  
  [MLflow endpoint]: https://www.mlflow.org/docs/latest/rest-api.html#transition-modelversion-stage`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = transitionStageJson.Unmarshall(&transitionStageReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowDatabricks.TransitionStage(ctx, transitionStageReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service MLflowDatabricks
