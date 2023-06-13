// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package query_history

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "query-history",
	Short: `Access the history of queries through SQL warehouses.`,
	Long:  `Access the history of queries through SQL warehouses.`,
	Annotations: map[string]string{
		"package": "sql",
	},
}

// start list command

var listReq sql.ListQueryHistoryRequest
var listJson flags.JsonFlag

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
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.QueryHistory.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service QueryHistory
