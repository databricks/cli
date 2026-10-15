// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package query_history

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	queryhistory "github.com/databricks/cli/libs/sdktypes/queryhistory/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*queryhistory.ListQueriesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq queryhistory.ListQueriesRequest

	// TODO: complex arg: filter_by
	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeMetrics, "include-metrics", `Whether to include the query metrics with each query.`)
	flags.IntPtrVar(cmd.Flags(), &listReq.MaxResults, "max-results", `Limit the number of results returned in one page.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `A token that can be used to get the next page of results.`)

	cmd.Use = "list"
	cmd.Short = `List Queries.`
	cmd.Long = `List Queries.

  List the history of queries through SQL warehouses, and serverless compute.

  You can filter by user ID, warehouse ID, status, and time range. Most recently
  started queries are returned first (up to max_results in request). The
  pagination token returned in response can be used to list subsequent query
  statuses.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := queryhistory.ListQueriesRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		response, err := w.QueryHistory.ListQueries(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := queryhistory.ListQueriesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
