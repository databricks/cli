package billable_usage

import (
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/billing"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "billable-usage",
	Short: `This API allows you to download billable usage logs for the specified account and date range.`, // TODO: fix FirstSentence logic and append dot to summary
}

var downloadReq billing.DownloadRequest

func init() {
	Cmd.AddCommand(downloadCmd)
	// TODO: short flags

	downloadCmd.Flags().StringVar(&downloadReq.EndMonth, "end-month", "", `Format: YYYY-MM.`)
	downloadCmd.Flags().BoolVar(&downloadReq.PersonalData, "personal-data", false, `Specify whether to include personally identifiable information in the billable usage logs, for example the email addresses of cluster creators.`)
	downloadCmd.Flags().StringVar(&downloadReq.StartMonth, "start-month", "", `Format: YYYY-MM.`)

}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: `Return billable usage logs Returns billable usage logs in CSV format for the specified account and date range.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		a := project.Get(ctx).AccountClient()
		err := a.BillableUsage.Download(ctx, downloadReq)
		if err != nil {
			return err
		}

		return nil
	},
}
