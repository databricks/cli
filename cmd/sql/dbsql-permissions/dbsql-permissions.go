package dbsql_permissions

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dbsql-permissions",
	Short: `The SQL Permissions API is similar to the endpoints of the :method:permissions/setobjectpermissions.`,
	Long: `The SQL Permissions API is similar to the endpoints of the
  :method:permissions/setobjectpermissions. However, this exposes only one
  endpoint, which gets the Access Control List for a given object. You cannot
  modify any permissions using this API.
  
  There are three levels of permission:
  
  - CAN_VIEW: Allows read-only access
  
  - CAN_RUN: Allows read access and run access (superset of CAN_VIEW)
  
  - CAN_MANAGE: Allows all actions: read, run, edit, delete, modify
  permissions (superset of CAN_RUN)`,
}

var getReq sql.GetDbsqlPermissionRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.ObjectId, "object-id", getReq.ObjectId, `Object ID.`)
	getCmd.Flags().Var(&getReq.ObjectType, "object-type", `The type of object permissions to check.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get object ACL.`,
	Long: `Get object ACL.
  
  Gets a JSON representation of the access control list (ACL) for a specified
  object.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.DbsqlPermissions.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var setReq sql.SetRequest

func init() {
	Cmd.AddCommand(setCmd)
	// TODO: short flags

	// TODO: array: access_control_list
	setCmd.Flags().StringVar(&setReq.ObjectId, "object-id", setReq.ObjectId, `Object ID.`)
	setCmd.Flags().Var(&setReq.ObjectType, "object-type", `The type of object permission to set.`)

}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: `Set object ACL.`,
	Long: `Set object ACL.
  
  Sets the access control list (ACL) for a specified object. This operation will
  complete rewrite the ACL.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.DbsqlPermissions.Set(ctx, setReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var transferOwnershipReq sql.TransferOwnershipRequest

func init() {
	Cmd.AddCommand(transferOwnershipCmd)
	// TODO: short flags

	transferOwnershipCmd.Flags().StringVar(&transferOwnershipReq.NewOwner, "new-owner", transferOwnershipReq.NewOwner, `Email address for the new owner, who must exist in the workspace.`)
	// TODO: complex arg: objectId
	transferOwnershipCmd.Flags().Var(&transferOwnershipReq.ObjectType, "object-type", `The type of object on which to change ownership.`)

}

var transferOwnershipCmd = &cobra.Command{
	Use:   "transfer-ownership",
	Short: `Transfer object ownership.`,
	Long: `Transfer object ownership.
  
  Transfers ownership of a dashboard, query, or alert to an active user.
  Requires an admin API key.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.DbsqlPermissions.TransferOwnership(ctx, transferOwnershipReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service DbsqlPermissions
