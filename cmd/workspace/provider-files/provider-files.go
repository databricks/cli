// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package provider_files

import (
	"context"
	"errors"
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

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "provider-files",
	Short:   `Marketplace offers a set of file APIs for various purposes such as preview notebooks and provider icons.`,
	GroupID: "marketplace",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Marketplace offers a set of file APIs for various purposes such as preview
  notebooks and provider icons.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "GA",
		"launch_stage_display": "GA",
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

	cmd.Flags().StringVar(&createReq.DisplayName, "display-name", createReq.DisplayName, `Name shown to consumers for the file, for example a notebook title.`)

	cmd.Use = "create"
	cmd.Short = `Create a file.`
	cmd.Long = `Create a file.

  Creates a file and returns a short-lived URL for uploading its content with an
  HTTP PUT request. Only provider icons (PROVIDER_ICON files on a PROVIDER
  parent) and sample notebooks (EMBEDDED_NOTEBOOK files on a LISTING parent) are
  supported.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
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

  Deletes a file. Fails if the file is still in use.

  Arguments:
    FILE_ID: ID of the file to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No FILE_ID argument specified. Loading names for Provider Files drop-down.")
			names, err := w.ProviderFiles.FileInfoDisplayNameToIdMap(ctx, marketplace.ListFilesRequest{})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Provider Files drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "ID of the file to delete")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have id of the file to delete")
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

  Gets a file's metadata, including a short-lived download URL.

  Arguments:
    FILE_ID: ID of the file.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No FILE_ID argument specified. Loading names for Provider Files drop-down.")
			names, err := w.ProviderFiles.FileInfoDisplayNameToIdMap(ctx, marketplace.ListFilesRequest{})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Provider Files drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "ID of the file")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have id of the file")
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
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	cmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Maximum number of results to return per page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List files.`
	cmd.Long = `List files.

  Lists the files attached to a provider profile or a listing.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		response := w.ProviderFiles.List(ctx, listReq)
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

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
