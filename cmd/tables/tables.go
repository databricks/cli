// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package tables

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "tables",
	Short: `A table resides in the third layer of Unity Catalog’s three-level namespace.`,
	Long: `A table resides in the third layer of Unity Catalog’s three-level namespace.
  It contains rows of data. To create a table, users must have CREATE_TABLE and
  USE_SCHEMA permissions on the schema, and they must have the USE_CATALOG
  permission on its parent catalog. To query a table, users must have the SELECT
  permission on the table, and they must have the USE_CATALOG permission on its
  parent catalog and the USE_SCHEMA permission on its parent schema.
  
  A table can be managed or external. From an API perspective, a __VIEW__ is a
  particular kind of table (rather than a managed or external table).`,
}

// start delete command

var deleteReq catalog.DeleteTableRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete FULL_NAME",
	Short: `Delete a table.`,
	Long: `Delete a table.
  
  Deletes a table from the specified parent catalog and schema. The caller must
  be the owner of the parent catalog, have the **USE_CATALOG** privilege on the
  parent catalog and be the owner of the parent schema, or be the owner of the
  table and have the **USE_CATALOG** privilege on the parent catalog and the
  **USE_SCHEMA** privilege on the parent schema.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Tables.TableInfoNameToTableIdMap(ctx, catalog.ListTablesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Full name of the table")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have full name of the table")
		}
		deleteReq.FullName = args[0]

		err = w.Tables.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetTableRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().BoolVar(&getReq.IncludeDeltaMetadata, "include-delta-metadata", getReq.IncludeDeltaMetadata, `Whether delta metadata should be included in the response.`)

}

var getCmd = &cobra.Command{
	Use:   "get FULL_NAME",
	Short: `Get a table.`,
	Long: `Get a table.
  
  Gets a table from the metastore for a specific catalog and schema. The caller
  must be a metastore admin, be the owner of the table and have the
  **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA**
  privilege on the parent schema, or be the owner of the table and have the
  **SELECT** privilege on it as well.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Tables.TableInfoNameToTableIdMap(ctx, catalog.ListTablesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Full name of the table")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have full name of the table")
		}
		getReq.FullName = args[0]

		response, err := w.Tables.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq catalog.ListTablesRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().BoolVar(&listReq.IncludeDeltaMetadata, "include-delta-metadata", listReq.IncludeDeltaMetadata, `Whether delta metadata should be included in the response.`)
	listCmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of tables to return (page length).`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque token to send for the next page of results (pagination).`)

}

var listCmd = &cobra.Command{
	Use:   "list CATALOG_NAME SCHEMA_NAME",
	Short: `List tables.`,
	Long: `List tables.
  
  Gets an array of all tables for the current metastore under the parent catalog
  and schema. The caller must be a metastore admin or an owner of (or have the
  **SELECT** privilege on) the table. For the latter case, the caller must also
  be the owner or have the **USE_CATALOG** privilege on the parent catalog and
  the **USE_SCHEMA** privilege on the parent schema. There is no guarantee of a
  specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		listReq.CatalogName = args[0]
		listReq.SchemaName = args[1]

		response, err := w.Tables.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-summaries command

var listSummariesReq catalog.ListSummariesRequest

func init() {
	Cmd.AddCommand(listSummariesCmd)
	// TODO: short flags

	listSummariesCmd.Flags().IntVar(&listSummariesReq.MaxResults, "max-results", listSummariesReq.MaxResults, `Maximum number of tables to return (page length).`)
	listSummariesCmd.Flags().StringVar(&listSummariesReq.PageToken, "page-token", listSummariesReq.PageToken, `Opaque token to send for the next page of results (pagination).`)
	listSummariesCmd.Flags().StringVar(&listSummariesReq.SchemaNamePattern, "schema-name-pattern", listSummariesReq.SchemaNamePattern, `A sql LIKE pattern (% and _) for schema names.`)
	listSummariesCmd.Flags().StringVar(&listSummariesReq.TableNamePattern, "table-name-pattern", listSummariesReq.TableNamePattern, `A sql LIKE pattern (% and _) for table names.`)

}

var listSummariesCmd = &cobra.Command{
	Use:   "list-summaries CATALOG_NAME",
	Short: `List table summaries.`,
	Long: `List table summaries.
  
  Gets an array of summaries for tables for a schema and catalog within the
  metastore. The table summaries returned are either:
  
  * summaries for all tables (within the current metastore and parent catalog
  and schema), when the user is a metastore admin, or: * summaries for all
  tables and schemas (within the current metastore and parent catalog) for which
  the user has ownership or the **SELECT** privilege on the table and ownership
  or **USE_SCHEMA** privilege on the schema, provided that the user also has
  ownership or the **USE_CATALOG** privilege on the parent catalog.
  
  There is no guarantee of a specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Tables.TableInfoNameToTableIdMap(ctx, catalog.ListTablesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Name of parent catalog for tables of interest")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have name of parent catalog for tables of interest")
		}
		listSummariesReq.CatalogName = args[0]

		response, err := w.Tables.ListSummaries(ctx, listSummariesReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service Tables
