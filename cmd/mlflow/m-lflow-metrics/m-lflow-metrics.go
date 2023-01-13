// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package m_lflow_metrics

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "m-lflow-metrics",
}

// start get-history command

var getHistoryReq mlflow.GetHistoryRequest

func init() {
	Cmd.AddCommand(getHistoryCmd)
	// TODO: short flags

	// getHistoryCmd.Flags().IntVar(&getHistoryReq.MaxResults, "max-results", getHistoryReq.MaxResults, `Maximum number of Metric records to return per paginated request.`)
	// getHistoryCmd.Flags().StringVar(&getHistoryReq.PageToken, "page-token", getHistoryReq.PageToken, `Token indicating the page of metric histories to fetch.`)
	getHistoryCmd.Flags().StringVar(&getHistoryReq.RunId, "run-id", getHistoryReq.RunId, `ID of the run from which to fetch metric values.`)
	getHistoryCmd.Flags().StringVar(&getHistoryReq.RunUuid, "run-uuid", getHistoryReq.RunUuid, `[Deprecated, use run_id instead] ID of the run from which to fetch metric values.`)

}

var getHistoryCmd = &cobra.Command{
	Use:   "get-history METRIC_KEY",
	Short: `Get history of a given metric within a run.`,
	Long: `Get history of a given metric within a run.
  
  Gets a list of all values for the specified metric for a given run.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getHistoryReq.MetricKey = args[0]

		response, err := w.MLflowMetrics.GetHistory(ctx, getHistoryReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service MLflowMetrics
