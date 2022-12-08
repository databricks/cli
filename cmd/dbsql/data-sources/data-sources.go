package data_sources

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
