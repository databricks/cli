// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package system_schemas

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
		Use:   "system-schemas",
		Short: `A system schema is a schema that lives within the system catalog.`,
		Long: `A system schema is a schema that lives within the system catalog. A system
  schema may contain information about customer usage of Unity Catalog such as
  audit-logs, billing-logs, lineage information, etc.`,
		GroupID: "catalog",
		Annotations: map[string]string{
			"package": "catalog",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newDisable())
	cmd.AddCommand(newEnable())
	cmd.AddCommand(newList())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start disable command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var disableOverrides []func(
	*cobra.Command,
	*catalog.DisableRequest,
)

func newDisable() *cobra.Command {
	cmd := &cobra.Command{}

	var disableReq catalog.DisableRequest

	cmd.Use = "disable METASTORE_ID SCHEMA_NAME"
	cmd.Short = `Disable a system schema.`
	cmd.Long = `Disable a system schema.
  
  Disables the system schema and removes it from the system catalog. The caller
  must be an account admin or a metastore admin.

  Arguments:
    METASTORE_ID: The metastore ID under which the system schema lives.
    SCHEMA_NAME: Full name of the system schema.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		disableReq.MetastoreId = args[0]
		disableReq.SchemaName = args[1]

		err = w.SystemSchemas.Disable(ctx, disableReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range disableOverrides {
		fn(cmd, &disableReq)
	}

	return cmd
}

// start enable command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var enableOverrides []func(
	*cobra.Command,
	*catalog.EnableRequest,
)

func newEnable() *cobra.Command {
	cmd := &cobra.Command{}

	var enableReq catalog.EnableRequest
	var enableJson flags.JsonFlag

	cmd.Flags().Var(&enableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&enableReq.CatalogName, "catalog-name", enableReq.CatalogName, `the catalog for which the system schema is to enabled in.`)

	cmd.Use = "enable METASTORE_ID SCHEMA_NAME"
	cmd.Short = `Enable a system schema.`
	cmd.Long = `Enable a system schema.
  
  Enables the system schema and adds it to the system catalog. The caller must
  be an account admin or a metastore admin.

  Arguments:
    METASTORE_ID: The metastore ID under which the system schema lives.
    SCHEMA_NAME: Full name of the system schema.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := enableJson.Unmarshal(&enableReq)
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
		enableReq.MetastoreId = args[0]
		enableReq.SchemaName = args[1]

		err = w.SystemSchemas.Enable(ctx, enableReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range enableOverrides {
		fn(cmd, &enableReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*catalog.ListSystemSchemasRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq catalog.ListSystemSchemasRequest

	cmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of schemas to return.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Opaque pagination token to go to next page based on previous query.`)

	cmd.Use = "list METASTORE_ID"
	cmd.Short = `List system schemas.`
	cmd.Long = `List system schemas.
  
  Gets an array of system schemas for a metastore. The caller must be an account
  admin or a metastore admin.

  Arguments:
    METASTORE_ID: The ID for the metastore in which the system schema resides.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listReq.MetastoreId = args[0]

		response := w.SystemSchemas.List(ctx, listReq)
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

// end service SystemSchemas
