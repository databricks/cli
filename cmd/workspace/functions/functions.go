// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package functions

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
		Use:   "functions",
		Short: `Functions implement User-Defined Functions (UDFs) in Unity Catalog.`,
		Long: `Functions implement User-Defined Functions (UDFs) in Unity Catalog.
  
  The function implementation can be any SQL expression or Query, and it can be
  invoked wherever a table reference is allowed in a query. In Unity Catalog, a
  function resides at the same level as a table, so it can be referenced with
  the form __catalog_name__.__schema_name__.__function_name__.`,
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
	*catalog.CreateFunction,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateFunction
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Comment, "comment", createReq.Comment, `User-provided free-form text description.`)
	cmd.Flags().StringVar(&createReq.ExternalLanguage, "external-language", createReq.ExternalLanguage, `External function language.`)
	cmd.Flags().StringVar(&createReq.ExternalName, "external-name", createReq.ExternalName, `External function name.`)
	// TODO: map via StringToStringVar: properties
	cmd.Flags().StringVar(&createReq.SqlPath, "sql-path", createReq.SqlPath, `List of schemes whose objects can be referenced without qualification.`)

	cmd.Use = "create"
	cmd.Short = `Create a function.`
	cmd.Long = `Create a function.
  
  Creates a new function
  
  The user must have the following permissions in order for the function to be
  created: - **USE_CATALOG** on the function's parent catalog - **USE_SCHEMA**
  and **CREATE_FUNCTION** on the function's parent schema`

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

		response, err := w.Functions.Create(ctx, createReq)
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
	*catalog.DeleteFunctionRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteFunctionRequest

	// TODO: short flags

	cmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if the function is notempty.`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a function.`
	cmd.Long = `Delete a function.
  
  Deletes the function that matches the supplied name. For the deletion to
  succeed, the user must satisfy one of the following conditions: - Is the owner
  of the function's parent catalog - Is the owner of the function's parent
  schema and have the **USE_CATALOG** privilege on its parent catalog - Is the
  owner of the function itself and have both the **USE_CATALOG** privilege on
  its parent catalog and the **USE_SCHEMA** privilege on its parent schema`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		err = w.Functions.Delete(ctx, deleteReq)
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
	*catalog.GetFunctionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetFunctionRequest

	// TODO: short flags

	cmd.Use = "get NAME"
	cmd.Short = `Get a function.`
	cmd.Long = `Get a function.
  
  Gets a function from within a parent catalog and schema. For the fetch to
  succeed, the user must satisfy one of the following requirements: - Is a
  metastore admin - Is an owner of the function's parent catalog - Have the
  **USE_CATALOG** privilege on the function's parent catalog and be the owner of
  the function - Have the **USE_CATALOG** privilege on the function's parent
  catalog, the **USE_SCHEMA** privilege on the function's parent schema, and the
  **EXECUTE** privilege on the function itself`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.Name = args[0]

		response, err := w.Functions.Get(ctx, getReq)
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
	*catalog.ListFunctionsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListFunctionsRequest

	// TODO: short flags

	cmd.Use = "list CATALOG_NAME SCHEMA_NAME"
	cmd.Short = `List functions.`
	cmd.Long = `List functions.
  
  List functions within the specified parent catalog and schema. If the user is
  a metastore admin, all functions are returned in the output list. Otherwise,
  the user must have the **USE_CATALOG** privilege on the catalog and the
  **USE_SCHEMA** privilege on the schema, and the output list contains only
  functions for which either the user has the **EXECUTE** privilege or the user
  is the owner. There is no guarantee of a specific ordering of the elements in
  the array.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		listReq.CatalogName = args[0]
		listReq.SchemaName = args[1]

		response, err := w.Functions.ListAll(ctx, listReq)
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
	*catalog.UpdateFunction,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateFunction

	// TODO: short flags

	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of function.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update a function.`
	cmd.Long = `Update a function.
  
  Updates the function that matches the supplied name. Only the owner of the
  function can be updated. If the user is not a metastore admin, the user must
  be a member of the group that is the new function owner. - Is a metastore
  admin - Is the owner of the function's parent catalog - Is the owner of the
  function's parent schema and has the **USE_CATALOG** privilege on its parent
  catalog - Is the owner of the function itself and has the **USE_CATALOG**
  privilege on its parent catalog as well as the **USE_SCHEMA** privilege on the
  function's parent schema.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		updateReq.Name = args[0]

		response, err := w.Functions.Update(ctx, updateReq)
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

// end service Functions
