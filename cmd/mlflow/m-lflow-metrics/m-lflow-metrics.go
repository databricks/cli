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

var getHistoryReq mlflow.GetHistoryRequest

func init() {
	Cmd.AddCommand(getHistoryCmd)
	// TODO: short flags

	getHistoryCmd.Flags().StringVar(&getHistoryReq.MetricKey, "metric-key", "", `Name of the metric.`)
	getHistoryCmd.Flags().StringVar(&getHistoryReq.RunId, "run-id", "", `ID of the run from which to fetch metric values.`)
	getHistoryCmd.Flags().StringVar(&getHistoryReq.RunUuid, "run-uuid", "", `[Deprecated, use run_id instead] ID of the run from which to fetch metric values.`)

}

var getHistoryCmd = &cobra.Command{
	Use:   "get-history",
	Short: `Get all history.`,
	Long: `Get all history.
  
  Gets a list of all values for the specified metric for a given run.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowMetrics.GetHistory(ctx, getHistoryReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}
