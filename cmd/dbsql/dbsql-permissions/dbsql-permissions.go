package dbsql_permissions

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dbsql-permissions",
	Short: `The SQL Permissions API is similar to the endpoints of the :method:permissions/setobjectpermissions.`,
}

var getPermissionsReq dbsql.GetPermissionsRequest

func init() {
	Cmd.AddCommand(getPermissionsCmd)
	// TODO: short flags

	getPermissionsCmd.Flags().StringVar(&getPermissionsReq.ObjectId, "object-id", "", `Object ID.`)
	getPermissionsCmd.Flags().Var(&getPermissionsReq.ObjectType, "object-type", `The type of object permissions to check.`)

}

var getPermissionsCmd = &cobra.Command{
	Use:   "get-permissions",
	Short: `Get object ACL.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.DbsqlPermissions.GetPermissions(ctx, getPermissionsReq)
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

var setPermissionsReq dbsql.SetPermissionsRequest

func init() {
	Cmd.AddCommand(setPermissionsCmd)
	// TODO: short flags

	// TODO: array: access_control_list
	setPermissionsCmd.Flags().StringVar(&setPermissionsReq.ObjectId, "object-id", "", `Object ID.`)
	setPermissionsCmd.Flags().Var(&setPermissionsReq.ObjectType, "object-type", `The type of object permission to set.`)

}

var setPermissionsCmd = &cobra.Command{
	Use:   "set-permissions",
	Short: `Set object ACL.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.DbsqlPermissions.SetPermissions(ctx, setPermissionsReq)
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

var transferOwnershipReq dbsql.TransferOwnershipRequest

func init() {
	Cmd.AddCommand(transferOwnershipCmd)
	// TODO: short flags

	transferOwnershipCmd.Flags().StringVar(&transferOwnershipReq.NewOwner, "new-owner", "", `Email address for the new owner, who must exist in the workspace.`)
	// TODO: complex arg: objectId
	transferOwnershipCmd.Flags().Var(&transferOwnershipReq.ObjectType, "object-type", `The type of object on which to change ownership.`)

}

var transferOwnershipCmd = &cobra.Command{
	Use:   "transfer-ownership",
	Short: `Transfer object ownership.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.DbsqlPermissions.TransferOwnership(ctx, transferOwnershipReq)
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
