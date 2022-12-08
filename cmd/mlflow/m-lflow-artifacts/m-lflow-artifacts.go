package m_lflow_artifacts

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
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

	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", "", `Token indicating the page of artifact results to fetch.`)
	listCmd.Flags().StringVar(&listReq.Path, "path", "", `Filter artifacts matching this path (a relative path from the root artifact directory).`)
	listCmd.Flags().StringVar(&listReq.RunId, "run-id", "", `ID of the run whose artifacts to list.`)
	listCmd.Flags().StringVar(&listReq.RunUuid, "run-uuid", "", `[Deprecated, use run_id instead] ID of the run whose artifacts to list.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get all artifacts.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.MLflowArtifacts.ListAll(ctx, listReq)
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
