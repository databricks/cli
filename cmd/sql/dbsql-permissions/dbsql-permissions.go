// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package dbsql_permissions

import (
	"fmt"

	"github.com/databricks/bricks/lib/jsonflag"
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

// start get command

var getReq sql.GetDbsqlPermissionRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get OBJECT_TYPE OBJECT_ID",
	Short: `Get object ACL.`,
	Long: `Get object ACL.
  
  Gets a JSON representation of the access control list (ACL) for a specified
  object.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		_, err = fmt.Sscan(args[0], &getReq.ObjectType)
		if err != nil {
			return fmt.Errorf("invalid OBJECT_TYPE: %s", args[0])
		}
		getReq.ObjectId = args[1]

		response, err := w.DbsqlPermissions.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set command

var setReq sql.SetRequest
var setJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(setCmd)
	// TODO: short flags
	setCmd.Flags().Var(&setJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: `Set object ACL.`,
	Long: `Set object ACL.
  
  Sets the access control list (ACL) for a specified object. This operation will
  complete rewrite the ACL.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = setJson.Unmarshall(&setReq)
		if err != nil {
			return err
		}
		_, err = fmt.Sscan(args[0], &setReq.ObjectType)
		if err != nil {
			return fmt.Errorf("invalid OBJECT_TYPE: %s", args[0])
		}
		setReq.ObjectId = args[1]

		response, err := w.DbsqlPermissions.Set(ctx, setReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start transfer-ownership command

var transferOwnershipReq sql.TransferOwnershipRequest
var transferOwnershipJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(transferOwnershipCmd)
	// TODO: short flags
	transferOwnershipCmd.Flags().Var(&transferOwnershipJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	transferOwnershipCmd.Flags().StringVar(&transferOwnershipReq.NewOwner, "new-owner", transferOwnershipReq.NewOwner, `Email address for the new owner, who must exist in the workspace.`)

}

var transferOwnershipCmd = &cobra.Command{
	Use:   "transfer-ownership",
	Short: `Transfer object ownership.`,
	Long: `Transfer object ownership.
  
  Transfers ownership of a dashboard, query, or alert to an active user.
  Requires an admin API key.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = transferOwnershipJson.Unmarshall(&transferOwnershipReq)
		if err != nil {
			return err
		}
		_, err = fmt.Sscan(args[0], &transferOwnershipReq.ObjectType)
		if err != nil {
			return fmt.Errorf("invalid OBJECT_TYPE: %s", args[0])
		}
		_, err = fmt.Sscan(args[1], &transferOwnershipReq.ObjectId)
		if err != nil {
			return fmt.Errorf("invalid OBJECT_ID: %s", args[1])
		}

		response, err := w.DbsqlPermissions.TransferOwnership(ctx, transferOwnershipReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service DbsqlPermissions
