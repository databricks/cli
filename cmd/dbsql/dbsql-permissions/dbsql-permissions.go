package dbsql_permissions

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
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

var getPermissionsReq dbsql.GetPermissionsRequest

func init() {
	Cmd.AddCommand(getPermissionsCmd)
	// TODO: short flags

	getPermissionsCmd.Flags().StringVar(&getPermissionsReq.ObjectId, "object-id", getPermissionsReq.ObjectId, `Object ID.`)
	getPermissionsCmd.Flags().Var(&getPermissionsReq.ObjectType, "object-type", `The type of object permissions to check.`)

}

var getPermissionsCmd = &cobra.Command{
	Use:   "get-permissions",
	Short: `Get object ACL.`,
	Long: `Get object ACL.
  
  Gets a JSON representation of the access control list (ACL) for a specified
  object.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.DbsqlPermissions.GetPermissions(ctx, getPermissionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var setPermissionsReq dbsql.SetPermissionsRequest

func init() {
	Cmd.AddCommand(setPermissionsCmd)
	// TODO: short flags

	// TODO: array: access_control_list
	setPermissionsCmd.Flags().StringVar(&setPermissionsReq.ObjectId, "object-id", setPermissionsReq.ObjectId, `Object ID.`)
	setPermissionsCmd.Flags().Var(&setPermissionsReq.ObjectType, "object-type", `The type of object permission to set.`)

}

var setPermissionsCmd = &cobra.Command{
	Use:   "set-permissions",
	Short: `Set object ACL.`,
	Long: `Set object ACL.
  
  Sets the access control list (ACL) for a specified object. This operation will
  complete rewrite the ACL.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.DbsqlPermissions.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var transferOwnershipReq dbsql.TransferOwnershipRequest

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
