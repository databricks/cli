package queries

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "queries",
	Short: `These endpoints are used for CRUD operations on query definitions.`,
	Long: `These endpoints are used for CRUD operations on query definitions. Query
  definitions include the target SQL warehouse, query text, name, description,
  tags, execution schedule, parameters, and visualizations.`,
}

var createQueryReq dbsql.QueryPostContent

func init() {
	Cmd.AddCommand(createQueryCmd)
	// TODO: short flags

	createQueryCmd.Flags().StringVar(&createQueryReq.DataSourceId, "data-source-id", "", `The ID of the data source / SQL warehouse where this query will run.`)
	createQueryCmd.Flags().StringVar(&createQueryReq.Description, "description", "", `General description that can convey additional information about this query such as usage notes.`)
	createQueryCmd.Flags().StringVar(&createQueryReq.Name, "name", "", `The name or title of this query to display in list views.`)
	// TODO: any: options
	createQueryCmd.Flags().StringVar(&createQueryReq.Query, "query", "", `The text of the query.`)
	createQueryCmd.Flags().StringVar(&createQueryReq.QueryId, "query-id", "", ``)
	// TODO: complex arg: schedule

}

var createQueryCmd = &cobra.Command{
	Use:   "create-query",
	Short: `Create a new query definition.`,
	Long: `Create a new query definition.
  
  Creates a new query definition. Queries created with this endpoint belong to
  the authenticated user making the request.
  
  The data_source_id field specifies the ID of the SQL warehouse to run this
  query against. You can use the Data Sources API to see a complete list of
  available SQL warehouses. Or you can copy the data_source_id from an
  existing query.
  
  **Note**: You cannot add a visualization until you create the query.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Queries.CreateQuery(ctx, createQueryReq)
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

var deleteQueryReq dbsql.DeleteQueryRequest

func init() {
	Cmd.AddCommand(deleteQueryCmd)
	// TODO: short flags

	deleteQueryCmd.Flags().StringVar(&deleteQueryReq.QueryId, "query-id", "", ``)

}

var deleteQueryCmd = &cobra.Command{
	Use:   "delete-query",
	Short: `Delete a query.`,
	Long: `Delete a query.
  
  Moves a query to the trash. Trashed queries immediately disappear from
  searches and list views, and they cannot be used for alerts. The trash is
  deleted after 30 days.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Queries.DeleteQuery(ctx, deleteQueryReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getQueryReq dbsql.GetQueryRequest

func init() {
	Cmd.AddCommand(getQueryCmd)
	// TODO: short flags

	getQueryCmd.Flags().StringVar(&getQueryReq.QueryId, "query-id", "", ``)

}

var getQueryCmd = &cobra.Command{
	Use:   "get-query",
	Short: `Get a query definition.`,
	Long: `Get a query definition.
  
  Retrieve a query object definition along with contextual permissions
  information about the currently authenticated user.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Queries.GetQuery(ctx, getQueryReq)
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

var listQueriesReq dbsql.ListQueriesRequest

func init() {
	Cmd.AddCommand(listQueriesCmd)
	// TODO: short flags

	listQueriesCmd.Flags().StringVar(&listQueriesReq.Order, "order", "", `Name of query attribute to order by.`)
	listQueriesCmd.Flags().IntVar(&listQueriesReq.Page, "page", 0, `Page number to retrieve.`)
	listQueriesCmd.Flags().IntVar(&listQueriesReq.PageSize, "page-size", 0, `Number of queries to return per page.`)
	listQueriesCmd.Flags().StringVar(&listQueriesReq.Q, "q", "", `Full text search term.`)

}

var listQueriesCmd = &cobra.Command{
	Use:   "list-queries",
	Short: `Get a list of queries.`,
	Long: `Get a list of queries.
  
  Gets a list of queries. Optionally, this list can be filtered by a search
  term.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Queries.ListQueriesAll(ctx, listQueriesReq)
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

var restoreQueryReq dbsql.RestoreQueryRequest

func init() {
	Cmd.AddCommand(restoreQueryCmd)
	// TODO: short flags

	restoreQueryCmd.Flags().StringVar(&restoreQueryReq.QueryId, "query-id", "", ``)

}

var restoreQueryCmd = &cobra.Command{
	Use:   "restore-query",
	Short: `Restore a query.`,
	Long: `Restore a query.
  
  Restore a query that has been moved to the trash. A restored query appears in
  list views and searches. You can use restored queries for alerts.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Queries.RestoreQuery(ctx, restoreQueryReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateQueryReq dbsql.QueryPostContent

func init() {
	Cmd.AddCommand(updateQueryCmd)
	// TODO: short flags

	updateQueryCmd.Flags().StringVar(&updateQueryReq.DataSourceId, "data-source-id", "", `The ID of the data source / SQL warehouse where this query will run.`)
	updateQueryCmd.Flags().StringVar(&updateQueryReq.Description, "description", "", `General description that can convey additional information about this query such as usage notes.`)
	updateQueryCmd.Flags().StringVar(&updateQueryReq.Name, "name", "", `The name or title of this query to display in list views.`)
	// TODO: any: options
	updateQueryCmd.Flags().StringVar(&updateQueryReq.Query, "query", "", `The text of the query.`)
	updateQueryCmd.Flags().StringVar(&updateQueryReq.QueryId, "query-id", "", ``)
	// TODO: complex arg: schedule

}

var updateQueryCmd = &cobra.Command{
	Use:   "update-query",
	Short: `Change a query definition.`,
	Long: `Change a query definition.
  
  Modify this query definition.
  
  **Note**: You cannot undo this operation.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Queries.UpdateQuery(ctx, updateQueryReq)
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

// end service Queries
