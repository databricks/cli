package m_lflow_artifacts

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "m-lflow-artifacts",
}

var listReq mlflow.ListArtifactsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Token indicating the page of artifact results to fetch.`)
	listCmd.Flags().StringVar(&listReq.Path, "path", listReq.Path, `Filter artifacts matching this path (a relative path from the root artifact directory).`)
	listCmd.Flags().StringVar(&listReq.RunId, "run-id", listReq.RunId, `ID of the run whose artifacts to list.`)
	listCmd.Flags().StringVar(&listReq.RunUuid, "run-uuid", listReq.RunUuid, `[Deprecated, use run_id instead] ID of the run whose artifacts to list.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all artifacts.`,
	Long: `Get all artifacts.
  
  List artifacts for a run. Takes an optional artifact_path prefix. If it is
  specified, the response contains only artifacts with the specified prefix.",`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowArtifacts.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service MLflowArtifacts
