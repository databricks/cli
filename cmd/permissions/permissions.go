package permissions

import (
	workspace_assignment "github.com/databricks/bricks/cmd/permissions/workspace-assignment"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/permissions"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "permissions",
	Short: `Permissions API are used to create read, write, edit, update and manage access for various users on different objects and endpoints.`,
	Long: `Permissions API are used to create read, write, edit, update and manage access
  for various users on different objects and endpoints.`,
}

// start get command

var getReq permissions.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.RequestObjectId, "request-object-id", getReq.RequestObjectId, ``)
	getCmd.Flags().StringVar(&getReq.RequestObjectType, "request-object-type", getReq.RequestObjectType, `<needs content>.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get object permissions.`,
	Long: `Get object permissions.
  
  Gets the permission of an object. Objects can inherit permissions from their
  parent objects or root objects.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Permissions.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-permission-levels command

var getPermissionLevelsReq permissions.GetPermissionLevels

func init() {
	Cmd.AddCommand(getPermissionLevelsCmd)
	// TODO: short flags

	getPermissionLevelsCmd.Flags().StringVar(&getPermissionLevelsReq.RequestObjectId, "request-object-id", getPermissionLevelsReq.RequestObjectId, `<needs content>.`)
	getPermissionLevelsCmd.Flags().StringVar(&getPermissionLevelsReq.RequestObjectType, "request-object-type", getPermissionLevelsReq.RequestObjectType, `<needs content>.`)

}

var getPermissionLevelsCmd = &cobra.Command{
	Use:   "get-permission-levels",
	Short: `Get permission levels.`,
	Long: `Get permission levels.
  
  Gets the permission levels that a user can have on an object.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Permissions.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set command

var setReq permissions.PermissionsRequest
var setJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(setCmd)
	// TODO: short flags
	setCmd.Flags().Var(&setJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list
	setCmd.Flags().StringVar(&setReq.RequestObjectId, "request-object-id", setReq.RequestObjectId, ``)
	setCmd.Flags().StringVar(&setReq.RequestObjectType, "request-object-type", setReq.RequestObjectType, `<needs content>.`)

}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: `Set permissions.`,
	Long: `Set permissions.
  
  Sets permissions on object. Objects can inherit permissions from their parent
  objects and root objects.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = setJson.Unmarshall(&setReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Permissions.Set(ctx, setReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq permissions.PermissionsRequest
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list
	updateCmd.Flags().StringVar(&updateReq.RequestObjectId, "request-object-id", updateReq.RequestObjectId, ``)
	updateCmd.Flags().StringVar(&updateReq.RequestObjectType, "request-object-type", updateReq.RequestObjectType, `<needs content>.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update permission.`,
	Long: `Update permission.
  
  Updates the permissions on an object.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.Permissions.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Permissions

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(workspace_assignment.Cmd)
}
