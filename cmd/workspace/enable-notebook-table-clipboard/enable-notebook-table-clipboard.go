// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package enable_notebook_table_clipboard

import (
	"errors"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable-notebook-table-clipboard",
		Short: `*Public Preview* Controls whether users can copy tabular data to the clipboard via the UI.`,
		Long: `This command is in Public Preview and may change without notice.

Controls whether users can copy tabular data to the clipboard via the UI. By
  default, this setting is enabled.`,
		RunE: root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	cmd.Short = `*Public Preview* Get the Results Table Clipboard features setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get the Results Table Clipboard features setting.

  Gets the Results Table Clipboard features setting.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w
		getEnableNotebookTableClipboardReq := map[string]any{}
		rawQuery := map[string]any{}
		rawPath := "/api/2.0/settings/types/enable-notebook-table-clipboard/names/default"

		maps.Copy(rawQuery, getEnableNotebookTableClipboardReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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
	*map[string]any,
)

func newPatchEnableNotebookTableClipboard() *cobra.Command {
	cmd := &cobra.Command{}
	var patchEnableNotebookTableClipboardReq map[string]any
	patchEnableNotebookTableClipboardReq = map[string]any{}
	var patchEnableNotebookTableClipboardAllowMissingRaw bool
	var patchEnableNotebookTableClipboardFieldMaskRaw string

	var patchEnableNotebookTableClipboardJson flags.JsonFlag

	cmd.Flags().Var(&patchEnableNotebookTableClipboardJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "patch-enable-notebook-table-clipboard"
	cmd.Short = `*Public Preview* Update the Results Table Clipboard features setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update the Results Table Clipboard features setting.

  Updates the Results Table Clipboard features setting. The model follows
  eventual consistency, which means the get after the update operation might
  receive stale values for some time.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := patchEnableNotebookTableClipboardJson.Unmarshal(&patchEnableNotebookTableClipboardReq)
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

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("allow-missing") || patchEnableNotebookTableClipboardAllowMissingRaw != false {
			patchEnableNotebookTableClipboardReq["allow_missing"] = patchEnableNotebookTableClipboardAllowMissingRaw
		}
		if cmd.Flags().Changed("field-mask") || patchEnableNotebookTableClipboardFieldMaskRaw != "" {
			patchEnableNotebookTableClipboardReq["field_mask"] = patchEnableNotebookTableClipboardFieldMaskRaw
		}
		rawPath := "/api/2.0/settings/types/enable-notebook-table-clipboard/names/default"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, patchEnableNotebookTableClipboardReq, &response)
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
