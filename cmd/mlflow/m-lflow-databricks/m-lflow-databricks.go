package m_lflow_databricks

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "m-lflow-databricks",
	Short: `These endpoints are modified versions of the MLflow API that accept additional input parameters or return additional information.`, // TODO: fix FirstSentence logic and append dot to summary
}

var getReq mlflow.GetMLflowDatabrickRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Name, "name", "", `Name of the model.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get model Get the details of a model.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.MLflowDatabricks.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var transitionStageReq mlflow.TransitionModelVersionStageDatabricks

func init() {
	Cmd.AddCommand(transitionStageCmd)
	// TODO: short flags

	transitionStageCmd.Flags().BoolVar(&transitionStageReq.ArchiveExistingVersions, "archive-existing-versions", false, `Specifies whether to archive all current model versions in the target stage.`)
	transitionStageCmd.Flags().StringVar(&transitionStageReq.Comment, "comment", "", `User-provided comment on the action.`)
	transitionStageCmd.Flags().StringVar(&transitionStageReq.Name, "name", "", `Name of the model.`)
	// TODO: complex arg: stage
	transitionStageCmd.Flags().StringVar(&transitionStageReq.Version, "version", "", `Version of the model.`)

}

var transitionStageCmd = &cobra.Command{
	Use:   "transition-stage",
	Short: `Transition a stage Transition a model version's stage.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.MLflowDatabricks.TransitionStage(ctx, transitionStageReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}
