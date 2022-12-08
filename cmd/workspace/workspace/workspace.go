package workspace

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
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

var deleteReq workspace.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Path, "path", "", `The absolute path of the notebook or directory.`)
	deleteCmd.Flags().BoolVar(&deleteReq.Recursive, "recursive", false, `The flag that specifies whether to delete the object recursively.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a workspace object.`,
	Long: `Delete a workspace object.
  
  Deletes an object or a directory (and optionally recursively deletes all
  objects in the directory). * If path does not exist, this call returns an
  error RESOURCE_DOES_NOT_EXIST. * If path is a non-empty directory and
  recursive is set to false, this call returns an error
  DIRECTORY_NOT_EMPTY.
  
  Object deletion cannot be undone and deleting a directory recursively is not
  atomic.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Workspace.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var exportReq workspace.Export

func init() {
	Cmd.AddCommand(exportCmd)
	// TODO: short flags

	exportCmd.Flags().BoolVar(&exportReq.DirectDownload, "direct-download", false, `Flag to enable direct download.`)
	exportCmd.Flags().Var(&exportReq.Format, "format", `This specifies the format of the exported file.`)
	exportCmd.Flags().StringVar(&exportReq.Path, "path", "", `The absolute path of the notebook or directory.`)

}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: `Export a notebook.`,
	Long: `Export a notebook.
  
  Exports a notebook or the contents of an entire directory.
  
  If path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.
  
  One can only export a directory in DBC format. If the exported data would
  exceed size limit, this call returns MAX_NOTEBOOK_SIZE_EXCEEDED. Currently,
  this API does not support exporting a library.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Workspace.Export(ctx, exportReq)
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

var getStatusReq workspace.GetStatus

func init() {
	Cmd.AddCommand(getStatusCmd)
	// TODO: short flags

	getStatusCmd.Flags().StringVar(&getStatusReq.Path, "path", "", `The absolute path of the notebook or directory.`)

}

var getStatusCmd = &cobra.Command{
	Use:   "get-status",
	Short: `Get status.`,
	Long: `Get status.
  
  Gets the status of an object or a directory. If path does not exist, this
  call returns an error RESOURCE_DOES_NOT_EXIST.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Workspace.GetStatus(ctx, getStatusReq)
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

var importReq workspace.Import

func init() {
	Cmd.AddCommand(importCmd)
	// TODO: short flags

	importCmd.Flags().StringVar(&importReq.Content, "content", "", `The base64-encoded content.`)
	importCmd.Flags().Var(&importReq.Format, "format", `This specifies the format of the file to be imported.`)
	importCmd.Flags().Var(&importReq.Language, "language", `The language of the object.`)
	importCmd.Flags().BoolVar(&importReq.Overwrite, "overwrite", false, `The flag that specifies whether to overwrite existing object.`)
	importCmd.Flags().StringVar(&importReq.Path, "path", "", `The absolute path of the notebook or directory.`)

}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: `Import a notebook.`,
	Long: `Import a notebook.
  
  Imports a notebook or the contents of an entire directory. If path already
  exists and overwrite is set to false, this call returns an error
  RESOURCE_ALREADY_EXISTS. One can only use DBC format to import a
  directory.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Workspace.Import(ctx, importReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var listReq workspace.List

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().IntVar(&listReq.NotebooksModifiedAfter, "notebooks-modified-after", 0, `<content needed>.`)
	listCmd.Flags().StringVar(&listReq.Path, "path", "", `The absolute path of the notebook or directory.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List contents.`,
	Long: `List contents.
  
  Lists the contents of a directory, or the object if it is not a directory.If
  the input path does not exist, this call returns an error
  RESOURCE_DOES_NOT_EXIST.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Workspace.ListAll(ctx, listReq)
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

var mkdirsReq workspace.Mkdirs

func init() {
	Cmd.AddCommand(mkdirsCmd)
	// TODO: short flags

	mkdirsCmd.Flags().StringVar(&mkdirsReq.Path, "path", "", `The absolute path of the directory.`)

}

var mkdirsCmd = &cobra.Command{
	Use:   "mkdirs",
	Short: `Create a directory.`,
	Long: `Create a directory.
  
  Creates the specified directory (and necessary parent directories if they do
  not exist). If there is an object (not a directory) at any prefix of the input
  path, this call returns an error RESOURCE_ALREADY_EXISTS.
  
  Note that if this operation fails it may have succeeded in creating some of
  the necessary\nparrent directories.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Workspace.Mkdirs(ctx, mkdirsReq)
		if err != nil {
			return err
		}

		return nil
	},
}
