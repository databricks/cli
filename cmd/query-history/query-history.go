// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package query_history

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "query-history",
	Short: `Access the history of queries through SQL warehouses.`,
	Long:  `Access the history of queries through SQL warehouses.`,
}

// start list command

var listReq sql.ListQueryHistoryRequest
var listJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: filter_by
	listCmd.Flags().BoolVar(&listReq.IncludeMetrics, "include-metrics", listReq.IncludeMetrics, `Whether to include metrics about query.`)
	listCmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Limit the number of results returned in one page.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A token that can be used to get the next page of results.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List Queries.`,
	Long: `List Queries.
  
  List the history of queries through SQL warehouses.
  
  You can filter by user ID, warehouse ID, status, and time range.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = listJson.Unmarshall(&listReq)
		if err != nil {
			return err
		}

		response, err := w.QueryHistory.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service QueryHistory
