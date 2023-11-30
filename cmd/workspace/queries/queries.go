// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package queries

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
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
		Use:   "queries",
		Short: `These endpoints are used for CRUD operations on query definitions.`,
		Long: `These endpoints are used for CRUD operations on query definitions. Query
  definitions include the target SQL warehouse, query text, name, description,
  tags, parameters, and visualizations. Queries can be scheduled using the
  sql_task type of the Jobs API, e.g. :method:jobs/create.`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},
	}

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

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a new query definition.`
	cmd.Long = `Create a new query definition.
  
  Creates a new query definition. Queries created with this endpoint belong to
  the authenticated user making the request.
  
  The data_source_id field specifies the ID of the SQL warehouse to run this
  query against. You can use the Data Sources API to see a complete list of
  available SQL warehouses. Or you can copy the data_source_id from an
  existing query.
  
  **Note**: You cannot add a visualization until you create the query.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Queries.Create(ctx, createReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sql.DeleteQueryRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sql.DeleteQueryRequest

	// TODO: short flags

	cmd.Use = "delete QUERY_ID"
	cmd.Short = `Delete a query.`
	cmd.Long = `Delete a query.
  
  Moves a query to the trash. Trashed queries immediately disappear from
  searches and list views, and they cannot be used for alerts. The trash is
  deleted after 30 days.

  Arguments:
    QUERY_ID: 
    `

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No QUERY_ID argument specified. Loading names for Queries drop-down."
			names, err := w.Queries.QueryNameToIdMap(ctx, sql.ListQueriesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Queries drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		deleteReq.QueryId = args[0]

		err = w.Queries.Delete(ctx, deleteReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDelete())
	})
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sql.GetQueryRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sql.GetQueryRequest

	// TODO: short flags

	cmd.Use = "get QUERY_ID"
	cmd.Short = `Get a query definition.`
	cmd.Long = `Get a query definition.
  
  Retrieve a query object definition along with contextual permissions
  information about the currently authenticated user.

  Arguments:
    QUERY_ID: 
    `

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No QUERY_ID argument specified. Loading names for Queries drop-down."
			names, err := w.Queries.QueryNameToIdMap(ctx, sql.ListQueriesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Queries drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		getReq.QueryId = args[0]

		response, err := w.Queries.Get(ctx, getReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGet())
	})
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*sql.ListQueriesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sql.ListQueriesRequest

	// TODO: short flags

	cmd.Flags().StringVar(&listReq.Order, "order", listReq.Order, `Name of query attribute to order by.`)
	cmd.Flags().IntVar(&listReq.Page, "page", listReq.Page, `Page number to retrieve.`)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Number of queries to return per page.`)
	cmd.Flags().StringVar(&listReq.Q, "q", listReq.Q, `Full text search term.`)

	cmd.Use = "list"
	cmd.Short = `Get a list of queries.`
	cmd.Long = `Get a list of queries.
  
  Gets a list of queries. Optionally, this list can be filtered by a search
  term.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.Queries.ListAll(ctx, listReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newList())
	})
}

// start restore command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreOverrides []func(
	*cobra.Command,
	*sql.RestoreQueryRequest,
)

func newRestore() *cobra.Command {
	cmd := &cobra.Command{}

	var restoreReq sql.RestoreQueryRequest

	// TODO: short flags

	cmd.Use = "restore QUERY_ID"
	cmd.Short = `Restore a query.`
	cmd.Long = `Restore a query.
  
  Restore a query that has been moved to the trash. A restored query appears in
  list views and searches. You can use restored queries for alerts.

  Arguments:
    QUERY_ID: 
    `

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No QUERY_ID argument specified. Loading names for Queries drop-down."
			names, err := w.Queries.QueryNameToIdMap(ctx, sql.ListQueriesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Queries drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		restoreReq.QueryId = args[0]

		err = w.Queries.Restore(ctx, restoreReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRestore())
	})
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

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.DataSourceId, "data-source-id", updateReq.DataSourceId, `Data source ID.`)
	cmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `General description that conveys additional information about this query such as usage notes.`)
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The title of this query that appears in list views, widget headings, and on the query page.`)
	// TODO: any: options
	cmd.Flags().StringVar(&updateReq.Query, "query", updateReq.Query, `The text of the query to be run.`)

	cmd.Use = "update QUERY_ID"
	cmd.Short = `Change a query definition.`
	cmd.Long = `Change a query definition.
  
  Modify this query definition.
  
  **Note**: You cannot undo this operation.

  Arguments:
    QUERY_ID: 
    `

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No QUERY_ID argument specified. Loading names for Queries drop-down."
			names, err := w.Queries.QueryNameToIdMap(ctx, sql.ListQueriesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Queries drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		updateReq.QueryId = args[0]

		response, err := w.Queries.Update(ctx, updateReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdate())
	})
}

// end service Queries
