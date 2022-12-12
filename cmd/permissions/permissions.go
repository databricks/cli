// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

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

}

var getCmd = &cobra.Command{
	Use:   "get REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID",
	Short: `Get object permissions.`,
	Long: `Get object permissions.
  
  Gets the permission of an object. Objects can inherit permissions from their
  parent objects or root objects.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getReq.RequestObjectType = args[0]
		getReq.RequestObjectId = args[1]

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

}

var getPermissionLevelsCmd = &cobra.Command{
	Use:   "get-permission-levels REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID",
	Short: `Get permission levels.`,
	Long: `Get permission levels.
  
  Gets the permission levels that a user can have on an object.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getPermissionLevelsReq.RequestObjectType = args[0]
		getPermissionLevelsReq.RequestObjectId = args[1]

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

}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: `Set permissions.`,
	Long: `Set permissions.
  
  Sets permissions on object. Objects can inherit permissions from their parent
  objects and root objects.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = setJson.Unmarshall(&setReq)
		if err != nil {
			return err
		}
		setReq.RequestObjectType = args[0]
		setReq.RequestObjectId = args[1]
		setReq.RequestObjectType = args[2]
		setReq.RequestObjectId = args[3]

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

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update permission.`,
	Long: `Update permission.
  
  Updates the permissions on an object.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.RequestObjectType = args[0]
		updateReq.RequestObjectId = args[1]
		updateReq.RequestObjectType = args[2]
		updateReq.RequestObjectId = args[3]

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
