// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

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
	}

	cmd.AddCommand(newDelete())
	cmd.AddCommand(newExport())
	cmd.AddCommand(newGetStatus())
	cmd.AddCommand(newImport())
	cmd.AddCommand(newList())
	cmd.AddCommand(newMkdirs())

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

	// TODO: short flags
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
  atomic.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
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

	// TODO: short flags

	cmd.Flags().Var(&exportReq.Format, "format", `This specifies the format of the exported file.`)

	cmd.Use = "export PATH"
	cmd.Short = `Export a workspace object.`
	cmd.Long = `Export a workspace object.
  
  Exports an object or the contents of an entire directory.
  
  If path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.
  
  If the exported data would exceed size limit, this call returns
  MAX_NOTEBOOK_SIZE_EXCEEDED. Currently, this API does not support exporting a
  library.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

	// TODO: short flags

	cmd.Use = "get-status PATH"
	cmd.Short = `Get status.`
	cmd.Long = `Get status.
  
  Gets the status of an object or a directory. If path does not exist, this
  call returns an error RESOURCE_DOES_NOT_EXIST.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

	// TODO: short flags
	cmd.Flags().Var(&importJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&importReq.Content, "content", importReq.Content, `The base64-encoded content.`)
	cmd.Flags().Var(&importReq.Format, "format", `This specifies the format of the file to be imported.`)
	cmd.Flags().Var(&importReq.Language, "language", `The language of the object.`)
	cmd.Flags().BoolVar(&importReq.Overwrite, "overwrite", importReq.Overwrite, `The flag that specifies whether to overwrite existing object.`)

	cmd.Use = "import PATH"
	cmd.Short = `Import a workspace object.`
	cmd.Long = `Import a workspace object.
  
  Imports a workspace object (for example, a notebook or file) or the contents
  of an entire directory. If path already exists and overwrite is set to
  false, this call returns an error RESOURCE_ALREADY_EXISTS. One can only
  use DBC format to import a directory.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = importJson.Unmarshal(&importReq)
			if err != nil {
				return err
			}
		} else {
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

	// TODO: short flags

	cmd.Flags().IntVar(&listReq.NotebooksModifiedAfter, "notebooks-modified-after", listReq.NotebooksModifiedAfter, `UTC timestamp in milliseconds.`)

	cmd.Use = "list PATH"
	cmd.Short = `List contents.`
	cmd.Long = `List contents.
  
  Lists the contents of a directory, or the object if it is not a directory. If
  the input path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		listReq.Path = args[0]

		response, err := w.Workspace.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

	// TODO: short flags
	cmd.Flags().Var(&mkdirsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "mkdirs PATH"
	cmd.Short = `Create a directory.`
	cmd.Long = `Create a directory.
  
  Creates the specified directory (and necessary parent directories if they do
  not exist). If there is an object (not a directory) at any prefix of the input
  path, this call returns an error RESOURCE_ALREADY_EXISTS.
  
  Note that if this operation fails it may have succeeded in creating some of
  the necessary parent directories.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = mkdirsJson.Unmarshal(&mkdirsReq)
			if err != nil {
				return err
			}
		} else {
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

// end service Workspace
