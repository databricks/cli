// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package enable_results_downloading

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
		Use:   "enable-results-downloading",
		Short: `Controls whether users can download notebook results.`,
		Long: `Controls whether users can download notebook results. By default, this setting
  is enabled.`,
		RunE: root.ReportUnknownSubcommand,
	}

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
	cmd.Short = `Get the Notebook results download setting.`
	cmd.Long = `Get the Notebook results download setting.
  
  Gets the Notebook results download setting.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.Settings.EnableResultsDownloading().GetEnableResultsDownloading(ctx)
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
	*settings.UpdateEnableResultsDownloadingRequest,
)

func newPatchEnableResultsDownloading() *cobra.Command {
	cmd := &cobra.Command{}

	var patchEnableResultsDownloadingReq settings.UpdateEnableResultsDownloadingRequest
	var patchEnableResultsDownloadingJson flags.JsonFlag

	cmd.Flags().Var(&patchEnableResultsDownloadingJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "patch-enable-results-downloading"
	cmd.Short = `Update the Notebook results download setting.`
	cmd.Long = `Update the Notebook results download setting.
  
  Updates the Notebook results download setting. The model follows eventual
  consistency, which means the get after the update operation might receive
  stale values for some time.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := patchEnableResultsDownloadingJson.Unmarshal(&patchEnableResultsDownloadingReq)
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

		response, err := w.Settings.EnableResultsDownloading().PatchEnableResultsDownloading(ctx, patchEnableResultsDownloadingReq)
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
