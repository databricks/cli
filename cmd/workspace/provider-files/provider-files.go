// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package provider_files

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/marketplace"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-files",
		Short: `Marketplace offers a set of file APIs for various purposes such as preview notebooks and provider icons.`,
		Long: `Marketplace offers a set of file APIs for various purposes such as preview
  notebooks and provider icons.`,
		GroupID: "marketplace",
		Annotations: map[string]string{
			"package": "marketplace",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*marketplace.CreateFileRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq marketplace.CreateFileRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.DisplayName, "display-name", createReq.DisplayName, ``)

	cmd.Use = "create"
	cmd.Short = `Create a file.`
	cmd.Long = `Create a file.
  
  Create a file. Currently, only provider icons and attached notebooks are
  supported.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.ProviderFiles.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*marketplace.DeleteFileRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq marketplace.DeleteFileRequest

	cmd.Use = "delete FILE_ID"
	cmd.Short = `Delete a file.`
	cmd.Long = `Delete a file.
  
  Delete a file`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FILE_ID argument specified. Loading names for Provider Files drop-down."
			names, err := w.ProviderFiles.FileInfoDisplayNameToIdMap(ctx, marketplace.ListFilesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Provider Files drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		deleteReq.FileId = args[0]

		err = w.ProviderFiles.Delete(ctx, deleteReq)
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*marketplace.GetFileRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq marketplace.GetFileRequest

	cmd.Use = "get FILE_ID"
	cmd.Short = `Get a file.`
	cmd.Long = `Get a file.
  
  Get a file`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FILE_ID argument specified. Loading names for Provider Files drop-down."
			names, err := w.ProviderFiles.FileInfoDisplayNameToIdMap(ctx, marketplace.ListFilesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Provider Files drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		getReq.FileId = args[0]

		response, err := w.ProviderFiles.Get(ctx, getReq)
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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*marketplace.ListFilesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq marketplace.ListFilesRequest
	var listJson flags.JsonFlag

	cmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, ``)

	cmd.Use = "list"
	cmd.Short = `List files.`
	cmd.Long = `List files.
  
  List files attached to a parent entity.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := listJson.Unmarshal(&listReq)
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
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response := w.ProviderFiles.List(ctx, listReq)
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

// end service ProviderFiles
