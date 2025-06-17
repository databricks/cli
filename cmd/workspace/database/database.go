// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package database

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "database",
		Short:   `Database Instances provide access to a database via REST API or direct SQL.`,
		Long:    `Database Instances provide access to a database via REST API or direct SQL.`,
		GroupID: "database",
		Annotations: map[string]string{
			"package": "database",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateDatabaseCatalog())
	cmd.AddCommand(newCreateDatabaseInstance())
	cmd.AddCommand(newCreateDatabaseTable())
	cmd.AddCommand(newCreateSyncedDatabaseTable())
	cmd.AddCommand(newDeleteDatabaseCatalog())
	cmd.AddCommand(newDeleteDatabaseInstance())
	cmd.AddCommand(newDeleteDatabaseTable())
	cmd.AddCommand(newDeleteSyncedDatabaseTable())
	cmd.AddCommand(newFindDatabaseInstanceByUid())
	cmd.AddCommand(newGenerateDatabaseCredential())
	cmd.AddCommand(newGetDatabaseCatalog())
	cmd.AddCommand(newGetDatabaseInstance())
	cmd.AddCommand(newGetDatabaseTable())
	cmd.AddCommand(newGetSyncedDatabaseTable())
	cmd.AddCommand(newListDatabaseInstances())
	cmd.AddCommand(newUpdateDatabaseInstance())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-database-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseCatalogOverrides []func(
	*cobra.Command,
	*database.CreateDatabaseCatalogRequest,
)

func newCreateDatabaseCatalog() *cobra.Command {
	cmd := &cobra.Command{}

	var createDatabaseCatalogReq database.CreateDatabaseCatalogRequest
	createDatabaseCatalogReq.Catalog = database.DatabaseCatalog{}
	var createDatabaseCatalogJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createDatabaseCatalogJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createDatabaseCatalogReq.Catalog.CreateDatabaseIfNotExists, "create-database-if-not-exists", createDatabaseCatalogReq.Catalog.CreateDatabaseIfNotExists, ``)

	cmd.Use = "create-database-catalog NAME DATABASE_INSTANCE_NAME DATABASE_NAME"
	cmd.Short = `Create a Database Catalog.`
	cmd.Long = `Create a Database Catalog.

  Arguments:
    NAME: The name of the catalog in UC.
    DATABASE_INSTANCE_NAME: The name of the DatabaseInstance housing the database.
    DATABASE_NAME: The name of the database (in a instance) associated with the catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'database_instance_name', 'database_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDatabaseCatalogJson.Unmarshal(&createDatabaseCatalogReq.Catalog)
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
		if !cmd.Flags().Changed("json") {
			createDatabaseCatalogReq.Catalog.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createDatabaseCatalogReq.Catalog.DatabaseInstanceName = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createDatabaseCatalogReq.Catalog.DatabaseName = args[2]
		}

		response, err := w.Database.CreateDatabaseCatalog(ctx, createDatabaseCatalogReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseCatalogOverrides {
		fn(cmd, &createDatabaseCatalogReq)
	}

	return cmd
}

// start create-database-instance command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseInstanceOverrides []func(
	*cobra.Command,
	*database.CreateDatabaseInstanceRequest,
)

func newCreateDatabaseInstance() *cobra.Command {
	cmd := &cobra.Command{}

	var createDatabaseInstanceReq database.CreateDatabaseInstanceRequest
	createDatabaseInstanceReq.DatabaseInstance = database.DatabaseInstance{}
	var createDatabaseInstanceJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createDatabaseInstanceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDatabaseInstanceReq.DatabaseInstance.Capacity, "capacity", createDatabaseInstanceReq.DatabaseInstance.Capacity, `The sku of the instance.`)
	cmd.Flags().BoolVar(&createDatabaseInstanceReq.DatabaseInstance.Stopped, "stopped", createDatabaseInstanceReq.DatabaseInstance.Stopped, `Whether the instance is stopped.`)

	cmd.Use = "create-database-instance NAME"
	cmd.Short = `Create a Database Instance.`
	cmd.Long = `Create a Database Instance.

  Arguments:
    NAME: The name of the instance. This is the unique identifier for the instance.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDatabaseInstanceJson.Unmarshal(&createDatabaseInstanceReq.DatabaseInstance)
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
		if !cmd.Flags().Changed("json") {
			createDatabaseInstanceReq.DatabaseInstance.Name = args[0]
		}

		response, err := w.Database.CreateDatabaseInstance(ctx, createDatabaseInstanceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseInstanceOverrides {
		fn(cmd, &createDatabaseInstanceReq)
	}

	return cmd
}

// start create-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseTableOverrides []func(
	*cobra.Command,
	*database.CreateDatabaseTableRequest,
)

func newCreateDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}

	var createDatabaseTableReq database.CreateDatabaseTableRequest
	createDatabaseTableReq.Table = database.DatabaseTable{}
	var createDatabaseTableJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createDatabaseTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDatabaseTableReq.Table.DatabaseInstanceName, "database-instance-name", createDatabaseTableReq.Table.DatabaseInstanceName, `Name of the target database instance.`)
	cmd.Flags().StringVar(&createDatabaseTableReq.Table.LogicalDatabaseName, "logical-database-name", createDatabaseTableReq.Table.LogicalDatabaseName, `Target Postgres database object (logical database) name for this table.`)

	cmd.Use = "create-database-table NAME"
	cmd.Short = `Create a Database Table.`
	cmd.Long = `Create a Database Table.

  Arguments:
    NAME: Full three-part (catalog, schema, table) name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDatabaseTableJson.Unmarshal(&createDatabaseTableReq.Table)
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
		if !cmd.Flags().Changed("json") {
			createDatabaseTableReq.Table.Name = args[0]
		}

		response, err := w.Database.CreateDatabaseTable(ctx, createDatabaseTableReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseTableOverrides {
		fn(cmd, &createDatabaseTableReq)
	}

	return cmd
}

// start create-synced-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSyncedDatabaseTableOverrides []func(
	*cobra.Command,
	*database.CreateSyncedDatabaseTableRequest,
)

func newCreateSyncedDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}

	var createSyncedDatabaseTableReq database.CreateSyncedDatabaseTableRequest
	createSyncedDatabaseTableReq.SyncedTable = database.SyncedDatabaseTable{}
	var createSyncedDatabaseTableJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createSyncedDatabaseTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: data_synchronization_status
	cmd.Flags().StringVar(&createSyncedDatabaseTableReq.SyncedTable.DatabaseInstanceName, "database-instance-name", createSyncedDatabaseTableReq.SyncedTable.DatabaseInstanceName, `Name of the target database instance.`)
	cmd.Flags().StringVar(&createSyncedDatabaseTableReq.SyncedTable.LogicalDatabaseName, "logical-database-name", createSyncedDatabaseTableReq.SyncedTable.LogicalDatabaseName, `Target Postgres database object (logical database) name for this table.`)
	// TODO: complex arg: spec

	cmd.Use = "create-synced-database-table NAME"
	cmd.Short = `Create a Synced Database Table.`
	cmd.Long = `Create a Synced Database Table.

  Arguments:
    NAME: Full three-part (catalog, schema, table) name of the table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createSyncedDatabaseTableJson.Unmarshal(&createSyncedDatabaseTableReq.SyncedTable)
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
		if !cmd.Flags().Changed("json") {
			createSyncedDatabaseTableReq.SyncedTable.Name = args[0]
		}

		response, err := w.Database.CreateSyncedDatabaseTable(ctx, createSyncedDatabaseTableReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSyncedDatabaseTableOverrides {
		fn(cmd, &createSyncedDatabaseTableReq)
	}

	return cmd
}

// start delete-database-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseCatalogOverrides []func(
	*cobra.Command,
	*database.DeleteDatabaseCatalogRequest,
)

func newDeleteDatabaseCatalog() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDatabaseCatalogReq database.DeleteDatabaseCatalogRequest

	// TODO: short flags

	cmd.Use = "delete-database-catalog NAME"
	cmd.Short = `Delete a Database Catalog.`
	cmd.Long = `Delete a Database Catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDatabaseCatalogReq.Name = args[0]

		err = w.Database.DeleteDatabaseCatalog(ctx, deleteDatabaseCatalogReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseCatalogOverrides {
		fn(cmd, &deleteDatabaseCatalogReq)
	}

	return cmd
}

// start delete-database-instance command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseInstanceOverrides []func(
	*cobra.Command,
	*database.DeleteDatabaseInstanceRequest,
)

func newDeleteDatabaseInstance() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDatabaseInstanceReq database.DeleteDatabaseInstanceRequest

	// TODO: short flags

	cmd.Flags().BoolVar(&deleteDatabaseInstanceReq.Force, "force", deleteDatabaseInstanceReq.Force, `By default, a instance cannot be deleted if it has descendant instances created via PITR.`)
	cmd.Flags().BoolVar(&deleteDatabaseInstanceReq.Purge, "purge", deleteDatabaseInstanceReq.Purge, `If false, the database instance is soft deleted.`)

	cmd.Use = "delete-database-instance NAME"
	cmd.Short = `Delete a Database Instance.`
	cmd.Long = `Delete a Database Instance.

  Arguments:
    NAME: Name of the instance to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDatabaseInstanceReq.Name = args[0]

		err = w.Database.DeleteDatabaseInstance(ctx, deleteDatabaseInstanceReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseInstanceOverrides {
		fn(cmd, &deleteDatabaseInstanceReq)
	}

	return cmd
}

// start delete-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseTableOverrides []func(
	*cobra.Command,
	*database.DeleteDatabaseTableRequest,
)

func newDeleteDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDatabaseTableReq database.DeleteDatabaseTableRequest

	// TODO: short flags

	cmd.Use = "delete-database-table NAME"
	cmd.Short = `Delete a Database Table.`
	cmd.Long = `Delete a Database Table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDatabaseTableReq.Name = args[0]

		err = w.Database.DeleteDatabaseTable(ctx, deleteDatabaseTableReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseTableOverrides {
		fn(cmd, &deleteDatabaseTableReq)
	}

	return cmd
}

// start delete-synced-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSyncedDatabaseTableOverrides []func(
	*cobra.Command,
	*database.DeleteSyncedDatabaseTableRequest,
)

func newDeleteSyncedDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSyncedDatabaseTableReq database.DeleteSyncedDatabaseTableRequest

	// TODO: short flags

	cmd.Use = "delete-synced-database-table NAME"
	cmd.Short = `Delete a Synced Database Table.`
	cmd.Long = `Delete a Synced Database Table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteSyncedDatabaseTableReq.Name = args[0]

		err = w.Database.DeleteSyncedDatabaseTable(ctx, deleteSyncedDatabaseTableReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSyncedDatabaseTableOverrides {
		fn(cmd, &deleteSyncedDatabaseTableReq)
	}

	return cmd
}

// start find-database-instance-by-uid command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var findDatabaseInstanceByUidOverrides []func(
	*cobra.Command,
	*database.FindDatabaseInstanceByUidRequest,
)

func newFindDatabaseInstanceByUid() *cobra.Command {
	cmd := &cobra.Command{}

	var findDatabaseInstanceByUidReq database.FindDatabaseInstanceByUidRequest

	// TODO: short flags

	cmd.Flags().StringVar(&findDatabaseInstanceByUidReq.Uid, "uid", findDatabaseInstanceByUidReq.Uid, `UID of the cluster to get.`)

	cmd.Use = "find-database-instance-by-uid"
	cmd.Short = `Find a Database Instance by uid.`
	cmd.Long = `Find a Database Instance by uid.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.Database.FindDatabaseInstanceByUid(ctx, findDatabaseInstanceByUidReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range findDatabaseInstanceByUidOverrides {
		fn(cmd, &findDatabaseInstanceByUidReq)
	}

	return cmd
}

// start generate-database-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateDatabaseCredentialOverrides []func(
	*cobra.Command,
	*database.GenerateDatabaseCredentialRequest,
)

func newGenerateDatabaseCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var generateDatabaseCredentialReq database.GenerateDatabaseCredentialRequest
	var generateDatabaseCredentialJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&generateDatabaseCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: instance_names
	cmd.Flags().StringVar(&generateDatabaseCredentialReq.RequestId, "request-id", generateDatabaseCredentialReq.RequestId, ``)

	cmd.Use = "generate-database-credential"
	cmd.Short = `Generates a credential that can be used to access database instances.`
	cmd.Long = `Generates a credential that can be used to access database instances.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := generateDatabaseCredentialJson.Unmarshal(&generateDatabaseCredentialReq)
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

		response, err := w.Database.GenerateDatabaseCredential(ctx, generateDatabaseCredentialReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range generateDatabaseCredentialOverrides {
		fn(cmd, &generateDatabaseCredentialReq)
	}

	return cmd
}

// start get-database-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseCatalogOverrides []func(
	*cobra.Command,
	*database.GetDatabaseCatalogRequest,
)

func newGetDatabaseCatalog() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatabaseCatalogReq database.GetDatabaseCatalogRequest

	// TODO: short flags

	cmd.Use = "get-database-catalog NAME"
	cmd.Short = `Get a Database Catalog.`
	cmd.Long = `Get a Database Catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatabaseCatalogReq.Name = args[0]

		response, err := w.Database.GetDatabaseCatalog(ctx, getDatabaseCatalogReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseCatalogOverrides {
		fn(cmd, &getDatabaseCatalogReq)
	}

	return cmd
}

// start get-database-instance command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseInstanceOverrides []func(
	*cobra.Command,
	*database.GetDatabaseInstanceRequest,
)

func newGetDatabaseInstance() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatabaseInstanceReq database.GetDatabaseInstanceRequest

	// TODO: short flags

	cmd.Use = "get-database-instance NAME"
	cmd.Short = `Get a Database Instance.`
	cmd.Long = `Get a Database Instance.

  Arguments:
    NAME: Name of the cluster to get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatabaseInstanceReq.Name = args[0]

		response, err := w.Database.GetDatabaseInstance(ctx, getDatabaseInstanceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseInstanceOverrides {
		fn(cmd, &getDatabaseInstanceReq)
	}

	return cmd
}

// start get-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseTableOverrides []func(
	*cobra.Command,
	*database.GetDatabaseTableRequest,
)

func newGetDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatabaseTableReq database.GetDatabaseTableRequest

	// TODO: short flags

	cmd.Use = "get-database-table NAME"
	cmd.Short = `Get a Database Table.`
	cmd.Long = `Get a Database Table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatabaseTableReq.Name = args[0]

		response, err := w.Database.GetDatabaseTable(ctx, getDatabaseTableReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseTableOverrides {
		fn(cmd, &getDatabaseTableReq)
	}

	return cmd
}

// start get-synced-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSyncedDatabaseTableOverrides []func(
	*cobra.Command,
	*database.GetSyncedDatabaseTableRequest,
)

func newGetSyncedDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}

	var getSyncedDatabaseTableReq database.GetSyncedDatabaseTableRequest

	// TODO: short flags

	cmd.Use = "get-synced-database-table NAME"
	cmd.Short = `Get a Synced Database Table.`
	cmd.Long = `Get a Synced Database Table.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSyncedDatabaseTableReq.Name = args[0]

		response, err := w.Database.GetSyncedDatabaseTable(ctx, getSyncedDatabaseTableReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSyncedDatabaseTableOverrides {
		fn(cmd, &getSyncedDatabaseTableReq)
	}

	return cmd
}

// start list-database-instances command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatabaseInstancesOverrides []func(
	*cobra.Command,
	*database.ListDatabaseInstancesRequest,
)

func newListDatabaseInstances() *cobra.Command {
	cmd := &cobra.Command{}

	var listDatabaseInstancesReq database.ListDatabaseInstancesRequest

	// TODO: short flags

	cmd.Flags().IntVar(&listDatabaseInstancesReq.PageSize, "page-size", listDatabaseInstancesReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listDatabaseInstancesReq.PageToken, "page-token", listDatabaseInstancesReq.PageToken, `Pagination token to go to the next page of Database Instances.`)

	cmd.Use = "list-database-instances"
	cmd.Short = `List Database Instances.`
	cmd.Long = `List Database Instances.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Database.ListDatabaseInstances(ctx, listDatabaseInstancesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatabaseInstancesOverrides {
		fn(cmd, &listDatabaseInstancesReq)
	}

	return cmd
}

// start update-database-instance command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatabaseInstanceOverrides []func(
	*cobra.Command,
	*database.UpdateDatabaseInstanceRequest,
)

func newUpdateDatabaseInstance() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDatabaseInstanceReq database.UpdateDatabaseInstanceRequest
	updateDatabaseInstanceReq.DatabaseInstance = database.DatabaseInstance{}
	var updateDatabaseInstanceJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateDatabaseInstanceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateDatabaseInstanceReq.DatabaseInstance.Capacity, "capacity", updateDatabaseInstanceReq.DatabaseInstance.Capacity, `The sku of the instance.`)
	cmd.Flags().BoolVar(&updateDatabaseInstanceReq.DatabaseInstance.Stopped, "stopped", updateDatabaseInstanceReq.DatabaseInstance.Stopped, `Whether the instance is stopped.`)

	cmd.Use = "update-database-instance NAME"
	cmd.Short = `Update a Database Instance.`
	cmd.Long = `Update a Database Instance.

  Arguments:
    NAME: The name of the instance. This is the unique identifier for the instance.`

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
			diags := updateDatabaseInstanceJson.Unmarshal(&updateDatabaseInstanceReq.DatabaseInstance)
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
		updateDatabaseInstanceReq.Name = args[0]

		response, err := w.Database.UpdateDatabaseInstance(ctx, updateDatabaseInstanceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatabaseInstanceOverrides {
		fn(cmd, &updateDatabaseInstanceReq)
	}

	return cmd
}

// end service Database
