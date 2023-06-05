// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package workspace

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace",
	Short: `The Workspace API allows you to list, import, export, and delete notebooks and folders.`,
	Long: `The Workspace API allows you to list, import, export, and delete notebooks and
  folders.
  
  A notebook is a web-based interface to a document that contains runnable code,
  visualizations, and explanatory text.`,
}

// start delete command

var deleteReq workspace.Delete
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	deleteCmd.Flags().BoolVar(&deleteReq.Recursive, "recursive", deleteReq.Recursive, `The flag that specifies whether to delete the object recursively.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete PATH",
	Short: `Delete a workspace object.`,
	Long: `Delete a workspace object.
  
  Deletes an object or a directory (and optionally recursively deletes all
  objects in the directory). * If path does not exist, this call returns an
  error RESOURCE_DOES_NOT_EXIST. * If path is a non-empty directory and
  recursive is set to false, this call returns an error
  DIRECTORY_NOT_EMPTY.
  
  Object deletion cannot be undone and deleting a directory recursively is not
  atomic.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
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
	},
}

// start export command

var exportReq workspace.ExportRequest
var exportJson flags.JsonFlag

func init() {
	Cmd.AddCommand(exportCmd)
	// TODO: short flags
	exportCmd.Flags().Var(&exportJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	exportCmd.Flags().BoolVar(&exportReq.DirectDownload, "direct-download", exportReq.DirectDownload, `Flag to enable direct download.`)
	exportCmd.Flags().Var(&exportReq.Format, "format", `This specifies the format of the exported file.`)

}

var exportCmd = &cobra.Command{
	Use:   "export PATH",
	Short: `Export a workspace object.`,
	Long: `Export a workspace object.
  
  Exports an object or the contents of an entire directory.
  
  If path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.
  
  One can only export a directory in DBC format. If the exported data would
  exceed size limit, this call returns MAX_NOTEBOOK_SIZE_EXCEEDED. Currently,
  this API does not support exporting a library.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = exportJson.Unmarshal(&exportReq)
			if err != nil {
				return err
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
		}

		response, err := w.Workspace.Export(ctx, exportReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get-status command

var getStatusReq workspace.GetStatusRequest
var getStatusJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getStatusCmd)
	// TODO: short flags
	getStatusCmd.Flags().Var(&getStatusJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getStatusCmd = &cobra.Command{
	Use:   "get-status PATH",
	Short: `Get status.`,
	Long: `Get status.
  
  Gets the status of an object or a directory. If path does not exist, this
  call returns an error RESOURCE_DOES_NOT_EXIST.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getStatusJson.Unmarshal(&getStatusReq)
			if err != nil {
				return err
			}
		} else {
			getStatusReq.Path = args[0]
		}

		response, err := w.Workspace.GetStatus(ctx, getStatusReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start import command

var importReq workspace.Import
var importJson flags.JsonFlag

func init() {
	Cmd.AddCommand(importCmd)
	// TODO: short flags
	importCmd.Flags().Var(&importJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	importCmd.Flags().StringVar(&importReq.Content, "content", importReq.Content, `The base64-encoded content.`)
	importCmd.Flags().Var(&importReq.Format, "format", `This specifies the format of the file to be imported.`)
	importCmd.Flags().Var(&importReq.Language, "language", `The language of the object.`)
	importCmd.Flags().BoolVar(&importReq.Overwrite, "overwrite", importReq.Overwrite, `The flag that specifies whether to overwrite existing object.`)

}

var importCmd = &cobra.Command{
	Use:   "import PATH",
	Short: `Import a workspace object.`,
	Long: `Import a workspace object.
  
  Imports a workspace object (for example, a notebook or file) or the contents
  of an entire directory. If path already exists and overwrite is set to
  false, this call returns an error RESOURCE_ALREADY_EXISTS. One can only
  use DBC format to import a directory.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = importJson.Unmarshal(&importReq)
			if err != nil {
				return err
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
				id, err := cmdio.Select(ctx, names, "The absolute path of the object or directory")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the absolute path of the object or directory")
			}
			importReq.Path = args[0]
		}

		err = w.Workspace.Import(ctx, importReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start list command

var listReq workspace.ListWorkspaceRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().IntVar(&listReq.NotebooksModifiedAfter, "notebooks-modified-after", listReq.NotebooksModifiedAfter, `<content needed>.`)

}

var listCmd = &cobra.Command{
	Use:   "list PATH",
	Short: `List contents.`,
	Long: `List contents.
  
  Lists the contents of a directory, or the object if it is not a directory.If
  the input path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
			listReq.Path = args[0]
		}

		response, err := w.Workspace.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start mkdirs command

var mkdirsReq workspace.Mkdirs
var mkdirsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(mkdirsCmd)
	// TODO: short flags
	mkdirsCmd.Flags().Var(&mkdirsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var mkdirsCmd = &cobra.Command{
	Use:   "mkdirs PATH",
	Short: `Create a directory.`,
	Long: `Create a directory.
  
  Creates the specified directory (and necessary parent directories if they do
  not exist). If there is an object (not a directory) at any prefix of the input
  path, this call returns an error RESOURCE_ALREADY_EXISTS.
  
  Note that if this operation fails it may have succeeded in creating some of
  the necessary parrent directories.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = mkdirsJson.Unmarshal(&mkdirsReq)
			if err != nil {
				return err
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
	},
}

// end service Workspace
