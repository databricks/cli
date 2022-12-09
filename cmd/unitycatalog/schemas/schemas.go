package schemas

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/unitycatalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "schemas",
	Short: `A schema (also called a database) is the second layer of Unity Catalog’s three-level namespace.`,
	Long: `A schema (also called a database) is the second layer of Unity Catalog’s
  three-level namespace. A schema organizes tables and views. To access (or
  list) a table or view in a schema, users must have the USAGE data permission
  on the schema and its parent catalog, and they must have the SELECT permission
  on the table or view.`,
}

var createReq unitycatalog.CreateSchema

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.CatalogName, "catalog-name", createReq.CatalogName, `[Create:REQ Update:IGN] Name of parent Catalog.`)
	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `[Create,Update:OPT] User-provided free-form text description.`)
	createCmd.Flags().Int64Var(&createReq.CreatedAt, "created-at", createReq.CreatedAt, `[Create,Update:IGN] Time at which this Schema was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.CreatedBy, "created-by", createReq.CreatedBy, `[Create,Update:IGN] Username of Schema creator.`)
	createCmd.Flags().StringVar(&createReq.FullName, "full-name", createReq.FullName, `[Create,Update:IGN] Full name of Schema, in form of <catalog_name>.<schema_name>.`)
	createCmd.Flags().StringVar(&createReq.MetastoreId, "metastore-id", createReq.MetastoreId, `[Create,Update:IGN] Unique identifier of parent Metastore.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `[Create:REQ Update:OPT] Name of Schema, relative to parent Catalog.`)
	createCmd.Flags().StringVar(&createReq.Owner, "owner", createReq.Owner, `[Create:IGN Update:OPT] Username of current owner of Schema.`)
	// TODO: array: privileges
	// TODO: array: properties
	createCmd.Flags().Int64Var(&createReq.UpdatedAt, "updated-at", createReq.UpdatedAt, `[Create,Update:IGN] Time at which this Schema was created, in epoch milliseconds.`)
	createCmd.Flags().StringVar(&createReq.UpdatedBy, "updated-by", createReq.UpdatedBy, `[Create,Update:IGN] Username of user who last modified Schema.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a schema.`,
	Long: `Create a schema.
  
  Creates a new schema for catalog in the Metatastore. The caller must be a
  Metastore admin, or have the CREATE privilege in the parentcatalog.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Schemas.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteReq unitycatalog.DeleteSchemaRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.FullName, "full-name", deleteReq.FullName, `Required.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a schema.`,
	Long: `Delete a schema.
  
  Deletes the specified schema from the parent catalog. The caller must be the
  owner of the schema or an owner of the parent catalog.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Schemas.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getReq unitycatalog.GetSchemaRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.FullName, "full-name", getReq.FullName, `Required.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a schema.`,
	Long: `Get a schema.
  
  Gets the specified schema for a catalog in the Metastore. The caller must be a
  Metastore admin, the owner of the schema, or a user that has the USAGE
  privilege on the schema.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Schemas.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var listReq unitycatalog.ListSchemasRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.CatalogName, "catalog-name", listReq.CatalogName, `Optional.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List schemas.`,
	Long: `List schemas.
  
  Gets an array of schemas for catalog in the Metastore. If the caller is the
  Metastore admin or the owner of the parent catalog, all schemas for the
  catalog will be retrieved. Otherwise, only schemas owned by the caller (or for
  which the caller has the USAGE privilege) will be retrieved.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Schemas.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var updateReq unitycatalog.UpdateSchema

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.CatalogName, "catalog-name", updateReq.CatalogName, `[Create:REQ Update:IGN] Name of parent Catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `[Create,Update:OPT] User-provided free-form text description.`)
	updateCmd.Flags().Int64Var(&updateReq.CreatedAt, "created-at", updateReq.CreatedAt, `[Create,Update:IGN] Time at which this Schema was created, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.CreatedBy, "created-by", updateReq.CreatedBy, `[Create,Update:IGN] Username of Schema creator.`)
	updateCmd.Flags().StringVar(&updateReq.FullName, "full-name", updateReq.FullName, `[Create,Update:IGN] Full name of Schema, in form of <catalog_name>.<schema_name>.`)
	updateCmd.Flags().StringVar(&updateReq.MetastoreId, "metastore-id", updateReq.MetastoreId, `[Create,Update:IGN] Unique identifier of parent Metastore.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `[Create:REQ Update:OPT] Name of Schema, relative to parent Catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `[Create:IGN Update:OPT] Username of current owner of Schema.`)
	// TODO: array: privileges
	// TODO: array: properties
	updateCmd.Flags().Int64Var(&updateReq.UpdatedAt, "updated-at", updateReq.UpdatedAt, `[Create,Update:IGN] Time at which this Schema was created, in epoch milliseconds.`)
	updateCmd.Flags().StringVar(&updateReq.UpdatedBy, "updated-by", updateReq.UpdatedBy, `[Create,Update:IGN] Username of user who last modified Schema.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a schema.`,
	Long: `Update a schema.
  
  Updates a schema for a catalog. The caller must be the owner of the schema. If
  the caller is a Metastore admin, only the __owner__ field can be changed in
  the update. If the __name__ field must be updated, the caller must be a
  Metastore admin or have the CREATE privilege on the parent catalog.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Schemas.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Schemas
