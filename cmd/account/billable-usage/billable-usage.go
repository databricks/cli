// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package billable_usage

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/billing"
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
		Annotations: map[string]string{
			"package": "billing",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*billing.DownloadRequest,
)

func newDownload() *cobra.Command {
	cmd := &cobra.Command{}

	var downloadReq billing.DownloadRequest

	cmd.Flags().BoolVar(&downloadReq.PersonalData, "personal-data", downloadReq.PersonalData, `Specify whether to include personally identifiable information in the billable usage logs, for example the email addresses of cluster creators.`)

	cmd.Use = "download START_MONTH END_MONTH"
	cmd.Short = `Return billable usage logs.`
	cmd.Long = `Return billable usage logs.
  
  Returns billable usage logs in CSV format for the specified account and date
  range. For the data schema, see:
  
  - AWS: [CSV file schema]. - GCP: [CSV file schema].
  
  Note that this method might take multiple minutes to complete.
  
  **Warning**: Depending on the queried date range, the number of workspaces in
  the account, the size of the response and the internet speed of the caller,
  this API may hit a timeout after a few minutes. If you experience this, try to
  mitigate by calling the API with narrower date ranges.
  
  [CSV file schema]: https://docs.gcp.databricks.com/administration-guide/account-settings/usage-analysis.html#csv-file-schema

  Arguments:
    START_MONTH: Format specification for month in the format YYYY-MM. This is used to
      specify billable usage start_month and end_month properties. **Note**:
      Billable usage logs are unavailable before March 2019 (2019-03).
    END_MONTH: Format: YYYY-MM. Last month to return billable usage logs for. This
      field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		downloadReq.StartMonth = args[0]
		downloadReq.EndMonth = args[1]

		response, err := a.BillableUsage.Download(ctx, downloadReq)
		if err != nil {
			return err
		}
		defer response.Contents.Close()
		return cmdio.Render(ctx, response.Contents)
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
