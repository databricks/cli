package query_history

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/warehouses"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "query-history",
	Short: `Access the history of queries through SQL warehouses.`, // TODO: fix FirstSentence logic and append dot to summary
}

var listQueriesReq warehouses.ListQueriesRequest

func init() {
	Cmd.AddCommand(listQueriesCmd)
	// TODO: short flags

	// TODO: complex arg: filter_by
	listQueriesCmd.Flags().BoolVar(&listQueriesReq.IncludeMetrics, "include-metrics", false, `Whether to include metrics about query.`)
	listQueriesCmd.Flags().IntVar(&listQueriesReq.MaxResults, "max-results", 0, `Limit the number of results returned in one page.`)
	listQueriesCmd.Flags().StringVar(&listQueriesReq.PageToken, "page-token", "", `A token that can be used to get the next page of results.`)

}

var listQueriesCmd = &cobra.Command{
	Use:   "list-queries",
	Short: `List List the history of queries through SQL warehouses.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.QueryHistory.ListQueriesAll(ctx, listQueriesReq)
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
