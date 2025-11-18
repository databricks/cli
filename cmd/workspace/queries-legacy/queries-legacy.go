// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package queries_legacy

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queries-legacy",
		Short: `These endpoints are used for CRUD operations on query definitions.`,
		Long: `These endpoints are used for CRUD operations on query definitions. Query
  definitions include the target SQL warehouse, query text, name, description,
  tags, parameters, and visualizations. Queries can be scheduled using the
  sql_task type of the Jobs API, e.g. :method:jobs/create.
  
  **Note**: A new version of the Databricks SQL API is now available. Please see
  the latest version. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newRestore())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*sql.QueryPostContent,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sql.QueryPostContent
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.DataSourceId, "data-source-id", createReq.DataSourceId, `Data source ID maps to the ID of the data source used by the resource and is distinct from the warehouse ID.`)
	cmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `General description that conveys additional information about this query such as usage notes.`)
	cmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `The title of this query that appears in list views, widget headings, and on the query page.`)
	// TODO: any: options
	cmd.Flags().StringVar(&createReq.Parent, "parent", createReq.Parent, `The identifier of the workspace folder containing the object.`)
	cmd.Flags().StringVar(&createReq.Query, "query", createReq.Query, `The text of the query to be run.`)
	cmd.Flags().Var(&createReq.RunAsRole, "run-as-role", `Sets the **Run as** role for the object. Supported values: [owner, viewer]`)
	// TODO: array: tags

	cmd.Use = "create"
	cmd.Short = `Create a new query definition.`
	cmd.Long = `Create a new query definition.
  
  Creates a new query definition. Queries created with this endpoint belong to
  the authenticated user making the request.
  
  The data_source_id field specifies the ID of the SQL warehouse to run this
  query against. You can use the Data Sources API to see a complete list of
  available SQL warehouses. Or you can copy the data_source_id from an
  existing query.
  
  **Note**: You cannot add a visualization until you create the query.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queries/create instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}

		response, err := w.QueriesLegacy.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sql.DeleteQueriesLegacyRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sql.DeleteQueriesLegacyRequest

	cmd.Use = "delete QUERY_ID"
	cmd.Short = `Delete a query.`
	cmd.Long = `Delete a query.
  
  Moves a query to the trash. Trashed queries immediately disappear from
  searches and list views, and they cannot be used for alerts. The trash is
  deleted after 30 days.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queries/delete instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.QueryId = args[0]

		err = w.QueriesLegacy.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sql.GetQueriesLegacyRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sql.GetQueriesLegacyRequest

	cmd.Use = "get QUERY_ID"
	cmd.Short = `Get a query definition.`
	cmd.Long = `Get a query definition.
  
  Retrieve a query object definition along with contextual permissions
  information about the currently authenticated user.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queries/get instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.QueryId = args[0]

		response, err := w.QueriesLegacy.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*sql.ListQueriesLegacyRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sql.ListQueriesLegacyRequest

	cmd.Flags().StringVar(&listReq.Order, "order", listReq.Order, `Name of query attribute to order by.`)
	cmd.Flags().IntVar(&listReq.Page, "page", listReq.Page, `Page number to retrieve.`)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Number of queries to return per page.`)
	cmd.Flags().StringVar(&listReq.Q, "q", listReq.Q, `Full text search term.`)

	cmd.Use = "list"
	cmd.Short = `Get a list of queries.`
	cmd.Long = `Get a list of queries.
  
  Gets a list of queries. Optionally, this list can be filtered by a search
  term.
  
  **Warning**: Calling this API concurrently 10 or more times could result in
  throttling, service degradation, or a temporary ban.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queries/list instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.QueriesLegacy.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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

// start restore command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreOverrides []func(
	*cobra.Command,
	*sql.RestoreQueriesLegacyRequest,
)

func newRestore() *cobra.Command {
	cmd := &cobra.Command{}

	var restoreReq sql.RestoreQueriesLegacyRequest

	cmd.Use = "restore QUERY_ID"
	cmd.Short = `Restore a query.`
	cmd.Long = `Restore a query.
  
  Restore a query that has been moved to the trash. A restored query appears in
  list views and searches. You can use restored queries for alerts.
  
  **Note**: A new version of the Databricks SQL API is now available. Please see
  the latest version. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		restoreReq.QueryId = args[0]

		err = w.QueriesLegacy.Restore(ctx, restoreReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range restoreOverrides {
		fn(cmd, &restoreReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*sql.QueryEditContent,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq sql.QueryEditContent
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.DataSourceId, "data-source-id", updateReq.DataSourceId, `Data source ID maps to the ID of the data source used by the resource and is distinct from the warehouse ID.`)
	cmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `General description that conveys additional information about this query such as usage notes.`)
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The title of this query that appears in list views, widget headings, and on the query page.`)
	// TODO: any: options
	cmd.Flags().StringVar(&updateReq.Query, "query", updateReq.Query, `The text of the query to be run.`)
	cmd.Flags().Var(&updateReq.RunAsRole, "run-as-role", `Sets the **Run as** role for the object. Supported values: [owner, viewer]`)
	// TODO: array: tags

	cmd.Use = "update QUERY_ID"
	cmd.Short = `Change a query definition.`
	cmd.Long = `Change a query definition.
  
  Modify this query definition.
  
  **Note**: You cannot undo this operation.
  
  **Note**: A new version of the Databricks SQL API is now available. Please use
  :method:queries/update instead. [Learn more]
  
  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateReq.QueryId = args[0]

		response, err := w.QueriesLegacy.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service QueriesLegacy
