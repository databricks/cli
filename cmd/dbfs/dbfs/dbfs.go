package dbfs

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
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
	Long: `Append data block.
  
  Appends a block of data to the stream specified by the input handle. If the
  handle does not exist, this call will throw an exception with
  RESOURCE_DOES_NOT_EXIST.
  
  If the block of data exceeds 1 MB, this call will throw an exception with
  MAX_BLOCK_SIZE_EXCEEDED.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Close the stream.
  
  Closes the stream specified by the input handle. If the handle does not exist,
  this call throws an exception with RESOURCE_DOES_NOT_EXIST.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Open a stream.
  
  "Opens a stream to write to a file and returns a handle to this stream. There
  is a 10 minute idle timeout on this handle. If a file or directory already
  exists on the given path and __overwrite__ is set to false, this call throws
  an exception with RESOURCE_ALREADY_EXISTS.
  
  A typical workflow for file upload would be:
  
  1. Issue a create call and get a handle. 2. Issue one or more add-block
  calls with the handle you have. 3. Issue a close call with the handle you
  have.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Delete a file/directory.
  
  Delete the file or directory (optionally recursively delete all files in the
  directory). This call throws an exception with IO_ERROR if the path is a
  non-empty directory and recursive is set to false or on other similar
  errors.
  
  When you delete a large number of files, the delete operation is done in
  increments. The call returns a response after approximately 45 seconds with an
  error message (503 Service Unavailable) asking you to re-invoke the delete
  operation until the directory structure is fully deleted.
  
  For operations that delete more than 10K files, we discourage using the DBFS
  REST API, but advise you to perform such operations in the context of a
  cluster, using the [File system utility
  (dbutils.fs)](/dev-tools/databricks-utils.html#dbutils-fs). dbutils.fs
  covers the functional scope of the DBFS REST API, but from notebooks. Running
  such operations using notebooks provides better control and manageability,
  such as selective deletes, and the possibility to automate periodic delete
  jobs.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Get the information of a file or directory.
  
  Gets the file information for a file or directory. If the file or directory
  does not exist, this call throws an exception with RESOURCE_DOES_NOT_EXIST.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `List directory contents or file details.
  
  List the contents of a directory, or details of the file. If the file or
  directory does not exist, this call throws an exception with
  RESOURCE_DOES_NOT_EXIST.
  
  When calling list on a large directory, the list operation will time out after
  approximately 60 seconds. We strongly recommend using list only on directories
  containing less than 10K files and discourage using the DBFS REST API for
  operations that list more than 10K files. Instead, we recommend that you
  perform such operations in the context of a cluster, using the [File system
  utility (dbutils.fs)](/dev-tools/databricks-utils.html#dbutils-fs), which
  provides the same functionality without timing out.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Create a directory.
  
  Creates the given directory and necessary parent directories if they do not
  exist. If a file (not a directory) exists at any prefix of the input path,
  this call throws an exception with RESOURCE_ALREADY_EXISTS. **Note**: If
  this operation fails, it might have succeeded in creating some of the
  necessary parent directories.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Move a file.
  
  Moves a file from one location to another location within DBFS. If the source
  file does not exist, this call throws an exception with
  RESOURCE_DOES_NOT_EXIST. If a file already exists in the destination path,
  this call throws an exception with RESOURCE_ALREADY_EXISTS. If the given
  source path is a directory, this call always recursively moves all files.",`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Upload a file.
  
  Uploads a file through the use of multipart form post. It is mainly used for
  streaming uploads, but can also be used as a convenient single call for data
  upload.
  
  Alternatively you can pass contents as base64 string.
  
  The amount of data that can be passed (when not streaming) using the
  __contents__ parameter is limited to 1 MB. MAX_BLOCK_SIZE_EXCEEDED will be
  thrown if this limit is exceeded.
  
  If you want to upload large files, use the streaming upload. For details, see
  :method:create, :method:addBlock, :method:close.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Get the contents of a file.
  
  "Returns the contents of a file. If the file does not exist, this call throws
  an exception with RESOURCE_DOES_NOT_EXIST. If the path is a directory, the
  read length is negative, or if the offset is negative, this call throws an
  exception with INVALID_PARAMETER_VALUE. If the read length exceeds 1 MB,
  this call throws an\nexception with MAX_READ_SIZE_EXCEEDED.
  
  If offset + length exceeds the number of bytes in a file, it reads the
  contents until the end of file.",`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
