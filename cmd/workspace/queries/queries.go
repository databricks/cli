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

var Cmd = &cobra.Command{
	Use:   "queries",
	Short: `These endpoints are used for CRUD operations on query definitions.`,
	Long: `These endpoints are used for CRUD operations on query definitions. Query
  definitions include the target SQL warehouse, query text, name, description,
  tags, parameters, and visualizations. Queries can be scheduled using the
  sql_task type of the Jobs API, e.g. :method:jobs/create.`,
	Annotations: map[string]string{
		"package": "sql",
	},
}

// start create command

var createReq sql.QueryPostContent
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.DataSourceId, "data-source-id", createReq.DataSourceId, `The ID of the data source / SQL warehouse where this query will run.`)
	createCmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `General description that can convey additional information about this query such as usage notes.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `The name or title of this query to display in list views.`)
	// TODO: any: options
	createCmd.Flags().StringVar(&createReq.Parent, "parent", createReq.Parent, `The identifier of the workspace folder containing the query.`)
	createCmd.Flags().StringVar(&createReq.Query, "query", createReq.Query, `The text of the query.`)

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
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Queries.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq sql.DeleteQueryRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete QUERY_ID",
	Short: `Delete a query.`,
	Long: `Delete a query.
  
  Moves a query to the trash. Trashed queries immediately disappear from
  searches and list views, and they cannot be used for alerts. The trash is
  deleted after 30 days.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
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
		}

		err = w.Queries.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq sql.GetQueryRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get QUERY_ID",
	Short: `Get a query definition.`,
	Long: `Get a query definition.
  
  Retrieve a query object definition along with contextual permissions
  information about the currently authenticated user.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
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
		}

		response, err := w.Queries.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq sql.ListQueriesRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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

		response, err := w.Queries.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start restore command

var restoreReq sql.RestoreQueryRequest
var restoreJson flags.JsonFlag

func init() {
	Cmd.AddCommand(restoreCmd)
	// TODO: short flags
	restoreCmd.Flags().Var(&restoreJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var restoreCmd = &cobra.Command{
	Use:   "restore QUERY_ID",
	Short: `Restore a query.`,
	Long: `Restore a query.
  
  Restore a query that has been moved to the trash. A restored query appears in
  list views and searches. You can use restored queries for alerts.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = restoreJson.Unmarshal(&restoreReq)
			if err != nil {
				return err
			}
		} else {
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
		}

		err = w.Queries.Restore(ctx, restoreReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq sql.QueryEditContent
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.DataSourceId, "data-source-id", updateReq.DataSourceId, `The ID of the data source / SQL warehouse where this query will run.`)
	updateCmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `General description that can convey additional information about this query such as usage notes.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `The name or title of this query to display in list views.`)
	// TODO: any: options
	updateCmd.Flags().StringVar(&updateReq.Query, "query", updateReq.Query, `The text of the query.`)

}

var updateCmd = &cobra.Command{
	Use:   "update QUERY_ID",
	Short: `Change a query definition.`,
	Long: `Change a query definition.
  
  Modify this query definition.
  
  **Note**: You cannot undo this operation.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
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
		}

		response, err := w.Queries.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service Queries
