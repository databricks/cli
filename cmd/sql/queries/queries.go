package queries

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "queries",
	Short: `These endpoints are used for CRUD operations on query definitions.`,
	Long: `These endpoints are used for CRUD operations on query definitions. Query
  definitions include the target SQL warehouse, query text, name, description,
  tags, execution schedule, parameters, and visualizations.`,
}

var createReq sql.QueryPostContent

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.DataSourceId, "data-source-id", createReq.DataSourceId, `The ID of the data source / SQL warehouse where this query will run.`)
	createCmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `General description that can convey additional information about this query such as usage notes.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `The name or title of this query to display in list views.`)
	// TODO: any: options
	createCmd.Flags().StringVar(&createReq.Query, "query", createReq.Query, `The text of the query.`)
	createCmd.Flags().StringVar(&createReq.QueryId, "query-id", createReq.QueryId, ``)
	// TODO: complex arg: schedule

}

var createCmd = &cobra.Command{
	Use:   "create",
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
		response, err := w.Queries.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteReq sql.DeleteQueryRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.QueryId, "query-id", deleteReq.QueryId, ``)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a query.`,
	Long: `Delete a query.
  
  Moves a query to the trash. Trashed queries immediately disappear from
  searches and list views, and they cannot be used for alerts. The trash is
  deleted after 30 days.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Queries.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getReq sql.GetQueryRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.QueryId, "query-id", getReq.QueryId, ``)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a query definition.`,
	Long: `Get a query definition.
  
  Retrieve a query object definition along with contextual permissions
  information about the currently authenticated user.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Queries.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var listReq sql.ListQueriesRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.Order, "order", listReq.Order, `Name of query attribute to order by.`)
	listCmd.Flags().IntVar(&listReq.Page, "page", listReq.Page, `Page number to retrieve.`)
	listCmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Number of queries to return per page.`)
	listCmd.Flags().StringVar(&listReq.Q, "q", listReq.Q, `Full text search term.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Get a list of queries.`,
	Long: `Get a list of queries.
  
  Gets a list of queries. Optionally, this list can be filtered by a search
  term.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Queries.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var restoreReq sql.RestoreQueryRequest

func init() {
	Cmd.AddCommand(restoreCmd)
	// TODO: short flags

	restoreCmd.Flags().StringVar(&restoreReq.QueryId, "query-id", restoreReq.QueryId, ``)

}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: `Restore a query.`,
	Long: `Restore a query.
  
  Restore a query that has been moved to the trash. A restored query appears in
  list views and searches. You can use restored queries for alerts.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Queries.Restore(ctx, restoreReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var updateReq sql.QueryPostContent

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.DataSourceId, "data-source-id", updateReq.DataSourceId, `The ID of the data source / SQL warehouse where this query will run.`)
	updateCmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `General description that can convey additional information about this query such as usage notes.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name or title of this query to display in list views.`)
	// TODO: any: options
	updateCmd.Flags().StringVar(&updateReq.Query, "query", updateReq.Query, `The text of the query.`)
	updateCmd.Flags().StringVar(&updateReq.QueryId, "query-id", updateReq.QueryId, ``)
	// TODO: complex arg: schedule

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Change a query definition.`,
	Long: `Change a query definition.
  
  Modify this query definition.
  
  **Note**: You cannot undo this operation.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Queries.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service Queries
