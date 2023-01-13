// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package schemas

import (
	"fmt"

	"github.com/databricks/bricks/lib/jsonflag"
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
  list) a table or view in a schema, users must have the USE_SCHEMA data
  permission on the schema and its parent catalog, and they must have the SELECT
  permission on the table or view.`,
}

// start create command

var createReq unitycatalog.CreateSchema
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	// TODO: map via StringToStringVar: properties

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a schema.`,
	Long: `Create a schema.
  
  Creates a new schema for catalog in the Metatastore. The caller must be a
  Metastore admin, or have the CREATE_SCHEMA privilege in the parent catalog.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]
		createReq.CatalogName = args[1]

		response, err := w.Schemas.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq unitycatalog.DeleteSchemaRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete FULL_NAME",
	Short: `Delete a schema.`,
	Long: `Delete a schema.
  
  Deletes the specified schema from the parent catalog. The caller must be the
  owner of the schema or an owner of the parent catalog.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, unitycatalog.ListSchemasRequest{})
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

		err = w.Schemas.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq unitycatalog.GetSchemaRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get FULL_NAME",
	Short: `Get a schema.`,
	Long: `Get a schema.
  
  Gets the specified schema for a catalog in the Metastore. The caller must be a
  Metastore admin, the owner of the schema, or a user that has the USE_SCHEMA
  privilege on the schema.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, unitycatalog.ListSchemasRequest{})
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

		response, err := w.Schemas.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

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
  which the caller has the USE_SCHEMA privilege) will be retrieved.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Schemas.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq unitycatalog.UpdateSchema
var updateJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.CatalogName, "catalog-name", updateReq.CatalogName, `Name of parent Catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of Schema, relative to parent Catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of Schema.`)
	// TODO: map via StringToStringVar: properties
	updateCmd.Flags().StringVar(&updateReq.StorageRoot, "storage-root", updateReq.StorageRoot, `Storage root URL for managed tables within schema.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a schema.`,
	Long: `Update a schema.
  
  Updates a schema for a catalog. The caller must be the owner of the schema. If
  the caller is a Metastore admin, only the __owner__ field can be changed in
  the update. If the __name__ field must be updated, the caller must be a
  Metastore admin or have the CREATE_SCHEMA privilege on the parent catalog.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.FullName = args[0]

		response, err := w.Schemas.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service Schemas
