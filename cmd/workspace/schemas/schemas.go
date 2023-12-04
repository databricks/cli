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

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemas",
		Short: `A schema (also called a database) is the second layer of Unity Catalog’s three-level namespace.`,
		Long: `A schema (also called a database) is the second layer of Unity Catalog’s
  three-level namespace. A schema organizes tables, views and functions. To
  access (or list) a table or view in a schema, users must have the USE_SCHEMA
  data permission on the schema and its parent catalog, and they must have the
  SELECT permission on the table or view.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
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
	*catalog.CreateSchema,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateSchema
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	// TODO: map via StringToStringVar: properties
	cmd.Flags().StringVar(&createReq.StorageRoot, "storage-root", createReq.StorageRoot, `Storage root URL for managed tables within schema.`)

	cmd.Use = "create NAME CATALOG_NAME"
	cmd.Short = `Create a schema.`
	cmd.Long = `Create a schema.
  
  Creates a new schema for catalog in the Metatastore. The caller must be a
  metastore admin, or have the **CREATE_SCHEMA** privilege in the parent
  catalog.

  Arguments:
    NAME: Name of schema, relative to parent catalog.
    CATALOG_NAME: Name of parent catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'catalog_name' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createReq.CatalogName = args[1]
		}

		response, err := w.Schemas.Create(ctx, createReq)
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
	*catalog.DeleteSchemaRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteSchemaRequest

	// TODO: short flags

	cmd.Use = "delete FULL_NAME"
	cmd.Short = `Delete a schema.`
	cmd.Long = `Delete a schema.
  
  Deletes the specified schema from the parent catalog. The caller must be the
  owner of the schema or an owner of the parent catalog.

  Arguments:
    FULL_NAME: Full name of the schema.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME argument specified. Loading names for Schemas drop-down."
			names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, catalog.ListSchemasRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Schemas drop-down. Please manually specify required arguments. Original error: %w", err)
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

		err = w.Schemas.Delete(ctx, deleteReq)
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
	*catalog.GetSchemaRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetSchemaRequest

	// TODO: short flags

	cmd.Use = "get FULL_NAME"
	cmd.Short = `Get a schema.`
	cmd.Long = `Get a schema.
  
  Gets the specified schema within the metastore. The caller must be a metastore
  admin, the owner of the schema, or a user that has the **USE_SCHEMA**
  privilege on the schema.

  Arguments:
    FULL_NAME: Full name of the schema.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No FULL_NAME argument specified. Loading names for Schemas drop-down."
			names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, catalog.ListSchemasRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Schemas drop-down. Please manually specify required arguments. Original error: %w", err)
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

		response, err := w.Schemas.Get(ctx, getReq)
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
	*catalog.ListSchemasRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListSchemasRequest

	// TODO: short flags

	cmd.Use = "list CATALOG_NAME"
	cmd.Short = `List schemas.`
	cmd.Long = `List schemas.
  
  Gets an array of schemas for a catalog in the metastore. If the caller is the
  metastore admin or the owner of the parent catalog, all schemas for the
  catalog will be retrieved. Otherwise, only schemas owned by the caller (or for
  which the caller has the **USE_SCHEMA** privilege) will be retrieved. There is
  no guarantee of a specific ordering of the elements in the array.

  Arguments:
    CATALOG_NAME: Parent catalog for schemas of interest.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		listReq.CatalogName = args[0]

		response, err := w.Schemas.ListAll(ctx, listReq)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateSchema,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateSchema
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Comment, "comment", updateReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().Var(&updateReq.EnablePredictiveOptimization, "enable-predictive-optimization", `Whether predictive optimization should be enabled for this object and objects under it.`)
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Name of schema, relative to parent catalog.`)
	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of schema.`)
	// TODO: map via StringToStringVar: properties

	cmd.Use = "update FULL_NAME"
	cmd.Short = `Update a schema.`
	cmd.Long = `Update a schema.
  
  Updates a schema for a catalog. The caller must be the owner of the schema or
  a metastore admin. If the caller is a metastore admin, only the __owner__
  field can be changed in the update. If the __name__ field must be updated, the
  caller must be a metastore admin or have the **CREATE_SCHEMA** privilege on
  the parent catalog.

  Arguments:
    FULL_NAME: Full name of the schema.`

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
			promptSpinner <- "No FULL_NAME argument specified. Loading names for Schemas drop-down."
			names, err := w.Schemas.SchemaInfoNameToFullNameMap(ctx, catalog.ListSchemasRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Schemas drop-down. Please manually specify required arguments. Original error: %w", err)
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

		response, err := w.Schemas.Update(ctx, updateReq)
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

// end service Schemas
