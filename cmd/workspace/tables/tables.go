// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package tables

import (
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
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newExists())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListSummaries())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*catalog.DeleteTableRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq catalog.DeleteTableRequest

	cmd.Use = "delete FULL_NAME"
	cmd.Short = `Delete a table.`
	cmd.Long = `Delete a table.
  
  Deletes a table from the specified parent catalog and schema. The caller must
  be the owner of the parent catalog, have the **USE_CATALOG** privilege on the
  parent catalog and be the owner of the parent schema, or be the owner of the
  table and have the **USE_CATALOG** privilege on the parent catalog and the
  **USE_SCHEMA** privilege on the parent schema.

  Arguments:
    FULL_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.FullName = args[0]

		err = w.Tables.Delete(ctx, deleteReq)
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

// start exists command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var existsOverrides []func(
	*cobra.Command,
	*catalog.ExistsRequest,
)

func newExists() *cobra.Command {
	cmd := &cobra.Command{}

	var existsReq catalog.ExistsRequest

	cmd.Use = "exists FULL_NAME"
	cmd.Short = `Get boolean reflecting if table exists.`
	cmd.Long = `Get boolean reflecting if table exists.
  
  Gets if a table exists in the metastore for a specific catalog and schema. The
  caller must satisfy one of the following requirements: * Be a metastore admin
  * Be the owner of the parent catalog * Be the owner of the parent schema and
  have the USE_CATALOG privilege on the parent catalog * Have the
  **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA**
  privilege on the parent schema, and either be the table owner or have the
  SELECT privilege on the table. * Have BROWSE privilege on the parent catalog *
  Have BROWSE privilege on the parent schema.

  Arguments:
    FULL_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		existsReq.FullName = args[0]

		response, err := w.Tables.Exists(ctx, existsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range existsOverrides {
		fn(cmd, &existsReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*catalog.GetTableRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq catalog.GetTableRequest

	cmd.Flags().BoolVar(&getReq.IncludeBrowse, "include-browse", getReq.IncludeBrowse, `Whether to include tables in the response for which the principal can only access selective metadata for.`)
	cmd.Flags().BoolVar(&getReq.IncludeDeltaMetadata, "include-delta-metadata", getReq.IncludeDeltaMetadata, `Whether delta metadata should be included in the response.`)
	cmd.Flags().BoolVar(&getReq.IncludeManifestCapabilities, "include-manifest-capabilities", getReq.IncludeManifestCapabilities, `Whether to include a manifest containing table capabilities in the response.`)

	cmd.Use = "get FULL_NAME"
	cmd.Short = `Get a table.`
	cmd.Long = `Get a table.
  
  Gets a table from the metastore for a specific catalog and schema. The caller
  must satisfy one of the following requirements: * Be a metastore admin * Be
  the owner of the parent catalog * Be the owner of the parent schema and have
  the USE_CATALOG privilege on the parent catalog * Have the **USE_CATALOG**
  privilege on the parent catalog and the **USE_SCHEMA** privilege on the parent
  schema, and either be the table owner or have the SELECT privilege on the
  table.

  Arguments:
    FULL_NAME: Full name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.FullName = args[0]

		response, err := w.Tables.Get(ctx, getReq)
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
	*catalog.ListTablesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListTablesRequest

	cmd.Flags().BoolVar(&listReq.IncludeBrowse, "include-browse", listReq.IncludeBrowse, `Whether to include tables in the response for which the principal can only access selective metadata for.`)
	cmd.Flags().BoolVar(&listReq.IncludeManifestCapabilities, "include-manifest-capabilities", listReq.IncludeManifestCapabilities, `Whether to include a manifest containing table capabilities in the response.`)
	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of tables to return.`)
	cmd.Flags().BoolVar(&listReq.OmitColumns, "omit-columns", listReq.OmitColumns, `Whether to omit the columns of the table from the response or not.`)
	cmd.Flags().BoolVar(&listReq.OmitProperties, "omit-properties", listReq.OmitProperties, `Whether to omit the properties of the table from the response or not.`)
	cmd.Flags().BoolVar(&listReq.OmitUsername, "omit-username", listReq.OmitUsername, `Whether to omit the username of the table (e.g.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque token to send for the next page of results (pagination).`)

	cmd.Use = "list CATALOG_NAME SCHEMA_NAME"
	cmd.Short = `List tables.`
	cmd.Long = `List tables.
  
  Gets an array of all tables for the current metastore under the parent catalog
  and schema. The caller must be a metastore admin or an owner of (or have the
  **SELECT** privilege on) the table. For the latter case, the caller must also
  be the owner or have the **USE_CATALOG** privilege on the parent catalog and
  the **USE_SCHEMA** privilege on the parent schema. There is no guarantee of a
  specific ordering of the elements in the array.

  Arguments:
    CATALOG_NAME: Name of parent catalog for tables of interest.
    SCHEMA_NAME: Parent schema of tables.`

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

		response := w.Tables.List(ctx, listReq)
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

// start list-summaries command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSummariesOverrides []func(
	*cobra.Command,
	*catalog.ListSummariesRequest,
)

func newListSummaries() *cobra.Command {
	cmd := &cobra.Command{}

	var listSummariesReq catalog.ListSummariesRequest

	cmd.Flags().BoolVar(&listSummariesReq.IncludeManifestCapabilities, "include-manifest-capabilities", listSummariesReq.IncludeManifestCapabilities, `Whether to include a manifest containing table capabilities in the response.`)
	cmd.Flags().IntVar(&listSummariesReq.MaxResults, "max-results", listSummariesReq.MaxResults, `Maximum number of summaries for tables to return.`)
	cmd.Flags().StringVar(&listSummariesReq.PageToken, "page-token", listSummariesReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)
	cmd.Flags().StringVar(&listSummariesReq.SchemaNamePattern, "schema-name-pattern", listSummariesReq.SchemaNamePattern, `A sql LIKE pattern (% and _) for schema names.`)
	cmd.Flags().StringVar(&listSummariesReq.TableNamePattern, "table-name-pattern", listSummariesReq.TableNamePattern, `A sql LIKE pattern (% and _) for table names.`)

	cmd.Use = "list-summaries CATALOG_NAME"
	cmd.Short = `List table summaries.`
	cmd.Long = `List table summaries.
  
  Gets an array of summaries for tables for a schema and catalog within the
  metastore. The table summaries returned are either:
  
  * summaries for tables (within the current metastore and parent catalog and
  schema), when the user is a metastore admin, or: * summaries for tables and
  schemas (within the current metastore and parent catalog) for which the user
  has ownership or the **SELECT** privilege on the table and ownership or
  **USE_SCHEMA** privilege on the schema, provided that the user also has
  ownership or the **USE_CATALOG** privilege on the parent catalog.
  
  There is no guarantee of a specific ordering of the elements in the array.

  Arguments:
    CATALOG_NAME: Name of parent catalog for tables of interest.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listSummariesReq.CatalogName = args[0]

		response := w.Tables.ListSummaries(ctx, listSummariesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSummariesOverrides {
		fn(cmd, &listSummariesReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*catalog.UpdateTableRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq catalog.UpdateTableRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Owner, "owner", updateReq.Owner, `Username of current owner of table.`)

	cmd.Use = "update FULL_NAME"
	cmd.Short = `Update a table owner.`
	cmd.Long = `Update a table owner.
  
  Change the owner of the table. The caller must be the owner of the parent
  catalog, have the **USE_CATALOG** privilege on the parent catalog and be the
  owner of the parent schema, or be the owner of the table and have the
  **USE_CATALOG** privilege on the parent catalog and the **USE_SCHEMA**
  privilege on the parent schema.

  Arguments:
    FULL_NAME: Full name of the table.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

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
		updateReq.FullName = args[0]

		err = w.Tables.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
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

// end service Tables
