// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package billable_usage

import (
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billable-usage",
		Short: `This API allows you to download billable usage logs for the specified account and date range.`,
		Long: `This API allows you to download billable usage logs for the specified account
  and date range. This feature works with all account types.`,
		GroupID: "billing",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newDownload())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start download command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var downloadOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newDownload() *cobra.Command {
	cmd := &cobra.Command{}
	var downloadReq map[string]any
	downloadReq = map[string]any{}
	var downloadEndMonthRaw string
	var downloadPersonalDataRaw bool
	var downloadStartMonthRaw string

	cmd.Flags().BoolVar(&downloadPersonalDataRaw, "personal-data", downloadPersonalDataRaw, `Specify whether to include personally identifiable information in the billable usage logs, for example the email addresses of cluster creators.`)

	cmd.Use = "download START_MONTH END_MONTH"
	cmd.Short = `Return billable usage logs.`
	cmd.Long = `Return billable usage logs.

  Returns billable usage logs in CSV format for the specified account and date
  range.

  Note that this method might take multiple minutes to complete.

  **Warning**: Depending on the queried date range, the number of workspaces in
  the account, the size of the response and the internet speed of the caller,
  this API may hit a timeout after a few minutes. If you experience this, try to
  mitigate by calling the API with narrower date ranges.

  Arguments:
    START_MONTH: Format specification for month in the format YYYY-MM. This is used to
      specify billable usage start_month and end_month properties. **Note**:
      Billable usage logs are unavailable before March 2019 (2019-03).
    END_MONTH: Format: YYYY-MM. Last month to return billable usage logs for. This
      field is required.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		downloadStartMonthRaw = args[0]
		downloadEndMonthRaw = args[1]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("end-month") || downloadEndMonthRaw != "" {
			rawQuery["end_month"] = downloadEndMonthRaw
		}
		if cmd.Flags().Changed("personal-data") || downloadPersonalDataRaw != false {
			rawQuery["personal_data"] = downloadPersonalDataRaw
		}
		if cmd.Flags().Changed("start-month") || downloadStartMonthRaw != "" {
			rawQuery["start_month"] = downloadStartMonthRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/accounts/%s/usage/download", a.Config.AccountID)

		maps.Copy(rawQuery, downloadReq)
		api, err := apiclient.New(a.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, nil, rawQuery, nil, &response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range downloadOverrides {
		fn(cmd, &downloadReq)
	}

	return cmd
}

// end service BillableUsage
