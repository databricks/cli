// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package enable_notebook_table_clipboard

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable-notebook-table-clipboard",
		Short: `Controls whether users can copy tabular data to the clipboard via the UI.`,
		Long: `Controls whether users can copy tabular data to the clipboard via the UI. By
  default, this setting is enabled.`,
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGetEnableNotebookTableClipboard())
	cmd.AddCommand(newPatchEnableNotebookTableClipboard())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-enable-notebook-table-clipboard command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEnableNotebookTableClipboardOverrides []func(
	*cobra.Command,
)

func newGetEnableNotebookTableClipboard() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-enable-notebook-table-clipboard"
	cmd.Short = `Get the Results Table Clipboard features setting.`
	cmd.Long = `Get the Results Table Clipboard features setting.
  
  Gets the Results Table Clipboard features setting.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.Settings.EnableNotebookTableClipboard().GetEnableNotebookTableClipboard(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEnableNotebookTableClipboardOverrides {
		fn(cmd)
	}

	return cmd
}

// start patch-enable-notebook-table-clipboard command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchEnableNotebookTableClipboardOverrides []func(
	*cobra.Command,
	*settings.UpdateEnableNotebookTableClipboardRequest,
)

func newPatchEnableNotebookTableClipboard() *cobra.Command {
	cmd := &cobra.Command{}

	var patchEnableNotebookTableClipboardReq settings.UpdateEnableNotebookTableClipboardRequest
	var patchEnableNotebookTableClipboardJson flags.JsonFlag

	cmd.Flags().Var(&patchEnableNotebookTableClipboardJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "patch-enable-notebook-table-clipboard"
	cmd.Short = `Update the Results Table Clipboard features setting.`
	cmd.Long = `Update the Results Table Clipboard features setting.
  
  Updates the Results Table Clipboard features setting. The model follows
  eventual consistency, which means the get after the update operation might
  receive stale values for some time.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := patchEnableNotebookTableClipboardJson.Unmarshal(&patchEnableNotebookTableClipboardReq)
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

		response, err := w.Settings.EnableNotebookTableClipboard().PatchEnableNotebookTableClipboard(ctx, patchEnableNotebookTableClipboardReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchEnableNotebookTableClipboardOverrides {
		fn(cmd, &patchEnableNotebookTableClipboardReq)
	}

	return cmd
}

// end service EnableNotebookTableClipboard
