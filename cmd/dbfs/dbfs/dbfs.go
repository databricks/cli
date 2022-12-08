package dbfs

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/dbfs"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "dbfs",
	Short: `DBFS API makes it simple to interact with various data sources without having to include a users credentials every time to read a file.`,
}

var addBlockReq dbfs.AddBlock

func init() {
	Cmd.AddCommand(addBlockCmd)
	// TODO: short flags

	addBlockCmd.Flags().StringVar(&addBlockReq.Data, "data", "", `The base64-encoded data to append to the stream.`)
	addBlockCmd.Flags().Int64Var(&addBlockReq.Handle, "handle", 0, `The handle on an open stream.`)

}

var addBlockCmd = &cobra.Command{
	Use:   "add-block",
	Short: `Append data block.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Dbfs.AddBlock(ctx, addBlockReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var closeReq dbfs.Close

func init() {
	Cmd.AddCommand(closeCmd)
	// TODO: short flags

	closeCmd.Flags().Int64Var(&closeReq.Handle, "handle", 0, `The handle on an open stream.`)

}

var closeCmd = &cobra.Command{
	Use:   "close",
	Short: `Close the stream.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Dbfs.Close(ctx, closeReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var createReq dbfs.Create

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.Overwrite, "overwrite", false, `The flag that specifies whether to overwrite existing file/files.`)
	createCmd.Flags().StringVar(&createReq.Path, "path", "", `The path of the new file.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Open a stream.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Dbfs.Create(ctx, createReq)
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

var deleteReq dbfs.Delete

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Path, "path", "", `The path of the file or directory to delete.`)
	deleteCmd.Flags().BoolVar(&deleteReq.Recursive, "recursive", false, `Whether or not to recursively delete the directory's contents.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a file/directory.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Dbfs.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getStatusReq dbfs.GetStatus

func init() {
	Cmd.AddCommand(getStatusCmd)
	// TODO: short flags

	getStatusCmd.Flags().StringVar(&getStatusReq.Path, "path", "", `The path of the file or directory.`)

}

var getStatusCmd = &cobra.Command{
	Use:   "get-status",
	Short: `Get the information of a file or directory.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Dbfs.GetStatus(ctx, getStatusReq)
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

var listReq dbfs.List

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.Path, "path", "", `The path of the file or directory.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List directory contents or file details.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Dbfs.ListAll(ctx, listReq)
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

var mkdirsReq dbfs.MkDirs

func init() {
	Cmd.AddCommand(mkdirsCmd)
	// TODO: short flags

	mkdirsCmd.Flags().StringVar(&mkdirsReq.Path, "path", "", `The path of the new directory.`)

}

var mkdirsCmd = &cobra.Command{
	Use:   "mkdirs",
	Short: `Create a directory.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Dbfs.Mkdirs(ctx, mkdirsReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var moveReq dbfs.Move

func init() {
	Cmd.AddCommand(moveCmd)
	// TODO: short flags

	moveCmd.Flags().StringVar(&moveReq.DestinationPath, "destination-path", "", `The destination path of the file or directory.`)
	moveCmd.Flags().StringVar(&moveReq.SourcePath, "source-path", "", `The source path of the file or directory.`)

}

var moveCmd = &cobra.Command{
	Use:   "move",
	Short: `Move a file.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Dbfs.Move(ctx, moveReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var putReq dbfs.Put

func init() {
	Cmd.AddCommand(putCmd)
	// TODO: short flags

	putCmd.Flags().StringVar(&putReq.Contents, "contents", "", `This parameter might be absent, and instead a posted file will be used.`)
	putCmd.Flags().BoolVar(&putReq.Overwrite, "overwrite", false, `The flag that specifies whether to overwrite existing file/files.`)
	putCmd.Flags().StringVar(&putReq.Path, "path", "", `The path of the new file.`)

}

var putCmd = &cobra.Command{
	Use:   "put",
	Short: `Upload a file.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Dbfs.Put(ctx, putReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var readReq dbfs.Read

func init() {
	Cmd.AddCommand(readCmd)
	// TODO: short flags

	readCmd.Flags().IntVar(&readReq.Length, "length", 0, `The number of bytes to read starting from the offset.`)
	readCmd.Flags().IntVar(&readReq.Offset, "offset", 0, `The offset to read from in bytes.`)
	readCmd.Flags().StringVar(&readReq.Path, "path", "", `The path of the file to read.`)

}

var readCmd = &cobra.Command{
	Use:   "read",
	Short: `Get the contents of a file.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Dbfs.Read(ctx, readReq)
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
