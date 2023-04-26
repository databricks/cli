// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package permissions

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "permissions",
	Short: `Permissions API are used to create read, write, edit, update and manage access for various users on different objects and endpoints.`,
	Long: `Permissions API are used to create read, write, edit, update and manage access
  for various users on different objects and endpoints.`,
}

// start get command

var getReq iam.GetPermissionRequest

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getReq.RequestObjectType = args[0]
		getReq.RequestObjectId = args[1]

		response, err := w.Permissions.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get-permission-levels command

var getPermissionLevelsReq iam.GetPermissionLevelsRequest

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getPermissionLevelsReq.RequestObjectType = args[0]
		getPermissionLevelsReq.RequestObjectId = args[1]

		response, err := w.Permissions.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start set command

var setReq iam.PermissionsRequest
var setJson flags.JsonFlag

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = setJson.Unmarshal(&setReq)
		if err != nil {
			return err
		}
		setReq.RequestObjectType = args[0]
		setReq.RequestObjectId = args[1]

		err = w.Permissions.Set(ctx, setReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq iam.PermissionsRequest
var updateJson flags.JsonFlag

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
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updateJson.Unmarshal(&updateReq)
		if err != nil {
			return err
		}
		updateReq.RequestObjectType = args[0]
		updateReq.RequestObjectId = args[1]

		err = w.Permissions.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Permissions
