// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package queries_legacy

import (
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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

  **Warning**: This API is deprecated. Please see the latest version of the
  Databricks SQL API. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`,
		GroupID: "sql",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*map[string]any,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq map[string]any
	createReq = map[string]any{}
	var createDataSourceIdRaw string
	var createDescriptionRaw string
	var createNameRaw string
	var createParentRaw string
	var createQueryRaw string
	var createRunAsRoleRaw string

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDataSourceIdRaw, "data-source-id", createDataSourceIdRaw, `Data source ID maps to the ID of the data source used by the resource and is distinct from the warehouse ID.`)
	cmd.Flags().StringVar(&createDescriptionRaw, "description", createDescriptionRaw, `General description that conveys additional information about this query such as usage notes.`)
	cmd.Flags().StringVar(&createNameRaw, "name", createNameRaw, `The title of this query that appears in list views, widget headings, and on the query page.`)
	// TODO: any: options
	cmd.Flags().StringVar(&createParentRaw, "parent", createParentRaw, `The identifier of the workspace folder containing the object.`)
	cmd.Flags().StringVar(&createQueryRaw, "query", createQueryRaw, `The text of the query to be run.`)
	flags.EnumVar(cmd.Flags(), &createRunAsRoleRaw, "run-as-role", `Sets the **Run as** role for the object. Supported values: [owner, viewer]`, "RunAsRole", "owner", "viewer")
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

  **Warning**: This API is deprecated. Please use :method:queries/create
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

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

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("data-source-id") || createDataSourceIdRaw != "" {
			createReq["data_source_id"] = createDataSourceIdRaw
		}
		if cmd.Flags().Changed("description") || createDescriptionRaw != "" {
			createReq["description"] = createDescriptionRaw
		}
		if cmd.Flags().Changed("name") || createNameRaw != "" {
			createReq["name"] = createNameRaw
		}
		if cmd.Flags().Changed("parent") || createParentRaw != "" {
			createReq["parent"] = createParentRaw
		}
		if cmd.Flags().Changed("query") || createQueryRaw != "" {
			createReq["query"] = createQueryRaw
		}
		if cmd.Flags().Changed("run-as-role") || createRunAsRoleRaw != "" {
			createReq["run_as_role"] = createRunAsRoleRaw
		}
		rawPath := "/api/2.0/preview/sql/queries"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createReq, &response)
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
	*map[string]any,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq map[string]any
	deleteReq = map[string]any{}
	var deleteQueryIdRaw string

	cmd.Use = "delete QUERY_ID"
	cmd.Short = `Delete a query.`
	cmd.Long = `Delete a query.

  Moves a query to the trash. Trashed queries immediately disappear from
  searches and list views, and they cannot be used for alerts. The trash is
  deleted after 30 days.

  **Warning**: This API is deprecated. Please use :method:queries/delete
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteQueryIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/queries/%s", fmt.Sprint(deleteQueryIdRaw))

		maps.Copy(rawQuery, deleteReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
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
	*map[string]any,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq map[string]any
	getReq = map[string]any{}
	var getQueryIdRaw string

	cmd.Use = "get QUERY_ID"
	cmd.Short = `Get a query definition.`
	cmd.Long = `Get a query definition.

  Retrieve a query object definition along with contextual permissions
  information about the currently authenticated user.

  **Warning**: This API is deprecated. Please use :method:queries/get instead.
  [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getQueryIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/queries/%s", fmt.Sprint(getQueryIdRaw))

		maps.Copy(rawQuery, getReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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
	*map[string]any,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq map[string]any
	listReq = map[string]any{}
	var listOrderRaw string
	var listPageRaw int
	var listPageSizeRaw int
	var listQRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	cmd.Flags().StringVar(&listOrderRaw, "order", listOrderRaw, `Name of query attribute to order by.`)
	cmd.Flags().StringVar(&listQRaw, "q", listQRaw, `Full text search term.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().IntVar(&listPageRaw, "page", listPageRaw, `Page number to retrieve.`)
	cmd.Flags().Lookup("page").Hidden = true
	cmd.Flags().IntVar(&listPageSizeRaw, "page-size", listPageSizeRaw, `Number of results per API page.`)
	cmd.Flags().Lookup("page-size").Hidden = true

	cmd.Use = "list"
	cmd.Short = `Get a list of queries.`
	cmd.Long = `Get a list of queries.

  Gets a list of queries. Optionally, this list can be filtered by a search
  term.

  **Warning**: Calling this API concurrently 10 or more times could result in
  throttling, service degradation, or a temporary ban.

  **Warning**: This API is deprecated. Please use :method:queries/list instead.
  [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

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

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("order") || listOrderRaw != "" {
			rawQuery["order"] = listOrderRaw
		}
		if cmd.Flags().Changed("page") || listPageRaw != 0 {
			rawQuery["page"] = listPageRaw
		}
		if cmd.Flags().Changed("page-size") || listPageSizeRaw != 0 {
			rawQuery["page_size"] = listPageSizeRaw
		}
		if cmd.Flags().Changed("q") || listQRaw != "" {
			rawQuery["q"] = listQRaw
		}
		rawPath := "/api/2.0/preview/sql/queries"

		maps.Copy(rawQuery, listReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
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

// start restore command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreOverrides []func(
	*cobra.Command,
	*map[string]any,
)

func newRestore() *cobra.Command {
	cmd := &cobra.Command{}
	var restoreReq map[string]any
	restoreReq = map[string]any{}
	var restoreQueryIdRaw string

	cmd.Use = "restore QUERY_ID"
	cmd.Short = `Restore a query.`
	cmd.Long = `Restore a query.

  Restore a query that has been moved to the trash. A restored query appears in
  list views and searches. You can use restored queries for alerts.

  **Warning**: This API is deprecated. Please see the latest version. [Learn
  more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		restoreQueryIdRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/queries/trash/%s", fmt.Sprint(restoreQueryIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, restoreReq, nil)
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
	*map[string]any,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq map[string]any
	updateReq = map[string]any{}
	var updateDataSourceIdRaw string
	var updateDescriptionRaw string
	var updateNameRaw string
	var updateQueryRaw string
	var updateQueryIdRaw string
	var updateRunAsRoleRaw string

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateDataSourceIdRaw, "data-source-id", updateDataSourceIdRaw, `Data source ID maps to the ID of the data source used by the resource and is distinct from the warehouse ID.`)
	cmd.Flags().StringVar(&updateDescriptionRaw, "description", updateDescriptionRaw, `General description that conveys additional information about this query such as usage notes.`)
	cmd.Flags().StringVar(&updateNameRaw, "name", updateNameRaw, `The title of this query that appears in list views, widget headings, and on the query page.`)
	// TODO: any: options
	cmd.Flags().StringVar(&updateQueryRaw, "query", updateQueryRaw, `The text of the query to be run.`)
	flags.EnumVar(cmd.Flags(), &updateRunAsRoleRaw, "run-as-role", `Sets the **Run as** role for the object. Supported values: [owner, viewer]`, "RunAsRole", "owner", "viewer")
	// TODO: array: tags

	cmd.Use = "update QUERY_ID"
	cmd.Short = `Change a query definition.`
	cmd.Long = `Change a query definition.

  Modify this query definition.

  **Note**: You cannot undo this operation.

  **Warning**: This API is deprecated. Please use :method:queries/update
  instead. [Learn more]

  [Learn more]: https://docs.databricks.com/en/sql/dbsql-api-latest.html`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateQueryIdRaw = args[0]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("data-source-id") || updateDataSourceIdRaw != "" {
			updateReq["data_source_id"] = updateDataSourceIdRaw
		}
		if cmd.Flags().Changed("description") || updateDescriptionRaw != "" {
			updateReq["description"] = updateDescriptionRaw
		}
		if cmd.Flags().Changed("name") || updateNameRaw != "" {
			updateReq["name"] = updateNameRaw
		}
		if cmd.Flags().Changed("query") || updateQueryRaw != "" {
			updateReq["query"] = updateQueryRaw
		}
		if cmd.Flags().Changed("run-as-role") || updateRunAsRoleRaw != "" {
			updateReq["run_as_role"] = updateRunAsRoleRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/preview/sql/queries/%s", fmt.Sprint(updateQueryIdRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateReq, &response)
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
