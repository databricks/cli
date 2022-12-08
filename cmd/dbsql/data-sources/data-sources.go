package data_sources

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "data-sources",
	Short: `This API is provided to assist you in making new query objects.`,
}

func init() {
	Cmd.AddCommand(listDataSourcesCmd)

}

var listDataSourcesCmd = &cobra.Command{
	Use:   "list-data-sources",
	Short: `Get a list of SQL warehouses.`,
	Long: `Get a list of SQL warehouses.
  
  Retrieves a full list of SQL warehouses available in this workspace. All
  fields that appear in this API response are enumerated for clarity. However,
  you need only a SQL warehouse's id to create new queries against it.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.DataSources.ListDataSources(ctx)
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
