// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package enable_export_notebook

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
		Use:   "enable-export-notebook",
		Short: `Controls whether users can export notebooks and files from the Workspace UI.`,
		Long: `Controls whether users can export notebooks and files from the Workspace UI.
  By default, this setting is enabled.`,
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGetEnableExportNotebook())
	cmd.AddCommand(newPatchEnableExportNotebook())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-enable-export-notebook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEnableExportNotebookOverrides []func(
	*cobra.Command,
)

func newGetEnableExportNotebook() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-enable-export-notebook"
	cmd.Short = `Get the Notebook and File exporting setting.`
	cmd.Long = `Get the Notebook and File exporting setting.
  
  Gets the Notebook and File exporting setting.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.Settings.EnableExportNotebook().GetEnableExportNotebook(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEnableExportNotebookOverrides {
		fn(cmd)
	}

	return cmd
}

// start patch-enable-export-notebook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchEnableExportNotebookOverrides []func(
	*cobra.Command,
	*settings.UpdateEnableExportNotebookRequest,
)

func newPatchEnableExportNotebook() *cobra.Command {
	cmd := &cobra.Command{}

	var patchEnableExportNotebookReq settings.UpdateEnableExportNotebookRequest
	var patchEnableExportNotebookJson flags.JsonFlag

	cmd.Flags().Var(&patchEnableExportNotebookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "patch-enable-export-notebook"
	cmd.Short = `Update the Notebook and File exporting setting.`
	cmd.Long = `Update the Notebook and File exporting setting.
  
  Updates the Notebook and File exporting setting. The model follows eventual
  consistency, which means the get after the update operation might receive
  stale values for some time.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := patchEnableExportNotebookJson.Unmarshal(&patchEnableExportNotebookReq)
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

		response, err := w.Settings.EnableExportNotebook().PatchEnableExportNotebook(ctx, patchEnableExportNotebookReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchEnableExportNotebookOverrides {
		fn(cmd, &patchEnableExportNotebookReq)
	}

	return cmd
}

// end service EnableExportNotebook
