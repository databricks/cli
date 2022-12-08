package workspace_assignment

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/permissions"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace-assignment",
	Short: `Databricks Workspace Assignment REST API.`,
}

var createReq permissions.CreateWorkspaceAssignments

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: permission_assignments
	createCmd.Flags().Int64Var(&createReq.WorkspaceId, "workspace-id", 0, `The workspace ID for the account.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create permission assignments.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.WorkspaceAssignment.Create(ctx, createReq)
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

var deleteReq permissions.DeleteWorkspaceAssignmentRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().Int64Var(&deleteReq.PrincipalId, "principal-id", 0, `The ID of the service principal.`)
	deleteCmd.Flags().Int64Var(&deleteReq.WorkspaceId, "workspace-id", 0, `The workspace ID.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete permissions assignment.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.WorkspaceAssignment.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq permissions.GetWorkspaceAssignmentRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().Int64Var(&getReq.WorkspaceId, "workspace-id", 0, `The workspace ID.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `List workspace permissions.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.WorkspaceAssignment.Get(ctx, getReq)
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

var listReq permissions.ListWorkspaceAssignmentRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().Int64Var(&listReq.WorkspaceId, "workspace-id", 0, `The workspace ID for the account.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get permission assignments.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		response, err := a.WorkspaceAssignment.ListAll(ctx, listReq)
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

var updateReq permissions.UpdateWorkspaceAssignments

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	// TODO: complex arg: permissions
	updateCmd.Flags().Int64Var(&updateReq.PrincipalId, "principal-id", 0, `The ID of the service principal.`)
	updateCmd.Flags().Int64Var(&updateReq.WorkspaceId, "workspace-id", 0, `The workspace ID.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update permissions assignment.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.WorkspaceAssignment.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
