// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package transition_requests

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "transition-requests",
}

// start approve command

var approveReq mlflow.ApproveTransitionRequest
var approveJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(approveCmd)
	// TODO: short flags
	approveCmd.Flags().Var(&approveJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	approveCmd.Flags().StringVar(&approveReq.Comment, "comment", approveReq.Comment, `User-provided comment on the action.`)

}

var approveCmd = &cobra.Command{
	Use:   "approve",
	Short: `Approve transition requests.`,
	Long: `Approve transition requests.
  
  Approves a model version stage transition request.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = approveJson.Unmarshall(&approveReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.Approve(ctx, approveReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start create command

var createReq mlflow.CreateTransitionRequest
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided comment on the action.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Make a transition request.`,
	Long: `Make a transition request.
  
  Creates a model version stage transition request.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq mlflow.DeleteTransitionRequestRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Comment, "comment", deleteReq.Comment, `User-provided comment on the action.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME VERSION STAGE CREATOR",
	Short: `Delete a ransition request.`,
	Long: `Delete a ransition request.
  
  Cancels a model version stage transition request.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(4),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.Name = args[0]
		deleteReq.Version = args[1]
		deleteReq.Stage = args[2]
		deleteReq.Creator = args[3]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.TransitionRequests.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start list command

var listReq mlflow.ListTransitionRequestsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

}

var listCmd = &cobra.Command{
	Use:   "list NAME VERSION",
	Short: `List transition requests.`,
	Long: `List transition requests.
  
  Gets a list of all open stage transition requests for the model version.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		listReq.Name = args[0]
		listReq.Version = args[1]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start reject command

var rejectReq mlflow.RejectTransitionRequest
var rejectJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(rejectCmd)
	// TODO: short flags
	rejectCmd.Flags().Var(&rejectJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	rejectCmd.Flags().StringVar(&rejectReq.Comment, "comment", rejectReq.Comment, `User-provided comment on the action.`)

}

var rejectCmd = &cobra.Command{
	Use:   "reject",
	Short: `Reject a transition request.`,
	Long: `Reject a transition request.
  
  Rejects a model version stage transition request.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = rejectJson.Unmarshall(&rejectReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.TransitionRequests.Reject(ctx, rejectReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service TransitionRequests
