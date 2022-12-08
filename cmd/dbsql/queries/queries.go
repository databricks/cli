package queries

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/dbsql"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "queries",
	Short: `These endpoints are used for CRUD operations on query definitions.`,
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
