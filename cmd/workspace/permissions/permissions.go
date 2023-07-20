// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package permissions

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: `Permissions API are used to create read, write, edit, update and manage access for various users on different objects and endpoints.`,
		Long: `Permissions API are used to create read, write, edit, update and manage access
  for various users on different objects and endpoints.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
	}

	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newSet())
	cmd.AddCommand(newUpdate())

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*iam.GetPermissionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq iam.GetPermissionRequest

	// TODO: short flags

	cmd.Use = "get REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Get object permissions.`
	cmd.Long = `Get object permissions.
  
  Gets the permission of an object. Objects can inherit permissions from their
  parent objects or root objects.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.RequestObjectType = args[0]
		getReq.RequestObjectId = args[1]

		response, err := w.Permissions.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*iam.GetPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq iam.GetPermissionLevelsRequest

	// TODO: short flags

	cmd.Use = "get-permission-levels REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Get permission levels.`
	cmd.Long = `Get permission levels.
  
  Gets the permission levels that a user can have on an object.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getPermissionLevelsReq.RequestObjectType = args[0]
		getPermissionLevelsReq.RequestObjectId = args[1]

		response, err := w.Permissions.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd, &getPermissionLevelsReq)
	}

	return cmd
}

// start set command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setOverrides []func(
	*cobra.Command,
	*iam.PermissionsRequest,
)

func newSet() *cobra.Command {
	cmd := &cobra.Command{}

	var setReq iam.PermissionsRequest
	var setJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&setJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Set permissions.`
	cmd.Long = `Set permissions.
  
  Sets permissions on object. Objects can inherit permissions from their parent
  objects and root objects.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = setJson.Unmarshal(&setReq)
			if err != nil {
				return err
			}
		}
		setReq.RequestObjectType = args[0]
		setReq.RequestObjectId = args[1]

		err = w.Permissions.Set(ctx, setReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setOverrides {
		fn(cmd, &setReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*iam.PermissionsRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq iam.PermissionsRequest
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Update permission.`
	cmd.Long = `Update permission.
  
  Updates the permissions on an object.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		}
		updateReq.RequestObjectType = args[0]
		updateReq.RequestObjectId = args[1]

		err = w.Permissions.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service Permissions
