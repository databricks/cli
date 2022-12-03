package model_version_comments

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "model-version-comments",
}

var createReq mlflow.CreateComment

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `User-provided comment on the action.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `Name of the model.`)
	createCmd.Flags().StringVar(&createReq.Version, "version", "", `Version of the model.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Post a comment Posts a comment on a model version.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.ModelVersionComments.Create(ctx, createReq)
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

var deleteReq mlflow.DeleteModelVersionCommentRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Id, "id", "", ``)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a comment Deletes a comment on a model version.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.ModelVersionComments.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq mlflow.UpdateComment

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", "", `User-provided comment on the action.`)
	updateCmd.Flags().StringVar(&updateReq.Id, "id", "", `Unique identifier of an activity.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a comment Post an edit to a comment on a model version.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.ModelVersionComments.Update(ctx, updateReq)
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
