// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package schemas

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "schemas",
	Short: `A schema (also called a database) is the second layer of Unity Catalog’s three-level namespace.`,
	Long: `A schema (also called a database) is the second layer of Unity Catalog’s
  three-level namespace. A schema organizes tables, views and functions. To
  access (or list) a table or view in a schema, users must have the USE_SCHEMA
  data permission on the schema and its parent catalog, and they must have the
  SELECT permission on the table or view.`,
}

// start create command

var createReq catalog.CreateSchema
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	// TODO: map via StringToStringVar: properties
	createCmd.Flags().StringVar(&createReq.StorageRoot, "storage-root", createReq.StorageRoot, `Storage root URL for managed tables within schema.`)

}

var createCmd = &cobra.Command{
	Use:   "create NAME CATALOG_NAME",
	Short: `Create a schema.`,
	Long: `Create a schema.
  
  Creates a new schema for catalog in the Metatastore. The caller must be a
  metastore admin, or have the **CREATE_SCHEMA** privilege in the parent
  catalog.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
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
			createReq.Name = args[0]
			createReq.CatalogName = args[1]
		}

		response, err := w.Schemas.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq catalog.DeleteSchemaRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete FULL_NAME",
	Short: `Delete a schema.`,
	Long: `Delete a schema.
  
  Deletes the specified schema from the parent catalog. The caller must be the
  owner of the schema or an owner of the parent catalog.`,

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
				promptSpinner <- "No FULL_NAME argument specified. Loading names for Schemas drop-down."
				names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, catalog.ListSchemasRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Schemas drop-down. Please manually specify required arguments")
				}
				id, err := cmdio.Select(ctx, names, "Full name of the schema")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have full name of the schema")
			}
			deleteReq.FullName = args[0]
		}

		err = w.Schemas.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq catalog.GetSchemaRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get FULL_NAME",
	Short: `Get a schema.`,
	Long: `Get a schema.
  
  Gets the specified schema within the metastore. The caller must be a metastore
  admin, the owner of the schema, or a user that has the **USE_SCHEMA**
  privilege on the schema.`,

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
				promptSpinner <- "No FULL_NAME argument specified. Loading names for Schemas drop-down."
				names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, catalog.ListSchemasRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Schemas drop-down. Please manually specify required arguments")
				}
				id, err := cmdio.Select(ctx, names, "Full name of the schema")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have full name of the schema")
			}
			getReq.FullName = args[0]
		}

		response, err := w.Schemas.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq catalog.ListSchemasRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var listCmd = &cobra.Command{
	Use:   "list CATALOG_NAME",
	Short: `List schemas.`,
	Long: `List schemas.
  
  Gets an array of schemas for a catalog in the metastore. If the caller is the
  metastore admin or the owner of the parent catalog, all schemas for the
  catalog will be retrieved. Otherwise, only schemas owned by the caller (or for
  which the caller has the **USE_SCHEMA** privilege) will be retrieved. There is
  no guarantee of a specific ordering of the elements in the array.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
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
			listReq.CatalogName = args[0]
		}

		response, err := w.Schemas.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update command

var updateReq catalog.UpdateSchema
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of schema, relative to parent catalog.`)
	updateCmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of schema.`)
	// TODO: map via StringToStringVar: properties

}

var updateCmd = &cobra.Command{
	Use:   "update FULL_NAME",
	Short: `Update a schema.`,
	Long: `Update a schema.
  
  Updates a schema for a catalog. The caller must be the owner of the schema or
  a metastore admin. If the caller is a metastore admin, only the __owner__
  field can be changed in the update. If the __name__ field must be updated, the
  caller must be a metastore admin or have the **CREATE_SCHEMA** privilege on
  the parent catalog.`,

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
				promptSpinner <- "No FULL_NAME argument specified. Loading names for Schemas drop-down."
				names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, catalog.ListSchemasRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Schemas drop-down. Please manually specify required arguments")
				}
				id, err := cmdio.Select(ctx, names, "Full name of the schema")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have full name of the schema")
			}
			updateReq.FullName = args[0]
		}

		response, err := w.Schemas.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service Schemas
