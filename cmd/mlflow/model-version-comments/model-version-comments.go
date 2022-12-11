// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

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

}

var createCmd = &cobra.Command{
	Use:   "create NAME VERSION COMMENT",
	Short: `Post a comment.`,
	Long: `Post a comment.
  
  Posts a comment on a model version. A comment can be submitted either by a
  user or programmatically to display relevant information about the model. For
  example, test results or deployment errors.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		createReq.Name = args[0]
		createReq.Version = args[1]
		createReq.Comment = args[2]
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

}

var deleteCmd = &cobra.Command{
	Use:   "delete ID",
	Short: `Delete a comment.`,
	Long: `Delete a comment.
  
  Deletes a comment on a model version.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.Id = args[0]
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

}

var updateCmd = &cobra.Command{
	Use:   "update ID COMMENT",
	Short: `Update a comment.`,
	Long: `Update a comment.
  
  Post an edit to a comment on a model version.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		updateReq.Id = args[0]
		updateReq.Comment = args[1]
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
