// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: `The Workspace API allows you to list, import, export, and delete notebooks and folders.`,
		Long: `The Workspace API allows you to list, import, export, and delete notebooks and
  folders.
  
  A notebook is a web-based interface to a document that contains runnable code,
  visualizations, and explanatory text.`,
		GroupID: "workspace",
		Annotations: map[string]string{
			"package": "workspace",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newExport())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetStatus())
	cmd.AddCommand(newImport())
	cmd.AddCommand(newList())
	cmd.AddCommand(newMkdirs())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newUpdatePermissions())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*workspace.Delete,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq workspace.Delete
	var deleteJson flags.JsonFlag

	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&deleteReq.Recursive, "recursive", deleteReq.Recursive, `The flag that specifies whether to delete the object recursively.`)

	cmd.Use = "delete PATH"
	cmd.Short = `Delete a workspace object.`
	cmd.Long = `Delete a workspace object.
  
  Deletes an object or a directory (and optionally recursively deletes all
  objects in the directory). * If path does not exist, this call returns an
  error RESOURCE_DOES_NOT_EXIST. * If path is a non-empty directory and
  recursive is set to false, this call returns an error
  DIRECTORY_NOT_EMPTY.
  
  Object deletion cannot be undone and deleting a directory recursively is not
  atomic.

  Arguments:
    PATH: The absolute path of the notebook or directory.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'path' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteJson.Unmarshal(&deleteReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No PATH argument specified. Loading names for Workspace drop-down."
				names, err := w.Workspace.ObjectInfoPathToObjectIdMap(ctx, workspace.ListWorkspaceRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Workspace drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The absolute path of the notebook or directory")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the absolute path of the notebook or directory")
			}
			deleteReq.Path = args[0]
		}

		err = w.Workspace.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start export command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var exportOverrides []func(
	*cobra.Command,
	*workspace.ExportRequest,
)

func newExport() *cobra.Command {
	cmd := &cobra.Command{}

	var exportReq workspace.ExportRequest

	cmd.Flags().Var(&exportReq.Format, "format", `This specifies the format of the exported file. Supported values: [
  AUTO,
  DBC,
  HTML,
  JUPYTER,
  RAW,
  R_MARKDOWN,
  SOURCE,
]`)

	cmd.Use = "export PATH"
	cmd.Short = `Export a workspace object.`
	cmd.Long = `Export a workspace object.
  
  Exports an object or the contents of an entire directory.
  
  If path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.
  
  If the exported data would exceed size limit, this call returns
  MAX_NOTEBOOK_SIZE_EXCEEDED. Currently, this API does not support exporting a
  library.

  Arguments:
    PATH: The absolute path of the object or directory. Exporting a directory is
      only supported for the DBC, SOURCE, and AUTO format.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No PATH argument specified. Loading names for Workspace drop-down."
			names, err := w.Workspace.ObjectInfoPathToObjectIdMap(ctx, workspace.ListWorkspaceRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Workspace drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The absolute path of the object or directory")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the absolute path of the object or directory")
		}
		exportReq.Path = args[0]

		response, err := w.Workspace.Export(ctx, exportReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range exportOverrides {
		fn(cmd, &exportReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*workspace.GetWorkspaceObjectPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq workspace.GetWorkspaceObjectPermissionLevelsRequest

	cmd.Use = "get-permission-levels WORKSPACE_OBJECT_TYPE WORKSPACE_OBJECT_ID"
	cmd.Short = `Get workspace object permission levels.`
	cmd.Long = `Get workspace object permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    WORKSPACE_OBJECT_TYPE: The workspace object type for which to get or manage permissions.
    WORKSPACE_OBJECT_ID: The workspace object for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionLevelsReq.WorkspaceObjectType = args[0]
		getPermissionLevelsReq.WorkspaceObjectId = args[1]

		response, err := w.Workspace.GetPermissionLevels(ctx, getPermissionLevelsReq)
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

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*workspace.GetWorkspaceObjectPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq workspace.GetWorkspaceObjectPermissionsRequest

	cmd.Use = "get-permissions WORKSPACE_OBJECT_TYPE WORKSPACE_OBJECT_ID"
	cmd.Short = `Get workspace object permissions.`
	cmd.Long = `Get workspace object permissions.
  
  Gets the permissions of a workspace object. Workspace objects can inherit
  permissions from their parent objects or root object.

  Arguments:
    WORKSPACE_OBJECT_TYPE: The workspace object type for which to get or manage permissions.
    WORKSPACE_OBJECT_ID: The workspace object for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionsReq.WorkspaceObjectType = args[0]
		getPermissionsReq.WorkspaceObjectId = args[1]

		response, err := w.Workspace.GetPermissions(ctx, getPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd, &getPermissionsReq)
	}

	return cmd
}

// start get-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getStatusOverrides []func(
	*cobra.Command,
	*workspace.GetStatusRequest,
)

func newGetStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var getStatusReq workspace.GetStatusRequest

	cmd.Use = "get-status PATH"
	cmd.Short = `Get status.`
	cmd.Long = `Get status.
  
  Gets the status of an object or a directory. If path does not exist, this
  call returns an error RESOURCE_DOES_NOT_EXIST.

  Arguments:
    PATH: The absolute path of the notebook or directory.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getStatusReq.Path = args[0]

		response, err := w.Workspace.GetStatus(ctx, getStatusReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getStatusOverrides {
		fn(cmd, &getStatusReq)
	}

	return cmd
}

// start import command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var importOverrides []func(
	*cobra.Command,
	*workspace.Import,
)

func newImport() *cobra.Command {
	cmd := &cobra.Command{}

	var importReq workspace.Import
	var importJson flags.JsonFlag

	cmd.Flags().Var(&importJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&importReq.Content, "content", importReq.Content, `The base64-encoded content.`)
	cmd.Flags().Var(&importReq.Format, "format", `This specifies the format of the file to be imported. Supported values: [
  AUTO,
  DBC,
  HTML,
  JUPYTER,
  RAW,
  R_MARKDOWN,
  SOURCE,
]`)
	cmd.Flags().Var(&importReq.Language, "language", `The language of the object. Supported values: [PYTHON, R, SCALA, SQL]`)
	cmd.Flags().BoolVar(&importReq.Overwrite, "overwrite", importReq.Overwrite, `The flag that specifies whether to overwrite existing object.`)

	cmd.Use = "import PATH"
	cmd.Short = `Import a workspace object.`
	cmd.Long = `Import a workspace object.
  
  Imports a workspace object (for example, a notebook or file) or the contents
  of an entire directory. If path already exists and overwrite is set to
  false, this call returns an error RESOURCE_ALREADY_EXISTS. To import a
  directory, you can use either the DBC format or the SOURCE format with the
  language field unset. To import a single file as SOURCE, you must set the
  language field.

  Arguments:
    PATH: The absolute path of the object or directory. Importing a directory is
      only supported for the DBC and SOURCE formats.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'path' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := importJson.Unmarshal(&importReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			importReq.Path = args[0]
		}

		err = w.Workspace.Import(ctx, importReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range importOverrides {
		fn(cmd, &importReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*workspace.ListWorkspaceRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq workspace.ListWorkspaceRequest

	cmd.Flags().Int64Var(&listReq.NotebooksModifiedAfter, "notebooks-modified-after", listReq.NotebooksModifiedAfter, `UTC timestamp in milliseconds.`)

	cmd.Use = "list PATH"
	cmd.Short = `List contents.`
	cmd.Long = `List contents.
  
  Lists the contents of a directory, or the object if it is not a directory. If
  the input path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.

  Arguments:
    PATH: The absolute path of the notebook or directory.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.Path = args[0]

		response := w.Workspace.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start mkdirs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var mkdirsOverrides []func(
	*cobra.Command,
	*workspace.Mkdirs,
)

func newMkdirs() *cobra.Command {
	cmd := &cobra.Command{}

	var mkdirsReq workspace.Mkdirs
	var mkdirsJson flags.JsonFlag

	cmd.Flags().Var(&mkdirsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "mkdirs PATH"
	cmd.Short = `Create a directory.`
	cmd.Long = `Create a directory.
  
  Creates the specified directory (and necessary parent directories if they do
  not exist). If there is an object (not a directory) at any prefix of the input
  path, this call returns an error RESOURCE_ALREADY_EXISTS.
  
  Note that if this operation fails it may have succeeded in creating some of
  the necessary parent directories.

  Arguments:
    PATH: The absolute path of the directory. If the parent directories do not
      exist, it will also create them. If the directory already exists, this
      command will do nothing and succeed.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'path' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := mkdirsJson.Unmarshal(&mkdirsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No PATH argument specified. Loading names for Workspace drop-down."
				names, err := w.Workspace.ObjectInfoPathToObjectIdMap(ctx, workspace.ListWorkspaceRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Workspace drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The absolute path of the directory")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the absolute path of the directory")
			}
			mkdirsReq.Path = args[0]
		}

		err = w.Workspace.Mkdirs(ctx, mkdirsReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range mkdirsOverrides {
		fn(cmd, &mkdirsReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*workspace.WorkspaceObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq workspace.WorkspaceObjectPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions WORKSPACE_OBJECT_TYPE WORKSPACE_OBJECT_ID"
	cmd.Short = `Set workspace object permissions.`
	cmd.Long = `Set workspace object permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their parent objects or root object.

  Arguments:
    WORKSPACE_OBJECT_TYPE: The workspace object type for which to get or manage permissions.
    WORKSPACE_OBJECT_ID: The workspace object for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		setPermissionsReq.WorkspaceObjectType = args[0]
		setPermissionsReq.WorkspaceObjectId = args[1]

		response, err := w.Workspace.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*workspace.WorkspaceObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq workspace.WorkspaceObjectPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions WORKSPACE_OBJECT_TYPE WORKSPACE_OBJECT_ID"
	cmd.Short = `Update workspace object permissions.`
	cmd.Long = `Update workspace object permissions.
  
  Updates the permissions on a workspace object. Workspace objects can inherit
  permissions from their parent objects or root object.

  Arguments:
    WORKSPACE_OBJECT_TYPE: The workspace object type for which to get or manage permissions.
    WORKSPACE_OBJECT_ID: The workspace object for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updatePermissionsReq.WorkspaceObjectType = args[0]
		updatePermissionsReq.WorkspaceObjectId = args[1]

		response, err := w.Workspace.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service Workspace
