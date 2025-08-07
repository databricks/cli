// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package functions

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

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
	*catalog.CreateFunctionRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq catalog.CreateFunctionRequest
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a function.`
	cmd.Long = `Create a function.
  
  **WARNING: This API is experimental and will change in future versions**
  
  Creates a new function
  
  The user must have the following permissions in order for the function to be
  created: - **USE_CATALOG** on the function's parent catalog - **USE_SCHEMA**
  and **CREATE_FUNCTION** on the function's parent schema`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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

	cmd.Flags().BoolVar(&deleteReq.Force, "force", deleteReq.Force, `Force deletion even if the function is notempty.`)

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a function.`
	cmd.Long = `Delete a function.
  
  Deletes the function that matches the supplied name. For the deletion to
  succeed, the user must satisfy one of the following conditions: - Is the owner
  of the function's parent catalog - Is the owner of the function's parent
  schema and have the **USE_CATALOG** privilege on its parent catalog - Is the
  owner of the function itself and have both the **USE_CATALOG** privilege on
  its parent catalog and the **USE_SCHEMA** privilege on its parent schema

  Arguments:
    NAME: The fully-qualified name of the function (of the form
      __catalog_name__.__schema_name__.__function__name__).`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No NAME argument specified. Loading names for Functions drop-down."
			names, err := w.Functions.FunctionInfoNameToFullNameMap(ctx, catalog.ListFunctionsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Functions drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The fully-qualified name of the function (of the form __catalog_name__.__schema_name__.__function__name__)")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the fully-qualified name of the function (of the form __catalog_name__.__schema_name__.__function__name__)")
		}
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

	cmd.Flags().BoolVar(&getReq.IncludeBrowse, "include-browse", getReq.IncludeBrowse, `Whether to include functions in the response for which the principal can only access selective metadata for.`)

	cmd.Use = "get NAME"
	cmd.Short = `Get a function.`
	cmd.Long = `Get a function.
  
  Gets a function from within a parent catalog and schema. For the fetch to
  succeed, the user must satisfy one of the following requirements: - Is a
  metastore admin - Is an owner of the function's parent catalog - Have the
  **USE_CATALOG** privilege on the function's parent catalog and be the owner of
  the function - Have the **USE_CATALOG** privilege on the function's parent
  catalog, the **USE_SCHEMA** privilege on the function's parent schema, and the
  **EXECUTE** privilege on the function itself

  Arguments:
    NAME: The fully-qualified name of the function (of the form
      __catalog_name__.__schema_name__.__function__name__).`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No NAME argument specified. Loading names for Functions drop-down."
			names, err := w.Functions.FunctionInfoNameToFullNameMap(ctx, catalog.ListFunctionsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Functions drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The fully-qualified name of the function (of the form __catalog_name__.__schema_name__.__function__name__)")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the fully-qualified name of the function (of the form __catalog_name__.__schema_name__.__function__name__)")
		}
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

	cmd.Flags().BoolVar(&listReq.IncludeBrowse, "include-browse", listReq.IncludeBrowse, `Whether to include functions in the response for which the principal can only access selective metadata for.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of functions to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list CATALOG_NAME SCHEMA_NAME"
	cmd.Short = `List functions.`
	cmd.Long = `List functions.
  
  List functions within the specified parent catalog and schema. If the user is
  a metastore admin, all functions are returned in the output list. Otherwise,
  the user must have the **USE_CATALOG** privilege on the catalog and the
  **USE_SCHEMA** privilege on the schema, and the output list contains only
  functions for which either the user has the **EXECUTE** privilege or the user
  is the owner. There is no guarantee of a specific ordering of the elements in
  the array.

  Arguments:
    CATALOG_NAME: Name of parent catalog for functions of interest.
    SCHEMA_NAME: Parent schema of functions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.CatalogName = args[0]
		listReq.SchemaName = args[1]

		response := w.Functions.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
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
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
  function's parent schema.

  Arguments:
    NAME: The fully-qualified name of the function (of the form
      __catalog_name__.__schema_name__.__function__name__).`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No NAME argument specified. Loading names for Functions drop-down."
			names, err := w.Functions.FunctionInfoNameToFullNameMap(ctx, catalog.ListFunctionsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Functions drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The fully-qualified name of the function (of the form __catalog_name__.__schema_name__.__function__name__)")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the fully-qualified name of the function (of the form __catalog_name__.__schema_name__.__function__name__)")
		}
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

// end service Functions
