// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package query_history

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-history",
		Short: `A service responsible for storing and retrieving the list of queries run against SQL endpoints and serverless compute.`,
		Long: `A service responsible for storing and retrieving the list of queries run
  against SQL endpoints and serverless compute.`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*sql.ListQueryHistoryRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sql.ListQueryHistoryRequest

	// TODO: complex arg: filter_by
	cmd.Flags().BoolVar(&listReq.IncludeMetrics, "include-metrics", listReq.IncludeMetrics, `Whether to include the query metrics with each query.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Limit the number of results returned in one page.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `A token that can be used to get the next page of results.`)

	cmd.Use = "list"
	cmd.Short = `List Queries.`
	cmd.Long = `List Queries.
  
  List the history of queries through SQL warehouses, and serverless compute.
  
  You can filter by user ID, warehouse ID, status, and time range. Most recently
  started queries are returned first (up to max_results in request). The
  pagination token returned in response can be used to list subsequent query
  statuses.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.QueryHistory.List(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// end service QueryHistory
