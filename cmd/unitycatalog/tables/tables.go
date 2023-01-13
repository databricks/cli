// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package tables

import (
	"fmt"

	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
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
  
  A table can be managed or external.`,
}

// start delete command

var deleteReq unitycatalog.DeleteTableRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete FULL_NAME",
	Short: `Delete a table.`,
	Long: `Delete a table.
  
  Deletes a table from the specified parent catalog and schema. The caller must
  be the owner of the parent catalog, have the USE_CATALOG privilege on the
  parent catalog and be the owner of the parent schema, or be the owner of the
  table and have the USE_CATALOG privilege on the parent catalog and the
  USE_SCHEMA privilege on the parent schema.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Tables.TableInfoNameToTableIdMap(ctx, unitycatalog.ListTablesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
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

var getReq unitycatalog.GetTableRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get FULL_NAME",
	Short: `Get a table.`,
	Long: `Get a table.
  
  Gets a table from the Metastore for a specific catalog and schema. The caller
  must be a Metastore admin, be the owner of the table and have the USE_CATALOG
  privilege on the parent catalog and the USE_SCHEMA privilege on the parent
  schema, or be the owner of the table and have the SELECT privilege on it as
  well.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Tables.TableInfoNameToTableIdMap(ctx, unitycatalog.ListTablesRequest{})
			if err != nil {
				return err
			}
			id, err := ui.PromptValue(cmd.InOrStdin(), names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
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

var listReq unitycatalog.ListTablesRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.CatalogName, "catalog-name", listReq.CatalogName, `Required.`)
	listCmd.Flags().StringVar(&listReq.SchemaName, "schema-name", listReq.SchemaName, `Required (for now -- may be optional for wildcard search in future).`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List tables.`,
	Long: `List tables.
  
  Gets an array of all tables for the current Metastore under the parent catalog
  and schema. The caller must be a Metastore admin or an owner of (or have the
  SELECT privilege on) the table. For the latter case, the caller must also be
  the owner or have the USE_CATALOG privilege on the parent catalog and the
  USE_SCHEMA privilege on the parent schema.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Tables.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start table-summaries command

var tableSummariesReq unitycatalog.TableSummariesRequest

func init() {
	Cmd.AddCommand(tableSummariesCmd)
	// TODO: short flags

	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.CatalogName, "catalog-name", tableSummariesReq.CatalogName, `Required.`)
	tableSummariesCmd.Flags().IntVar(&tableSummariesReq.MaxResults, "max-results", tableSummariesReq.MaxResults, `Optional.`)
	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.PageToken, "page-token", tableSummariesReq.PageToken, `Optional.`)
	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.SchemaNamePattern, "schema-name-pattern", tableSummariesReq.SchemaNamePattern, `Optional.`)
	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.TableNamePattern, "table-name-pattern", tableSummariesReq.TableNamePattern, `Optional.`)

}

var tableSummariesCmd = &cobra.Command{
	Use:   "table-summaries",
	Short: `List table summaries.`,
	Long: `List table summaries.
  
  Gets an array of summaries for tables for a schema and catalog within the
  Metastore. The table summaries returned are either:
  
  * summaries for all tables (within the current Metastore and parent catalog
  and schema), when the user is a Metastore admin, or: * summaries for all
  tables and schemas (within the current Metastore and parent catalog) for which
  the user has ownership or the SELECT privilege on the Table and ownership or
  USE_SCHEMA privilege on the Schema, provided that the user also has ownership
  or the USE_CATALOG privilege on the parent Catalog`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Tables.TableSummaries(ctx, tableSummariesReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service Tables
