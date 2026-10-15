// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	database "github.com/databricks/cli/libs/sdktypes/database/v1"
	sdkdatabase "github.com/databricks/sdk-go/database/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: `*Public Preview* Database Instances provide access to a database via REST API or direct SQL.`,
		Long: `This command is in Public Preview and may change without notice.

Database Instances provide access to a database via REST API or direct SQL.`,
		GroupID: "database",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newCreateDatabaseCatalog())
	cmd.AddCommand(newCreateDatabaseInstance())
	cmd.AddCommand(newCreateDatabaseInstanceRole())
	cmd.AddCommand(newCreateDatabaseTable())
	cmd.AddCommand(newCreateSyncedDatabaseTable())
	cmd.AddCommand(newDeleteDatabaseCatalog())
	cmd.AddCommand(newDeleteDatabaseInstance())
	cmd.AddCommand(newDeleteDatabaseInstanceRole())
	cmd.AddCommand(newDeleteDatabaseTable())
	cmd.AddCommand(newDeleteSyncedDatabaseTable())
	cmd.AddCommand(newFindDatabaseInstanceByUid())
	cmd.AddCommand(newGenerateDatabaseCredential())
	cmd.AddCommand(newGetDatabaseCatalog())
	cmd.AddCommand(newGetDatabaseInstance())
	cmd.AddCommand(newGetDatabaseInstanceRole())
	cmd.AddCommand(newGetDatabaseTable())
	cmd.AddCommand(newGetSyncedDatabaseTable())
	cmd.AddCommand(newListDatabaseCatalogs())
	cmd.AddCommand(newListDatabaseInstanceRoles())
	cmd.AddCommand(newListDatabaseInstances())
	cmd.AddCommand(newListSyncedDatabaseTables())
	cmd.AddCommand(newUpdateDatabaseCatalog())
	cmd.AddCommand(newUpdateDatabaseInstance())
	cmd.AddCommand(newUpdateSyncedDatabaseTable())

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
	createDatabaseCatalogReq.Catalog = &database.DatabaseCatalog{}

	var createDatabaseCatalogJson flags.JsonFlag

	cmd.Flags().Var(&createDatabaseCatalogJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createDatabaseCatalogReq.Catalog.CreateDatabaseIfNotExists, "create-database-if-not-exists", ``)

	cmd.Use = "create-database-catalog NAME DATABASE_INSTANCE_NAME DATABASE_NAME"
	cmd.Short = `*Public Preview* Create a Database Catalog.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a Database Catalog.

  Arguments:
    NAME: The name of the catalog in UC.
    DATABASE_INSTANCE_NAME: The name of the DatabaseInstance housing the database.
    DATABASE_NAME: The name of the database (in an instance) associated with the catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'database_instance_name', 'database_name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createDatabaseCatalogJson.Unmarshal(&createDatabaseCatalogReq.Catalog)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createDatabaseCatalogReq.Catalog.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createDatabaseCatalogReq.Catalog.DatabaseInstanceName = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createDatabaseCatalogReq.Catalog.DatabaseName = new(args[2])
		}

		sdkReq, err := database.CreateDatabaseCatalogRequestToSDK(&createDatabaseCatalogReq)
		if err != nil {
			return err
		}
		response, err := w.Database.CreateDatabaseCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseCatalogFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	createDatabaseInstanceReq.DatabaseInstance = &database.DatabaseInstance{}

	var createDatabaseInstanceJson flags.JsonFlag

	var createDatabaseInstanceSkipWait bool
	var createDatabaseInstanceTimeout time.Duration

	cmd.Flags().BoolVar(&createDatabaseInstanceSkipWait, "no-wait", createDatabaseInstanceSkipWait, `do not wait to reach AVAILABLE state`)
	cmd.Flags().DurationVar(&createDatabaseInstanceTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach AVAILABLE state`)

	cmd.Flags().Var(&createDatabaseInstanceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createDatabaseInstanceReq.DatabaseInstance.Capacity, "capacity", `The sku of the instance.`)
	// TODO: array: child_instance_refs
	// TODO: array: custom_tags
	// TODO: array: effective_custom_tags
	flags.BoolPtrVar(cmd.Flags(), &createDatabaseInstanceReq.DatabaseInstance.EnablePgNativeLogin, "enable-pg-native-login", `Whether to enable PG native password login on the instance.`)
	flags.BoolPtrVar(cmd.Flags(), &createDatabaseInstanceReq.DatabaseInstance.EnableReadableSecondaries, "enable-readable-secondaries", `Whether to enable secondaries to serve read-only traffic.`)
	flags.IntPtrVar(cmd.Flags(), &createDatabaseInstanceReq.DatabaseInstance.NodeCount, "node-count", `The number of nodes in the instance, composed of 1 primary and 0 or more secondaries.`)
	// TODO: complex arg: parent_instance_ref
	flags.IntPtrVar(cmd.Flags(), &createDatabaseInstanceReq.DatabaseInstance.RetentionWindowInDays, "retention-window-in-days", `The retention window for the instance.`)
	flags.BoolPtrVar(cmd.Flags(), &createDatabaseInstanceReq.DatabaseInstance.Stopped, "stopped", `Whether to stop the instance.`)
	flags.StringPtrVar(cmd.Flags(), &createDatabaseInstanceReq.DatabaseInstance.UsagePolicyId, "usage-policy-id", `The desired usage policy to associate with the instance.`)

	cmd.Use = "create-database-instance NAME"
	cmd.Short = `*Public Preview* Create a Database Instance.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a Database Instance.

  Arguments:
    NAME: The name of the instance. This is the unique identifier for the instance.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createDatabaseInstanceJson.Unmarshal(&createDatabaseInstanceReq.DatabaseInstance)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createDatabaseInstanceReq.DatabaseInstance.Name = new(args[0])
		}

		sdkReq, err := database.CreateDatabaseInstanceRequestToSDK(&createDatabaseInstanceReq)
		if err != nil {
			return err
		}
		wait, err := w.Database.CreateDatabaseInstance(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if createDatabaseInstanceSkipWait {
			// Render the current state, as of right after the call.
			current, err := w.Database.GetDatabaseInstance(ctx, sdkdatabase.GetDatabaseInstanceRequest{Name: new(wait.GetName())})
			if err != nil {
				return err
			}
			out, err := database.DatabaseInstanceFromSDK(current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create-database-instance to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createDatabaseInstanceTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := database.DatabaseInstanceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start create-database-instance-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseInstanceRoleOverrides []func(
	*cobra.Command,
	*database.CreateDatabaseInstanceRoleRequest,
)

func newCreateDatabaseInstanceRole() *cobra.Command {
	cmd := &cobra.Command{}
	var createDatabaseInstanceRoleReq database.CreateDatabaseInstanceRoleRequest
	createDatabaseInstanceRoleReq.DatabaseInstanceRole = &database.DatabaseInstanceRole{}

	var createDatabaseInstanceRoleJson flags.JsonFlag

	cmd.Flags().Var(&createDatabaseInstanceRoleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createDatabaseInstanceRoleReq.DatabaseInstanceName, "database-instance-name", ``)
	// TODO: complex arg: attributes
	// TODO: complex arg: effective_attributes
	flags.EnumVar(cmd.Flags(), &createDatabaseInstanceRoleReq.DatabaseInstanceRole.IdentityType, "identity-type", `The type of the role. Supported values: [GROUP, PG_ONLY, SERVICE_PRINCIPAL, USER]`, "DatabaseInstanceRoleIdentityType", "GROUP", "PG_ONLY", "SERVICE_PRINCIPAL", "USER")
	flags.StringPtrVar(cmd.Flags(), &createDatabaseInstanceRoleReq.InstanceName, "instance-name", ``)
	flags.EnumVar(cmd.Flags(), &createDatabaseInstanceRoleReq.DatabaseInstanceRole.MembershipRole, "membership-role", `An enum value for a standard role that this role is a member of. Supported values: [DATABRICKS_SUPERUSER]`, "DatabaseInstanceRoleMembershipRole", "DATABRICKS_SUPERUSER")

	cmd.Use = "create-database-instance-role INSTANCE_NAME NAME"
	cmd.Short = `Create a role for a Database Instance.`
	cmd.Long = `Create a role for a Database Instance.

  Arguments:
    INSTANCE_NAME:
    NAME: The name of the role. This is the unique identifier for the role in an
      instance.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only INSTANCE_NAME as positional arguments. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createDatabaseInstanceRoleJson.Unmarshal(&createDatabaseInstanceRoleReq.DatabaseInstanceRole)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createDatabaseInstanceRoleReq.InstanceName = new(args[0])
		if !cmd.Flags().Changed("json") {
			createDatabaseInstanceRoleReq.DatabaseInstanceRole.Name = new(args[1])
		}

		sdkReq, err := database.CreateDatabaseInstanceRoleRequestToSDK(&createDatabaseInstanceRoleReq)
		if err != nil {
			return err
		}
		response, err := w.Database.CreateDatabaseInstanceRole(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseInstanceRoleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseInstanceRoleOverrides {
		fn(cmd, &createDatabaseInstanceRoleReq)
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
	createDatabaseTableReq.Table = &database.DatabaseTable{}

	var createDatabaseTableJson flags.JsonFlag

	cmd.Flags().Var(&createDatabaseTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createDatabaseTableReq.Table.DatabaseInstanceName, "database-instance-name", `Name of the target database instance.`)
	flags.StringPtrVar(cmd.Flags(), &createDatabaseTableReq.Table.LogicalDatabaseName, "logical-database-name", `Target Postgres database object (logical database) name for this table.`)

	cmd.Use = "create-database-table NAME"
	cmd.Short = `*Public Preview* Create a Database Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a Database Table.

  Create a Database Table. Useful for registering pre-existing PG tables in UC.
  See CreateSyncedDatabaseTable for creating synced tables in PG from a source
  table in UC.

  Arguments:
    NAME: Full three-part (catalog, schema, table) name of the table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createDatabaseTableJson.Unmarshal(&createDatabaseTableReq.Table)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createDatabaseTableReq.Table.Name = new(args[0])
		}

		sdkReq, err := database.CreateDatabaseTableRequestToSDK(&createDatabaseTableReq)
		if err != nil {
			return err
		}
		response, err := w.Database.CreateDatabaseTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	createSyncedDatabaseTableReq.SyncedTable = &database.SyncedDatabaseTable{}

	var createSyncedDatabaseTableJson flags.JsonFlag

	cmd.Flags().Var(&createSyncedDatabaseTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: data_synchronization_status
	flags.StringPtrVar(cmd.Flags(), &createSyncedDatabaseTableReq.SyncedTable.DatabaseInstanceName, "database-instance-name", `Name of the target database instance.`)
	flags.StringPtrVar(cmd.Flags(), &createSyncedDatabaseTableReq.SyncedTable.LogicalDatabaseName, "logical-database-name", `Target Postgres database object (logical database) name for this table.`)
	// TODO: complex arg: spec

	cmd.Use = "create-synced-database-table NAME"
	cmd.Short = `*Public Preview* Create a Synced Database Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create a Synced Database Table.

  Arguments:
    NAME: Full three-part (catalog, schema, table) name of the table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createSyncedDatabaseTableJson.Unmarshal(&createSyncedDatabaseTableReq.SyncedTable)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createSyncedDatabaseTableReq.SyncedTable.Name = new(args[0])
		}

		sdkReq, err := database.CreateSyncedDatabaseTableRequestToSDK(&createSyncedDatabaseTableReq)
		if err != nil {
			return err
		}
		response, err := w.Database.CreateSyncedDatabaseTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.SyncedDatabaseTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkdatabase.DeleteDatabaseCatalogRequest,
)

func newDeleteDatabaseCatalog() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDatabaseCatalogReq sdkdatabase.DeleteDatabaseCatalogRequest

	cmd.Use = "delete-database-catalog NAME"
	cmd.Short = `*Public Preview* Delete a Database Catalog.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a Database Catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteDatabaseCatalogReq.Name = new(args[0])

		sdkReq := &deleteDatabaseCatalogReq
		err = w.Database.DeleteDatabaseCatalog(ctx, *sdkReq)
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

	flags.BoolPtrVar(cmd.Flags(), &deleteDatabaseInstanceReq.Force, "force", `By default, an instance cannot be deleted if it has descendant instances created via PITR.`)
	flags.BoolPtrVar(cmd.Flags(), &deleteDatabaseInstanceReq.Purge, "purge", `Deprecated.`)

	cmd.Use = "delete-database-instance NAME"
	cmd.Short = `*Public Preview* Delete a Database Instance.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a Database Instance.

  Arguments:
    NAME: Name of the instance to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteDatabaseInstanceReq.Name = new(args[0])

		sdkReq, err := database.DeleteDatabaseInstanceRequestToSDK(&deleteDatabaseInstanceReq)
		if err != nil {
			return err
		}
		err = w.Database.DeleteDatabaseInstance(ctx, *sdkReq)
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

// start delete-database-instance-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseInstanceRoleOverrides []func(
	*cobra.Command,
	*database.DeleteDatabaseInstanceRoleRequest,
)

func newDeleteDatabaseInstanceRole() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDatabaseInstanceRoleReq database.DeleteDatabaseInstanceRoleRequest

	flags.BoolPtrVar(cmd.Flags(), &deleteDatabaseInstanceRoleReq.AllowMissing, "allow-missing", `This is the AIP standard name for the equivalent of Postgres' IF EXISTS option.`)
	flags.StringPtrVar(cmd.Flags(), &deleteDatabaseInstanceRoleReq.ReassignOwnedTo, "reassign-owned-to", ``)

	cmd.Use = "delete-database-instance-role INSTANCE_NAME NAME"
	cmd.Short = `Delete a role for a Database Instance.`
	cmd.Long = `Delete a role for a Database Instance.

  Deletes a role for a Database Instance.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteDatabaseInstanceRoleReq.InstanceName = new(args[0])
		deleteDatabaseInstanceRoleReq.Name = new(args[1])

		sdkReq, err := database.DeleteDatabaseInstanceRoleRequestToSDK(&deleteDatabaseInstanceRoleReq)
		if err != nil {
			return err
		}
		err = w.Database.DeleteDatabaseInstanceRole(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseInstanceRoleOverrides {
		fn(cmd, &deleteDatabaseInstanceRoleReq)
	}

	return cmd
}

// start delete-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseTableOverrides []func(
	*cobra.Command,
	*sdkdatabase.DeleteDatabaseTableRequest,
)

func newDeleteDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDatabaseTableReq sdkdatabase.DeleteDatabaseTableRequest

	cmd.Use = "delete-database-table NAME"
	cmd.Short = `*Public Preview* Delete a Database Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a Database Table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteDatabaseTableReq.Name = new(args[0])

		sdkReq := &deleteDatabaseTableReq
		err = w.Database.DeleteDatabaseTable(ctx, *sdkReq)
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

	flags.BoolPtrVar(cmd.Flags(), &deleteSyncedDatabaseTableReq.PurgeData, "purge-data", `Optional.`)

	cmd.Use = "delete-synced-database-table NAME"
	cmd.Short = `*Public Preview* Delete a Synced Database Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete a Synced Database Table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteSyncedDatabaseTableReq.Name = new(args[0])

		sdkReq, err := database.DeleteSyncedDatabaseTableRequestToSDK(&deleteSyncedDatabaseTableReq)
		if err != nil {
			return err
		}
		err = w.Database.DeleteSyncedDatabaseTable(ctx, *sdkReq)
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

	flags.StringPtrVar(cmd.Flags(), &findDatabaseInstanceByUidReq.Uid, "uid", `UID of the cluster to get.`)

	cmd.Use = "find-database-instance-by-uid"
	cmd.Short = `*Public Preview* Find a Database Instance by uid.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Find a Database Instance by uid.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := database.FindDatabaseInstanceByUidRequestToSDK(&findDatabaseInstanceByUidReq)
		if err != nil {
			return err
		}
		response, err := w.Database.FindDatabaseInstanceByUid(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseInstanceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	cmd.Flags().Var(&generateDatabaseCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: claims
	// TODO: array: instance_names
	flags.StringPtrVar(cmd.Flags(), &generateDatabaseCredentialReq.RequestId, "request-id", ``)

	cmd.Use = "generate-database-credential"
	cmd.Short = `*Public Preview* Generates a credential that can be used to access database instances.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Generates a credential that can be used to access database instances.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := generateDatabaseCredentialJson.Unmarshal(&generateDatabaseCredentialReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}

		sdkReq, err := database.GenerateDatabaseCredentialRequestToSDK(&generateDatabaseCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.Database.GenerateDatabaseCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseCredentialFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkdatabase.GetDatabaseCatalogRequest,
)

func newGetDatabaseCatalog() *cobra.Command {
	cmd := &cobra.Command{}
	var getDatabaseCatalogReq sdkdatabase.GetDatabaseCatalogRequest

	cmd.Use = "get-database-catalog NAME"
	cmd.Short = `*Public Preview* Get a Database Catalog.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a Database Catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getDatabaseCatalogReq.Name = new(args[0])

		sdkReq := &getDatabaseCatalogReq
		response, err := w.Database.GetDatabaseCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseCatalogFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkdatabase.GetDatabaseInstanceRequest,
)

func newGetDatabaseInstance() *cobra.Command {
	cmd := &cobra.Command{}
	var getDatabaseInstanceReq sdkdatabase.GetDatabaseInstanceRequest

	cmd.Use = "get-database-instance NAME"
	cmd.Short = `*Public Preview* Get a Database Instance.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a Database Instance.

  Arguments:
    NAME: Name of the cluster to get.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getDatabaseInstanceReq.Name = new(args[0])

		sdkReq := &getDatabaseInstanceReq
		response, err := w.Database.GetDatabaseInstance(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseInstanceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start get-database-instance-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseInstanceRoleOverrides []func(
	*cobra.Command,
	*sdkdatabase.GetDatabaseInstanceRoleRequest,
)

func newGetDatabaseInstanceRole() *cobra.Command {
	cmd := &cobra.Command{}
	var getDatabaseInstanceRoleReq sdkdatabase.GetDatabaseInstanceRoleRequest

	cmd.Use = "get-database-instance-role INSTANCE_NAME NAME"
	cmd.Short = `Get a role for a Database Instance.`
	cmd.Long = `Get a role for a Database Instance.

  Gets a role for a Database Instance.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getDatabaseInstanceRoleReq.InstanceName = new(args[0])
		getDatabaseInstanceRoleReq.Name = new(args[1])

		sdkReq := &getDatabaseInstanceRoleReq
		response, err := w.Database.GetDatabaseInstanceRole(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseInstanceRoleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseInstanceRoleOverrides {
		fn(cmd, &getDatabaseInstanceRoleReq)
	}

	return cmd
}

// start get-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseTableOverrides []func(
	*cobra.Command,
	*sdkdatabase.GetDatabaseTableRequest,
)

func newGetDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}
	var getDatabaseTableReq sdkdatabase.GetDatabaseTableRequest

	cmd.Use = "get-database-table NAME"
	cmd.Short = `*Public Preview* Get a Database Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a Database Table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getDatabaseTableReq.Name = new(args[0])

		sdkReq := &getDatabaseTableReq
		response, err := w.Database.GetDatabaseTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkdatabase.GetSyncedDatabaseTableRequest,
)

func newGetSyncedDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}
	var getSyncedDatabaseTableReq sdkdatabase.GetSyncedDatabaseTableRequest

	cmd.Use = "get-synced-database-table NAME"
	cmd.Short = `*Public Preview* Get a Synced Database Table.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get a Synced Database Table.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getSyncedDatabaseTableReq.Name = new(args[0])

		sdkReq := &getSyncedDatabaseTableReq
		response, err := w.Database.GetSyncedDatabaseTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.SyncedDatabaseTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start list-database-catalogs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatabaseCatalogsOverrides []func(
	*cobra.Command,
	*database.ListDatabaseCatalogsRequest,
)

func newListDatabaseCatalogs() *cobra.Command {
	cmd := &cobra.Command{}
	var listDatabaseCatalogsReq database.ListDatabaseCatalogsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDatabaseCatalogsLimit int

	flags.IntPtrVar(cmd.Flags(), &listDatabaseCatalogsReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDatabaseCatalogsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listDatabaseCatalogsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-database-catalogs INSTANCE_NAME"
	cmd.Short = `List all Database Catalogs in a Database Instance.`
	cmd.Long = `List all Database Catalogs in a Database Instance.

  This API is currently unimplemented, but exposed for Terraform support.

  Arguments:
    INSTANCE_NAME: Name of the instance to get database catalogs for.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listDatabaseCatalogsReq.InstanceName = new(args[0])

		sdkReq, err := database.ListDatabaseCatalogsRequestToSDK(&listDatabaseCatalogsReq)
		if err != nil {
			return err
		}
		if listDatabaseCatalogsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDatabaseCatalogsLimit)
		}
		if listDatabaseCatalogsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDatabaseCatalogsLimit)
		}
		response := w.Database.ListDatabaseCatalogsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, database.DatabaseCatalogFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatabaseCatalogsOverrides {
		fn(cmd, &listDatabaseCatalogsReq)
	}

	return cmd
}

// start list-database-instance-roles command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatabaseInstanceRolesOverrides []func(
	*cobra.Command,
	*database.ListDatabaseInstanceRolesRequest,
)

func newListDatabaseInstanceRoles() *cobra.Command {
	cmd := &cobra.Command{}
	var listDatabaseInstanceRolesReq database.ListDatabaseInstanceRolesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDatabaseInstanceRolesLimit int

	flags.IntPtrVar(cmd.Flags(), &listDatabaseInstanceRolesReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDatabaseInstanceRolesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listDatabaseInstanceRolesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-database-instance-roles INSTANCE_NAME"
	cmd.Short = `List roles for a Database Instance.`
	cmd.Long = `List roles for a Database Instance.

  START OF PG ROLE APIs Section These APIs are marked a PUBLIC with stage <
  PUBLIC_PREVIEW. With more recent Lakebase V2 plans, we don't plan to ever
  advance these to PUBLIC_PREVIEW. These APIs will remain effectively
  undocumented/UI-only and we'll aim for a new public roles API as part of V2
  PuPr.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listDatabaseInstanceRolesReq.InstanceName = new(args[0])

		sdkReq, err := database.ListDatabaseInstanceRolesRequestToSDK(&listDatabaseInstanceRolesReq)
		if err != nil {
			return err
		}
		if listDatabaseInstanceRolesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDatabaseInstanceRolesLimit)
		}
		if listDatabaseInstanceRolesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDatabaseInstanceRolesLimit)
		}
		response := w.Database.ListDatabaseInstanceRolesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, database.DatabaseInstanceRoleFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatabaseInstanceRolesOverrides {
		fn(cmd, &listDatabaseInstanceRolesReq)
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDatabaseInstancesLimit int

	flags.IntPtrVar(cmd.Flags(), &listDatabaseInstancesReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDatabaseInstancesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listDatabaseInstancesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-database-instances"
	cmd.Short = `*Public Preview* List Database Instances.`
	cmd.Long = `This command is in Public Preview and may change without notice.

List Database Instances.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := database.ListDatabaseInstancesRequestToSDK(&listDatabaseInstancesReq)
		if err != nil {
			return err
		}
		if listDatabaseInstancesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDatabaseInstancesLimit)
		}
		if listDatabaseInstancesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDatabaseInstancesLimit)
		}
		response := w.Database.ListDatabaseInstancesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, database.DatabaseInstanceFromSDK))
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

// start list-synced-database-tables command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSyncedDatabaseTablesOverrides []func(
	*cobra.Command,
	*database.ListSyncedDatabaseTablesRequest,
)

func newListSyncedDatabaseTables() *cobra.Command {
	cmd := &cobra.Command{}
	var listSyncedDatabaseTablesReq database.ListSyncedDatabaseTablesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSyncedDatabaseTablesLimit int

	flags.IntPtrVar(cmd.Flags(), &listSyncedDatabaseTablesReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSyncedDatabaseTablesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSyncedDatabaseTablesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-synced-database-tables INSTANCE_NAME"
	cmd.Short = `List all synced database tables in a Database Instance.`
	cmd.Long = `List all synced database tables in a Database Instance.

  List synced database tables in a Database Instance.

  Arguments:
    INSTANCE_NAME: Name of the instance to get synced tables for.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listSyncedDatabaseTablesReq.InstanceName = new(args[0])

		sdkReq, err := database.ListSyncedDatabaseTablesRequestToSDK(&listSyncedDatabaseTablesReq)
		if err != nil {
			return err
		}
		if listSyncedDatabaseTablesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSyncedDatabaseTablesLimit)
		}
		if listSyncedDatabaseTablesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSyncedDatabaseTablesLimit)
		}
		response := w.Database.ListSyncedDatabaseTablesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, database.SyncedDatabaseTableFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSyncedDatabaseTablesOverrides {
		fn(cmd, &listSyncedDatabaseTablesReq)
	}

	return cmd
}

// start update-database-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatabaseCatalogOverrides []func(
	*cobra.Command,
	*database.UpdateDatabaseCatalogRequest,
)

func newUpdateDatabaseCatalog() *cobra.Command {
	cmd := &cobra.Command{}
	var updateDatabaseCatalogReq database.UpdateDatabaseCatalogRequest
	updateDatabaseCatalogReq.DatabaseCatalog = &database.DatabaseCatalog{}

	var updateDatabaseCatalogJson flags.JsonFlag

	cmd.Flags().Var(&updateDatabaseCatalogJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateDatabaseCatalogReq.DatabaseCatalog.CreateDatabaseIfNotExists, "create-database-if-not-exists", ``)

	cmd.Use = "update-database-catalog NAME UPDATE_MASK DATABASE_INSTANCE_NAME DATABASE_NAME"
	cmd.Short = `Update a Database Catalog.`
	cmd.Long = `Update a Database Catalog.

  This API is currently unimplemented, but exposed for Terraform support.

  Arguments:
    NAME: The name of the catalog in UC.
    UPDATE_MASK: The list of fields to update. Setting this field is not yet supported.
    DATABASE_INSTANCE_NAME: The name of the DatabaseInstance housing the database.
    DATABASE_NAME: The name of the database (in an instance) associated with the catalog.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'name', 'database_instance_name', 'database_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateDatabaseCatalogJson.Unmarshal(&updateDatabaseCatalogReq.DatabaseCatalog)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateDatabaseCatalogReq.DatabaseCatalog.Name = new(args[0])
		updateDatabaseCatalogReq.UpdateMask = new(args[1])
		if !cmd.Flags().Changed("json") {
			updateDatabaseCatalogReq.DatabaseCatalog.DatabaseInstanceName = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			updateDatabaseCatalogReq.DatabaseCatalog.DatabaseName = new(args[3])
		}

		sdkReq, err := database.UpdateDatabaseCatalogRequestToSDK(&updateDatabaseCatalogReq)
		if err != nil {
			return err
		}
		response, err := w.Database.UpdateDatabaseCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseCatalogFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatabaseCatalogOverrides {
		fn(cmd, &updateDatabaseCatalogReq)
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
	updateDatabaseInstanceReq.DatabaseInstance = &database.DatabaseInstance{}

	var updateDatabaseInstanceJson flags.JsonFlag

	cmd.Flags().Var(&updateDatabaseInstanceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateDatabaseInstanceReq.DatabaseInstance.Capacity, "capacity", `The sku of the instance.`)
	// TODO: array: child_instance_refs
	// TODO: array: custom_tags
	// TODO: array: effective_custom_tags
	flags.BoolPtrVar(cmd.Flags(), &updateDatabaseInstanceReq.DatabaseInstance.EnablePgNativeLogin, "enable-pg-native-login", `Whether to enable PG native password login on the instance.`)
	flags.BoolPtrVar(cmd.Flags(), &updateDatabaseInstanceReq.DatabaseInstance.EnableReadableSecondaries, "enable-readable-secondaries", `Whether to enable secondaries to serve read-only traffic.`)
	flags.IntPtrVar(cmd.Flags(), &updateDatabaseInstanceReq.DatabaseInstance.NodeCount, "node-count", `The number of nodes in the instance, composed of 1 primary and 0 or more secondaries.`)
	// TODO: complex arg: parent_instance_ref
	flags.IntPtrVar(cmd.Flags(), &updateDatabaseInstanceReq.DatabaseInstance.RetentionWindowInDays, "retention-window-in-days", `The retention window for the instance.`)
	flags.BoolPtrVar(cmd.Flags(), &updateDatabaseInstanceReq.DatabaseInstance.Stopped, "stopped", `Whether to stop the instance.`)
	flags.StringPtrVar(cmd.Flags(), &updateDatabaseInstanceReq.DatabaseInstance.UsagePolicyId, "usage-policy-id", `The desired usage policy to associate with the instance.`)

	cmd.Use = "update-database-instance NAME UPDATE_MASK"
	cmd.Short = `*Public Preview* Update a Database Instance.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update a Database Instance.

  Arguments:
    NAME: The name of the instance. This is the unique identifier for the instance.
    UPDATE_MASK: The list of fields to update. If unspecified, all fields will be updated
      when possible. To wipe out custom_tags, specify custom_tags in the
      update_mask with an empty custom_tags map.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateDatabaseInstanceJson.Unmarshal(&updateDatabaseInstanceReq.DatabaseInstance)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateDatabaseInstanceReq.DatabaseInstance.Name = new(args[0])
		updateDatabaseInstanceReq.UpdateMask = new(args[1])

		sdkReq, err := database.UpdateDatabaseInstanceRequestToSDK(&updateDatabaseInstanceReq)
		if err != nil {
			return err
		}
		response, err := w.Database.UpdateDatabaseInstance(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.DatabaseInstanceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start update-synced-database-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSyncedDatabaseTableOverrides []func(
	*cobra.Command,
	*database.UpdateSyncedDatabaseTableRequest,
)

func newUpdateSyncedDatabaseTable() *cobra.Command {
	cmd := &cobra.Command{}
	var updateSyncedDatabaseTableReq database.UpdateSyncedDatabaseTableRequest
	updateSyncedDatabaseTableReq.SyncedTable = &database.SyncedDatabaseTable{}

	var updateSyncedDatabaseTableJson flags.JsonFlag

	cmd.Flags().Var(&updateSyncedDatabaseTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: data_synchronization_status
	flags.StringPtrVar(cmd.Flags(), &updateSyncedDatabaseTableReq.SyncedTable.DatabaseInstanceName, "database-instance-name", `Name of the target database instance.`)
	flags.StringPtrVar(cmd.Flags(), &updateSyncedDatabaseTableReq.SyncedTable.LogicalDatabaseName, "logical-database-name", `Target Postgres database object (logical database) name for this table.`)
	// TODO: complex arg: spec

	cmd.Use = "update-synced-database-table NAME UPDATE_MASK"
	cmd.Short = `Update a Synced Database Table.`
	cmd.Long = `Update a Synced Database Table.

  This API is currently unimplemented, but exposed for Terraform support.

  Arguments:
    NAME: Full three-part (catalog, schema, table) name of the table.
    UPDATE_MASK: The list of fields to update. Setting this field is not yet supported.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateSyncedDatabaseTableJson.Unmarshal(&updateSyncedDatabaseTableReq.SyncedTable)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateSyncedDatabaseTableReq.SyncedTable.Name = new(args[0])
		updateSyncedDatabaseTableReq.UpdateMask = new(args[1])

		sdkReq, err := database.UpdateSyncedDatabaseTableRequestToSDK(&updateSyncedDatabaseTableReq)
		if err != nil {
			return err
		}
		response, err := w.Database.UpdateSyncedDatabaseTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := database.SyncedDatabaseTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSyncedDatabaseTableOverrides {
		fn(cmd, &updateSyncedDatabaseTableReq)
	}

	return cmd
}

// end service Database
