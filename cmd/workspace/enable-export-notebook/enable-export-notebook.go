// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package enable_export_notebook

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
		Use:   "enable-export-notebook",
		Short: `*Public Preview* Controls whether users can export notebooks and files from the Workspace UI.`,
		Long: `This command is in Public Preview and may change without notice.

Controls whether users can export notebooks and files from the Workspace UI.
  By default, this setting is enabled.`,
		RunE: root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

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
	cmd.Short = `*Public Preview* Get the Notebook and File exporting setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get the Notebook and File exporting setting.

  Gets the Notebook and File exporting setting.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w
		getEnableExportNotebookReq := map[string]any{}
		rawQuery := map[string]any{}
		rawPath := "/api/2.0/settings/types/enable-export-notebook/names/default"

		maps.Copy(rawQuery, getEnableExportNotebookReq)
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
	*map[string]any,
)

func newPatchEnableExportNotebook() *cobra.Command {
	cmd := &cobra.Command{}
	var patchEnableExportNotebookReq map[string]any
	patchEnableExportNotebookReq = map[string]any{}
	var patchEnableExportNotebookAllowMissingRaw bool
	var patchEnableExportNotebookFieldMaskRaw string

	var patchEnableExportNotebookJson flags.JsonFlag

	cmd.Flags().Var(&patchEnableExportNotebookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "patch-enable-export-notebook"
	cmd.Short = `*Public Preview* Update the Notebook and File exporting setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update the Notebook and File exporting setting.

  Updates the Notebook and File exporting setting. The model follows eventual
  consistency, which means the get after the update operation might receive
  stale values for some time.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := patchEnableExportNotebookJson.Unmarshal(&patchEnableExportNotebookReq)
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
		if cmd.Flags().Changed("allow-missing") || patchEnableExportNotebookAllowMissingRaw != false {
			patchEnableExportNotebookReq["allow_missing"] = patchEnableExportNotebookAllowMissingRaw
		}
		if cmd.Flags().Changed("field-mask") || patchEnableExportNotebookFieldMaskRaw != "" {
			patchEnableExportNotebookReq["field_mask"] = patchEnableExportNotebookFieldMaskRaw
		}
		rawPath := "/api/2.0/settings/types/enable-export-notebook/names/default"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, patchEnableExportNotebookReq, &response)
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
