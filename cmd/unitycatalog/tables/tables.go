package tables

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "tables",
	Short: `A table resides in the third layer of Unity Catalog’s three-level namespace.`,
	Long: `A table resides in the third layer of Unity Catalog’s three-level namespace.
  It contains rows of data. To create a table, users must have CREATE and USAGE
  permissions on the schema, and they must have the USAGE permission on its
  parent catalog. To query a table, users must have the SELECT permission on the
  table, and they must have the USAGE permission on its parent schema and
  catalog.
  
  A table can be managed or external.`,
}

var createReq unitycatalog.CreateTable

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.CatalogName, "catalog-name", "", `[Create:REQ Update:IGN] Name of parent Catalog.`)
	// TODO: array: columns
	createCmd.Flags().StringVar(&createReq.Comment, "comment", "", `[Create,Update:OPT] User-provided free-form text description.`)
	createCmd.Flags().Int64Var(&createReq.CreatedAt, "created-at", 0, `[Create,Update:IGN] Time at which this Table was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.CreatedBy, "created-by", "", `[Create,Update:IGN] Username of Table creator.`)
	createCmd.Flags().StringVar(&createReq.DataAccessConfigurationId, "data-access-configuration-id", "", `[Create,Update:IGN] Unique ID of the data_access_configuration to use.`)
	createCmd.Flags().Var(&createReq.DataSourceFormat, "data-source-format", `[Create:REQ Update:OPT] Data source format ("DELTA", "CSV", etc.).`)
	createCmd.Flags().StringVar(&createReq.FullName, "full-name", "", `[Create,Update:IGN] Full name of Table, in form of <catalog_name>.<schema_name>.<table_name>.`)
	createCmd.Flags().StringVar(&createReq.MetastoreId, "metastore-id", "", `[Create,Update:IGN] Unique identifier of parent Metastore.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `[Create:REQ Update:OPT] Name of Table, relative to parent Schema.`)
	createCmd.Flags().StringVar(&createReq.Owner, "owner", "", `[Create: IGN Update:OPT] Username of current owner of Table.`)
	// TODO: array: privileges
	// TODO: array: properties
	createCmd.Flags().StringVar(&createReq.SchemaName, "schema-name", "", `[Create:REQ Update:IGN] Name of parent Schema relative to its parent Catalog.`)
	createCmd.Flags().StringVar(&createReq.SqlPath, "sql-path", "", `[Create,Update:OPT] List of schemes whose objects can be referenced without qualification.`)
	createCmd.Flags().StringVar(&createReq.StorageCredentialName, "storage-credential-name", "", `[Create:OPT Update:IGN] Name of the storage credential this table used.`)
	createCmd.Flags().StringVar(&createReq.StorageLocation, "storage-location", "", `[Create:REQ Update:OPT] Storage root URL for table (for MANAGED, EXTERNAL tables).`)
	createCmd.Flags().StringVar(&createReq.TableId, "table-id", "", `[Create:IGN Update:IGN] Name of Table, relative to parent Schema.`)
	createCmd.Flags().Var(&createReq.TableType, "table-type", `[Create:REQ Update:OPT] Table type ("MANAGED", "EXTERNAL", "VIEW").`)
	createCmd.Flags().Int64Var(&createReq.UpdatedAt, "updated-at", 0, `[Create,Update:IGN] Time at which this Table was last modified, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.UpdatedBy, "updated-by", "", `[Create,Update:IGN] Username of user who last modified the Table.`)
	createCmd.Flags().StringVar(&createReq.ViewDefinition, "view-definition", "", `[Create,Update:OPT] View definition SQL (when table_type == "VIEW").`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a table.`,
	Long: `Create a table.
  
  Creates a new table in the Metastore for a specific catalog and schema.
  **WARNING**: Do not use this API at this time.
  
  The caller must be the owner of or have the USAGE privilege for both the
  parent catalog and schema, or be the owner of the parent schema (or have the
  CREATE privilege on it).
  
  If the new table has a __table_type__ of EXTERNAL specified, the user must be
  a Metastore admin or meet the permissions requirements of the storage
  credential or the external location.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Tables.Create(ctx, createReq)
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

var createStagingTableReq unitycatalog.CreateStagingTable

func init() {
	Cmd.AddCommand(createStagingTableCmd)
	// TODO: short flags

	createStagingTableCmd.Flags().StringVar(&createStagingTableReq.CatalogName, "catalog-name", "", `[Create:REQ] Name of parent Catalog.`)
	createStagingTableCmd.Flags().StringVar(&createStagingTableReq.Id, "id", "", `[Create:IGN] Unique id generated for the staging table.`)
	createStagingTableCmd.Flags().StringVar(&createStagingTableReq.Name, "name", "", `[Create:REQ] Name of Table, relative to parent Schema.`)
	createStagingTableCmd.Flags().StringVar(&createStagingTableReq.SchemaName, "schema-name", "", `[Create:REQ] Name of parent Schema relative to its parent Catalog.`)
	createStagingTableCmd.Flags().StringVar(&createStagingTableReq.StagingLocation, "staging-location", "", `[Create:IGN] URI generated for the staging table.`)

}

var createStagingTableCmd = &cobra.Command{
	Use:   "create-staging-table",
	Short: `Create a staging table.`,
	Long: `Create a staging table.
  
  Creates a new staging table for a schema. The caller must have both the USAGE
  privilege on the parent Catalog and the USAGE and CREATE privileges on the
  parent schema.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Tables.CreateStagingTable(ctx, createStagingTableReq)
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

var deleteReq unitycatalog.DeleteTableRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.FullName, "full-name", "", `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a table.`,
	Long: `Delete a table.
  
  Deletes a table from the specified parent catalog and schema. The caller must
  be the owner of the parent catalog, have the USAGE privilege on the parent
  catalog and be the owner of the parent schema, or be the owner of the table
  and have the USAGE privilege on both the parent catalog and schema.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Tables.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq unitycatalog.GetTableRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.FullName, "full-name", "", `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a table.`,
	Long: `Get a table.
  
  Gets a table from the Metastore for a specific catalog and schema. The caller
  must be a Metastore admin, be the owner of the table and have the USAGE
  privilege on both the parent catalog and schema, or be the owner of the table
  and have the SELECT privilege on it as well.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Tables.Get(ctx, getReq)
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

var listReq unitycatalog.ListTablesRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.CatalogName, "catalog-name", "", `Required.`)
	listCmd.Flags().StringVar(&listReq.SchemaName, "schema-name", "", `Required (for now -- may be optional for wildcard search in future).`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List tables.`,
	Long: `List tables.
  
  Gets an array of all tables for the current Metastore under the parent catalog
  and schema. The caller must be a Metastore admin or an owner of (or have the
  SELECT privilege on) the table. For the latter case, the caller must also be
  the owner or have the USAGE privilege on the parent catalog and schema.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Tables.ListAll(ctx, listReq)
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

var tableSummariesReq unitycatalog.TableSummariesRequest

func init() {
	Cmd.AddCommand(tableSummariesCmd)
	// TODO: short flags

	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.CatalogName, "catalog-name", "", `Required.`)
	tableSummariesCmd.Flags().IntVar(&tableSummariesReq.MaxResults, "max-results", 0, `Optional.`)
	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.PageToken, "page-token", "", `Optional.`)
	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.SchemaNamePattern, "schema-name-pattern", "", `Optional.`)
	tableSummariesCmd.Flags().StringVar(&tableSummariesReq.TableNamePattern, "table-name-pattern", "", `Optional.`)

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
  USAGE privilege on the Schema, provided that the user also has ownership or
  the USAGE privilege on the parent Catalog`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Tables.TableSummaries(ctx, tableSummariesReq)
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

var updateReq unitycatalog.UpdateTable

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.CatalogName, "catalog-name", "", `[Create:REQ Update:IGN] Name of parent Catalog.`)
	// TODO: array: columns
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", "", `[Create,Update:OPT] User-provided free-form text description.`)
	updateCmd.Flags().Int64Var(&updateReq.CreatedAt, "created-at", 0, `[Create,Update:IGN] Time at which this Table was created, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.CreatedBy, "created-by", "", `[Create,Update:IGN] Username of Table creator.`)
	updateCmd.Flags().StringVar(&updateReq.DataAccessConfigurationId, "data-access-configuration-id", "", `[Create,Update:IGN] Unique ID of the data_access_configuration to use.`)
	updateCmd.Flags().Var(&updateReq.DataSourceFormat, "data-source-format", `[Create:REQ Update:OPT] Data source format ("DELTA", "CSV", etc.).`)
	updateCmd.Flags().StringVar(&updateReq.FullName, "full-name", "", `[Create,Update:IGN] Full name of Table, in form of <catalog_name>.<schema_name>.<table_name>.`)
	updateCmd.Flags().StringVar(&updateReq.MetastoreId, "metastore-id", "", `[Create,Update:IGN] Unique identifier of parent Metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", "", `[Create:REQ Update:OPT] Name of Table, relative to parent Schema.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", "", `[Create: IGN Update:OPT] Username of current owner of Table.`)
	// TODO: array: privileges
	// TODO: array: properties
	updateCmd.Flags().StringVar(&updateReq.SchemaName, "schema-name", "", `[Create:REQ Update:IGN] Name of parent Schema relative to its parent Catalog.`)
	updateCmd.Flags().StringVar(&updateReq.SqlPath, "sql-path", "", `[Create,Update:OPT] List of schemes whose objects can be referenced without qualification.`)
	updateCmd.Flags().StringVar(&updateReq.StorageCredentialName, "storage-credential-name", "", `[Create:OPT Update:IGN] Name of the storage credential this table used.`)
	updateCmd.Flags().StringVar(&updateReq.StorageLocation, "storage-location", "", `[Create:REQ Update:OPT] Storage root URL for table (for MANAGED, EXTERNAL tables).`)
	updateCmd.Flags().StringVar(&updateReq.TableId, "table-id", "", `[Create:IGN Update:IGN] Name of Table, relative to parent Schema.`)
	updateCmd.Flags().Var(&updateReq.TableType, "table-type", `[Create:REQ Update:OPT] Table type ("MANAGED", "EXTERNAL", "VIEW").`)
	updateCmd.Flags().Int64Var(&updateReq.UpdatedAt, "updated-at", 0, `[Create,Update:IGN] Time at which this Table was last modified, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.UpdatedBy, "updated-by", "", `[Create,Update:IGN] Username of user who last modified the Table.`)
	updateCmd.Flags().StringVar(&updateReq.ViewDefinition, "view-definition", "", `[Create,Update:OPT] View definition SQL (when table_type == "VIEW").`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a table.`,
	Long: `Update a table.
  
  Updates a table in the specified catalog and schema. The caller must be the
  owner of have the USAGE privilege on the parent catalog and schema, or, if
  changing the owner, be a Metastore admin as well.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Tables.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
