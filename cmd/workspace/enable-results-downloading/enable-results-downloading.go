// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package enable_results_downloading

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
		Use:   "enable-results-downloading",
		Short: `*Public Preview* Controls whether users can download notebook results.`,
		Long: `This command is in Public Preview and may change without notice.

Controls whether users can download notebook results. By default, this setting
  is enabled.`,
		RunE: root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newGetEnableResultsDownloading())
	cmd.AddCommand(newPatchEnableResultsDownloading())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-enable-results-downloading command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEnableResultsDownloadingOverrides []func(
	*cobra.Command,
)

func newGetEnableResultsDownloading() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-enable-results-downloading"
	cmd.Short = `*Public Preview* Get the Notebook results download setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get the Notebook results download setting.

  Gets the Notebook results download setting.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w
		getEnableResultsDownloadingReq := map[string]any{}
		rawQuery := map[string]any{}
		rawPath := "/api/2.0/settings/types/enable-results-downloading/names/default"

		maps.Copy(rawQuery, getEnableResultsDownloadingReq)
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
	for _, fn := range getEnableResultsDownloadingOverrides {
		fn(cmd)
	}

	return cmd
}

// start patch-enable-results-downloading command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchEnableResultsDownloadingOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newPatchEnableResultsDownloading() *cobra.Command {
	cmd := &cobra.Command{}
	var patchEnableResultsDownloadingReq map[string]any
	patchEnableResultsDownloadingReq = map[string]any{}
	var patchEnableResultsDownloadingAllowMissingRaw bool
	var patchEnableResultsDownloadingFieldMaskRaw string

	var patchEnableResultsDownloadingJson flags.JsonFlag

	cmd.Flags().Var(&patchEnableResultsDownloadingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "patch-enable-results-downloading"
	cmd.Short = `*Public Preview* Update the Notebook results download setting.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update the Notebook results download setting.

  Updates the Notebook results download setting. The model follows eventual
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
			diags := patchEnableResultsDownloadingJson.Unmarshal(&patchEnableResultsDownloadingReq)
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
		if cmd.Flags().Changed("allow-missing") || patchEnableResultsDownloadingAllowMissingRaw != false {
			patchEnableResultsDownloadingReq["allow_missing"] = patchEnableResultsDownloadingAllowMissingRaw
		}
		if cmd.Flags().Changed("field-mask") || patchEnableResultsDownloadingFieldMaskRaw != "" {
			patchEnableResultsDownloadingReq["field_mask"] = patchEnableResultsDownloadingFieldMaskRaw
		}
		rawPath := "/api/2.0/settings/types/enable-results-downloading/names/default"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, patchEnableResultsDownloadingReq, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchEnableResultsDownloadingOverrides {
		fn(cmd, &patchEnableResultsDownloadingReq)
	}

	return cmd
}

// end service EnableResultsDownloading
