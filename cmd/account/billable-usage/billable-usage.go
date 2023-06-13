// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package billable_usage

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "billable-usage",
	Short: `This API allows you to download billable usage logs for the specified account and date range.`,
	Long: `This API allows you to download billable usage logs for the specified account
  and date range. This feature works with all account types.`,
}

// start download command

var downloadReq billing.DownloadRequest
var downloadJson flags.JsonFlag

func init() {
	Cmd.AddCommand(downloadCmd)
	// TODO: short flags
	downloadCmd.Flags().Var(&downloadJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	downloadCmd.Flags().BoolVar(&downloadReq.PersonalData, "personal-data", downloadReq.PersonalData, `Specify whether to include personally identifiable information in the billable usage logs, for example the email addresses of cluster creators.`)

}

var downloadCmd = &cobra.Command{
	Use:   "download START_MONTH END_MONTH",
	Short: `Return billable usage logs.`,
	Long: `Return billable usage logs.
  
  Returns billable usage logs in CSV format for the specified account and date
  range. For the data schema, see [CSV file schema]. Note that this method might
  take multiple seconds to complete.
  
  [CSV file schema]: https://docs.databricks.com/administration-guide/account-settings/usage-analysis.html#schema`,

	Annotations: map[string]string{
		"package": "billing",
	},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustAccountClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		if cmd.Flags().Changed("json") {
			err = downloadJson.Unmarshal(&downloadReq)
			if err != nil {
				return err
			}
		} else {
			downloadReq.StartMonth = args[0]
			downloadReq.EndMonth = args[1]
		}

		err = a.BillableUsage.Download(ctx, downloadReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service BillableUsage
