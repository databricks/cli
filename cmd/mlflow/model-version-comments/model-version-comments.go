package model_version_comments

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "model-version-comments",
}

// start create command

var createReq mlflow.CreateComment

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided comment on the action.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Name of the model.`)
	createCmd.Flags().StringVar(&createReq.Version, "version", createReq.Version, `Version of the model.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Post a comment.`,
	Long: `Post a comment.
  
  Posts a comment on a model version. A comment can be submitted either by a
  user or programmatically to display relevant information about the model. For
  example, test results or deployment errors.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.ModelVersionComments.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq mlflow.DeleteModelVersionCommentRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", deleteReq.Id, ``)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a comment.`,
	Long: `Delete a comment.
  
  Deletes a comment on a model version.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.ModelVersionComments.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq mlflow.UpdateComment

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided comment on the action.`)
	updateCmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Unique identifier of an activity.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a comment.`,
	Long: `Update a comment.
  
  Post an edit to a comment on a model version.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.ModelVersionComments.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service ModelVersionComments
