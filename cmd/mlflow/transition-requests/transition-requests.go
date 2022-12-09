package transition_requests

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "transition-requests",
}

var approveReq mlflow.ApproveTransitionRequest

func init() {
	Cmd.AddCommand(approveCmd)
	// TODO: short flags

	approveCmd.Flags().BoolVar(&approveReq.ArchiveExistingVersions, "archive-existing-versions", false, `Specifies whether to archive all current model versions in the target stage.`)
	approveCmd.Flags().StringVar(&approveReq.Comment, "comment", "", `User-provided comment on the action.`)
	approveCmd.Flags().StringVar(&approveReq.Name, "name", "", `Name of the model.`)
	approveCmd.Flags().Var(&approveReq.Stage, "stage", `Target stage of the transition.`)
	approveCmd.Flags().StringVar(&approveReq.Version, "version", "", `Version of the model.`)

}

var approveCmd = &cobra.Command{
	Use:   "approve",
	Short: `Approve transition requests.`,
	Long: `Approve transition requests.
  
  Approves a model version stage transition request.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.Approve(ctx, approveReq)
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

var createReq mlflow.CreateTransitionRequest

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `User-provided comment on the action.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `Name of the model.`)
	createCmd.Flags().Var(&createReq.Stage, "stage", `Target stage of the transition.`)
	createCmd.Flags().StringVar(&createReq.Version, "version", "", `Version of the model.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Make a transition request.`,
	Long: `Make a transition request.
  
  Creates a model version stage transition request.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.Create(ctx, createReq)
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

var deleteReq mlflow.DeleteTransitionRequestRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Comment, "comment", "", `User-provided comment on the action.`)
	deleteCmd.Flags().StringVar(&deleteReq.Creator, "creator", "", `Username of the user who created this request.`)
	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", "", `Name of the model.`)
	deleteCmd.Flags().StringVar(&deleteReq.Stage, "stage", "", `Target stage of the transition request.`)
	deleteCmd.Flags().StringVar(&deleteReq.Version, "version", "", `Version of the model.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a ransition request.`,
	Long: `Delete a ransition request.
  
  Cancels a model version stage transition request.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.TransitionRequests.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var listReq mlflow.ListTransitionRequestsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.Name, "name", "", `Name of the model.`)
	listCmd.Flags().StringVar(&listReq.Version, "version", "", `Version of the model.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List transition requests.`,
	Long: `List transition requests.
  
  Gets a list of all open stage transition requests for the model version.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.ListAll(ctx, listReq)
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

var rejectReq mlflow.RejectTransitionRequest

func init() {
	Cmd.AddCommand(rejectCmd)
	// TODO: short flags

	rejectCmd.Flags().StringVar(&rejectReq.Comment, "comment", "", `User-provided comment on the action.`)
	rejectCmd.Flags().StringVar(&rejectReq.Name, "name", "", `Name of the model.`)
	rejectCmd.Flags().Var(&rejectReq.Stage, "stage", `Target stage of the transition.`)
	rejectCmd.Flags().StringVar(&rejectReq.Version, "version", "", `Version of the model.`)

}

var rejectCmd = &cobra.Command{
	Use:   "reject",
	Short: `Reject a transition request.`,
	Long: `Reject a transition request.
  
  Rejects a model version stage transition request.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.Reject(ctx, rejectReq)
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

// end service TransitionRequests
