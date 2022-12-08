package workspace

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "workspace",
	Short: `The Workspace API allows you to list, import, export, and delete notebooks and folders.`,
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Workspace.Mkdirs(ctx, mkdirsReq)
		if err != nil {
			return err
		}

		return nil
	},
}
