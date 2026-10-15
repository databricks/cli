// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package postgres

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	sdkcoretypes "github.com/databricks/sdk-go/core/types"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postgres",
		Short: `Use the Postgres API to create and manage Lakebase Autoscaling Postgres infrastructure, including projects, branches, compute endpoints, and roles.`,
		Long: `Use the Postgres API to create and manage Lakebase Autoscaling Postgres
  infrastructure, including projects, branches, compute endpoints, and roles.

  This API manages database infrastructure only. To query or modify data, use
  the Data API or direct SQL connections.

  **About resource IDs and names**

  Resources are identified by hierarchical resource names like
  projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}. The
  name field on each resource contains this full path and is output-only. Note
  that name refers to this resource path, not the user-visible display_name.`,
		GroupID: "postgres",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateBranch())
	cmd.AddCommand(newCreateCatalog())
	cmd.AddCommand(newCreateCdfConfig())
	cmd.AddCommand(newCreateDataApi())
	cmd.AddCommand(newCreateDatabase())
	cmd.AddCommand(newCreateEndpoint())
	cmd.AddCommand(newCreateProject())
	cmd.AddCommand(newCreateRole())
	cmd.AddCommand(newCreateSnapshot())
	cmd.AddCommand(newCreateSyncedTable())
	cmd.AddCommand(newDeleteBranch())
	cmd.AddCommand(newDeleteCatalog())
	cmd.AddCommand(newDeleteCdfConfig())
	cmd.AddCommand(newDeleteDataApi())
	cmd.AddCommand(newDeleteDatabase())
	cmd.AddCommand(newDeleteEndpoint())
	cmd.AddCommand(newDeleteProject())
	cmd.AddCommand(newDeleteRole())
	cmd.AddCommand(newDeleteSnapshot())
	cmd.AddCommand(newDeleteSyncedTable())
	cmd.AddCommand(newGenerateDatabaseCredential())
	cmd.AddCommand(newGetBranch())
	cmd.AddCommand(newGetCatalog())
	cmd.AddCommand(newGetCdfConfig())
	cmd.AddCommand(newGetCdfStatus())
	cmd.AddCommand(newGetDataApi())
	cmd.AddCommand(newGetDatabase())
	cmd.AddCommand(newGetEndpoint())
	cmd.AddCommand(newGetOperation())
	cmd.AddCommand(newGetProject())
	cmd.AddCommand(newGetRole())
	cmd.AddCommand(newGetSnapshot())
	cmd.AddCommand(newGetSnapshotSchedule())
	cmd.AddCommand(newGetSyncedTable())
	cmd.AddCommand(newListBranches())
	cmd.AddCommand(newListCdfConfigs())
	cmd.AddCommand(newListCdfStatuses())
	cmd.AddCommand(newListDatabases())
	cmd.AddCommand(newListEndpoints())
	cmd.AddCommand(newListProjects())
	cmd.AddCommand(newListRoles())
	cmd.AddCommand(newListSnapshots())
	cmd.AddCommand(newUndeleteBranch())
	cmd.AddCommand(newUndeleteProject())
	cmd.AddCommand(newUpdateBranch())
	cmd.AddCommand(newUpdateDataApi())
	cmd.AddCommand(newUpdateDatabase())
	cmd.AddCommand(newUpdateEndpoint())
	cmd.AddCommand(newUpdateProject())
	cmd.AddCommand(newUpdateRole())
	cmd.AddCommand(newUpdateSnapshotSchedule())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createBranchOverrides []func(
	*cobra.Command,
	*postgres.CreateBranchRequest,
)

func newCreateBranch() *cobra.Command {
	cmd := &cobra.Command{}
	var createBranchReq postgres.CreateBranchRequest
	createBranchReq.Branch = &postgres.Branch{}

	var createBranchJson flags.JsonFlag

	var createBranchSkipWait bool
	var createBranchTimeout time.Duration

	cmd.Flags().BoolVar(&createBranchSkipWait, "no-wait", createBranchSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createBranchTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createBranchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createBranchReq.ReplaceExisting, "replace-existing", `If true, update the branch if it already exists instead of returning an error.`)
	flags.StringPtrVar(cmd.Flags(), &createBranchReq.Branch.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-branch PARENT BRANCH_ID"
	cmd.Short = `Create a Branch.`
	cmd.Long = `Create a Branch.

  Creates a new database branch in the project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Project where this Branch will be created. Format:
      projects/{project_id}
    BRANCH_ID: The ID to use for the Branch. This becomes the final component of the
      branch's resource name. The ID is required and must be 1-63 characters
      long, start with a lowercase letter, and contain only lowercase letters,
      numbers, and hyphens. For example, development becomes
      projects/my-app/branches/development.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := createBranchJson.Unmarshal(&createBranchReq.Branch)
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
		createBranchReq.Parent = new(args[0])
		createBranchReq.BranchId = new(args[1])

		sdkReq, err := postgres.CreateBranchRequestToSDK(&createBranchReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createBranchSkipWait:
			wait, err := w.Postgres.CreateBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-branch to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createBranchTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.BranchFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createBranchOverrides {
		fn(cmd, &createBranchReq)
	}

	return cmd
}

// start create-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCatalogOverrides []func(
	*cobra.Command,
	*postgres.CreateCatalogRequest,
)

func newCreateCatalog() *cobra.Command {
	cmd := &cobra.Command{}
	var createCatalogReq postgres.CreateCatalogRequest
	createCatalogReq.Catalog = &postgres.Catalog{}

	var createCatalogJson flags.JsonFlag

	var createCatalogSkipWait bool
	var createCatalogTimeout time.Duration

	cmd.Flags().BoolVar(&createCatalogSkipWait, "no-wait", createCatalogSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createCatalogTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createCatalogJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createCatalogReq.Catalog.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-catalog CATALOG_ID"
	cmd.Short = `Register a Database in UC.`
	cmd.Long = `Register a Database in UC.

  Register a Postgres database in the Unity Catalog.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    CATALOG_ID: The ID in the Unity Catalog. It becomes the full resource name, for
      example "my_catalog" becomes "catalogs/my_catalog".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createCatalogJson.Unmarshal(&createCatalogReq.Catalog)
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
		createCatalogReq.CatalogId = new(args[0])

		sdkReq, err := postgres.CreateCatalogRequestToSDK(&createCatalogReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createCatalogSkipWait:
			wait, err := w.Postgres.CreateCatalog(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateCatalog(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-catalog to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createCatalogTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.CatalogFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCatalogOverrides {
		fn(cmd, &createCatalogReq)
	}

	return cmd
}

// start create-cdf-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCdfConfigOverrides []func(
	*cobra.Command,
	*postgres.CreateCdfConfigRequest,
)

func newCreateCdfConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var createCdfConfigReq postgres.CreateCdfConfigRequest
	createCdfConfigReq.CdfConfig = &postgres.CdfConfig{}

	var createCdfConfigJson flags.JsonFlag

	var createCdfConfigSkipWait bool
	var createCdfConfigTimeout time.Duration

	cmd.Flags().BoolVar(&createCdfConfigSkipWait, "no-wait", createCdfConfigSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createCdfConfigTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createCdfConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createCdfConfigReq.CdfConfigId, "cdf-config-id", `The user-specified id for the CdfConfig, forming the final segment of its resource name.`)
	flags.StringPtrVar(cmd.Flags(), &createCdfConfigReq.CdfConfig.Name, "name", `Output only.`)

	cmd.Use = "create-cdf-config PARENT CATALOG SCHEMA POSTGRES_SCHEMA"
	cmd.Short = `*Beta* Create a change data feed configuration.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a change data feed configuration.

  Create a CDF configuration that materializes the change data feed for all
  tables in a Postgres schema as open-format Delta tables in Unity Catalog. Once
  created, each table's change history is continuously written to its
  corresponding Lakehouse table.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The parent database under which to create the CdfConfig. Format:
      projects/{project}/branches/{branch}/databases/{database}
    CATALOG: The Unity Catalog catalog that replicated tables are written into. Set at
      creation; the CdfConfig is immutable.
    SCHEMA: The Unity Catalog schema that replicated tables are written into. Set at
      creation; the CdfConfig is immutable.
    POSTGRES_SCHEMA: The Postgres schema this CdfConfig replicates from. Unique within the
      parent database. Set at creation; the CdfConfig is immutable.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'catalog', 'schema', 'postgres_schema' in your JSON input")
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
			diags := createCdfConfigJson.Unmarshal(&createCdfConfigReq.CdfConfig)
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
		createCdfConfigReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			createCdfConfigReq.CdfConfig.Catalog = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createCdfConfigReq.CdfConfig.Schema = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			createCdfConfigReq.CdfConfig.PostgresSchema = new(args[3])
		}

		sdkReq, err := postgres.CreateCdfConfigRequestToSDK(&createCdfConfigReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createCdfConfigSkipWait:
			wait, err := w.Postgres.CreateCdfConfig(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateCdfConfig(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-cdf-config to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createCdfConfigTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.CdfConfigFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCdfConfigOverrides {
		fn(cmd, &createCdfConfigReq)
	}

	return cmd
}

// start create-data-api command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDataApiOverrides []func(
	*cobra.Command,
	*postgres.CreateDataApiRequest,
)

func newCreateDataApi() *cobra.Command {
	cmd := &cobra.Command{}
	var createDataApiReq postgres.CreateDataApiRequest
	createDataApiReq.DataApi = &postgres.DataApi{}

	var createDataApiJson flags.JsonFlag

	var createDataApiSkipWait bool
	var createDataApiTimeout time.Duration

	cmd.Flags().BoolVar(&createDataApiSkipWait, "no-wait", createDataApiSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createDataApiTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createDataApiJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createDataApiReq.DataApi.Name, "name", `Resource name: projects/{project_id}/branches/{branch_id}/databases/{database_id}/data-api.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-data-api PARENT"
	cmd.Short = `Enable Data API for a database.`
	cmd.Long = `Enable Data API for a database.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: Parent database:
      projects/{project_id}/branches/{branch_id}/databases/{database_id}`

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

		if cmd.Flags().Changed("json") {
			diags := createDataApiJson.Unmarshal(&createDataApiReq.DataApi)
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
		createDataApiReq.Parent = new(args[0])

		sdkReq, err := postgres.CreateDataApiRequestToSDK(&createDataApiReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createDataApiSkipWait:
			wait, err := w.Postgres.CreateDataApi(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateDataApi(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-data-api to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createDataApiTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.DataApiFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDataApiOverrides {
		fn(cmd, &createDataApiReq)
	}

	return cmd
}

// start create-database command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseOverrides []func(
	*cobra.Command,
	*postgres.CreateDatabaseRequest,
)

func newCreateDatabase() *cobra.Command {
	cmd := &cobra.Command{}
	var createDatabaseReq postgres.CreateDatabaseRequest
	createDatabaseReq.Database = &postgres.Database{}

	var createDatabaseJson flags.JsonFlag

	var createDatabaseSkipWait bool
	var createDatabaseTimeout time.Duration

	cmd.Flags().BoolVar(&createDatabaseSkipWait, "no-wait", createDatabaseSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createDatabaseTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createDatabaseJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createDatabaseReq.DatabaseId, "database-id", `The ID to use for the Database, which will become the final component of the database's resource name.`)
	flags.BoolPtrVar(cmd.Flags(), &createDatabaseReq.ReplaceExisting, "replace-existing", `If true, update the database if it already exists instead of returning an error.`)
	flags.StringPtrVar(cmd.Flags(), &createDatabaseReq.Database.Name, "name", `The resource name of the database.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-database PARENT"
	cmd.Short = `Create a Database.`
	cmd.Long = `Create a Database.

  Create a Database.

  Creates a database in the specified branch. A branch can have multiple
  databases.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Branch where this Database will be created. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createDatabaseJson.Unmarshal(&createDatabaseReq.Database)
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
		createDatabaseReq.Parent = new(args[0])

		sdkReq, err := postgres.CreateDatabaseRequestToSDK(&createDatabaseReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createDatabaseSkipWait:
			wait, err := w.Postgres.CreateDatabase(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateDatabase(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-database to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createDatabaseTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.DatabaseFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseOverrides {
		fn(cmd, &createDatabaseReq)
	}

	return cmd
}

// start create-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createEndpointOverrides []func(
	*cobra.Command,
	*postgres.CreateEndpointRequest,
)

func newCreateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var createEndpointReq postgres.CreateEndpointRequest
	createEndpointReq.Endpoint = &postgres.Endpoint{}

	var createEndpointJson flags.JsonFlag

	var createEndpointSkipWait bool
	var createEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&createEndpointSkipWait, "no-wait", createEndpointSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createEndpointTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createEndpointReq.ReplaceExisting, "replace-existing", `If true, update the endpoint if it already exists instead of returning an error.`)
	flags.StringPtrVar(cmd.Flags(), &createEndpointReq.Endpoint.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-endpoint PARENT ENDPOINT_ID"
	cmd.Short = `Create an Endpoint.`
	cmd.Long = `Create an Endpoint.

  Creates a new compute endpoint in the branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Branch where this Endpoint will be created. Format:
      projects/{project_id}/branches/{branch_id}
    ENDPOINT_ID: The ID to use for the Endpoint. This becomes the final component of the
      endpoint's resource name. The ID is required and must be 1-63 characters
      long, start with a lowercase letter, and contain only lowercase letters,
      numbers, and hyphens. For example, primary becomes
      projects/my-app/branches/development/endpoints/primary.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := createEndpointJson.Unmarshal(&createEndpointReq.Endpoint)
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
		createEndpointReq.Parent = new(args[0])
		createEndpointReq.EndpointId = new(args[1])

		sdkReq, err := postgres.CreateEndpointRequestToSDK(&createEndpointReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createEndpointSkipWait:
			wait, err := w.Postgres.CreateEndpoint(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateEndpoint(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-endpoint to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createEndpointTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.EndpointFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createEndpointOverrides {
		fn(cmd, &createEndpointReq)
	}

	return cmd
}

// start create-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createProjectOverrides []func(
	*cobra.Command,
	*postgres.CreateProjectRequest,
)

func newCreateProject() *cobra.Command {
	cmd := &cobra.Command{}
	var createProjectReq postgres.CreateProjectRequest
	createProjectReq.Project = &postgres.Project{}

	var createProjectJson flags.JsonFlag

	var createProjectSkipWait bool
	var createProjectTimeout time.Duration

	cmd.Flags().BoolVar(&createProjectSkipWait, "no-wait", createProjectSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createProjectTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createProjectJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: initial_branch_spec
	// TODO: complex arg: initial_endpoint_spec
	flags.StringPtrVar(cmd.Flags(), &createProjectReq.Project.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-project PROJECT_ID"
	cmd.Short = `Create a Project.`
	cmd.Long = `Create a Project.

  Creates a new Lakebase Autoscaling Postgres database project, which contains
  branches and compute endpoints.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PROJECT_ID: The ID to use for the Project. This becomes the final component of the
      project's resource name. The ID is required and must be 1-63 characters
      long, start with a lowercase letter, and contain only lowercase letters,
      numbers, and hyphens. For example, my-app becomes projects/my-app.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createProjectJson.Unmarshal(&createProjectReq.Project)
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
		createProjectReq.ProjectId = new(args[0])

		sdkReq, err := postgres.CreateProjectRequestToSDK(&createProjectReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createProjectSkipWait:
			wait, err := w.Postgres.CreateProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-project to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createProjectTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.ProjectFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createProjectOverrides {
		fn(cmd, &createProjectReq)
	}

	return cmd
}

// start create-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createRoleOverrides []func(
	*cobra.Command,
	*postgres.CreateRoleRequest,
)

func newCreateRole() *cobra.Command {
	cmd := &cobra.Command{}
	var createRoleReq postgres.CreateRoleRequest
	createRoleReq.Role = &postgres.Role{}

	var createRoleJson flags.JsonFlag

	var createRoleSkipWait bool
	var createRoleTimeout time.Duration

	cmd.Flags().BoolVar(&createRoleSkipWait, "no-wait", createRoleSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createRoleTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createRoleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createRoleReq.ReplaceExisting, "replace-existing", `If true, update the role if it already exists instead of returning an error.`)
	flags.StringPtrVar(cmd.Flags(), &createRoleReq.RoleId, "role-id", `The ID to use for the Role, which will become the final component of the role's resource name.`)
	flags.StringPtrVar(cmd.Flags(), &createRoleReq.Role.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-role PARENT"
	cmd.Short = `Create a Postgres Role for a Branch.`
	cmd.Long = `Create a Postgres Role for a Branch.

  Creates a new Postgres role in the branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Branch where this Role is created. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createRoleJson.Unmarshal(&createRoleReq.Role)
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
		createRoleReq.Parent = new(args[0])

		sdkReq, err := postgres.CreateRoleRequestToSDK(&createRoleReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createRoleSkipWait:
			wait, err := w.Postgres.CreateRole(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateRole(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-role to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createRoleTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.RoleFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createRoleOverrides {
		fn(cmd, &createRoleReq)
	}

	return cmd
}

// start create-snapshot command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSnapshotOverrides []func(
	*cobra.Command,
	*postgres.CreateSnapshotRequest,
)

func newCreateSnapshot() *cobra.Command {
	cmd := &cobra.Command{}
	var createSnapshotReq postgres.CreateSnapshotRequest
	createSnapshotReq.Snapshot = &postgres.Snapshot{}

	var createSnapshotJson flags.JsonFlag

	var createSnapshotSkipWait bool
	var createSnapshotTimeout time.Duration

	cmd.Flags().BoolVar(&createSnapshotSkipWait, "no-wait", createSnapshotSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createSnapshotTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createSnapshotJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSnapshotReq.Snapshot.Name, "name", `The resource name of the snapshot.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-snapshot PARENT SNAPSHOT_ID"
	cmd.Short = `*Beta* Create a Snapshot.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a Snapshot.

  Creates a snapshot, an immutable point-in-time copy of a branch's data, within
  the project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The project in which to create the snapshot. Format: projects/{project_id}
    SNAPSHOT_ID: Client-chosen ID for the snapshot. It becomes the final segment of the
      snapshot resource name and cannot be changed after creation.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
			diags := createSnapshotJson.Unmarshal(&createSnapshotReq.Snapshot)
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
		createSnapshotReq.Parent = new(args[0])
		createSnapshotReq.SnapshotId = new(args[1])

		sdkReq, err := postgres.CreateSnapshotRequestToSDK(&createSnapshotReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createSnapshotSkipWait:
			wait, err := w.Postgres.CreateSnapshot(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateSnapshot(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-snapshot to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createSnapshotTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.SnapshotFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSnapshotOverrides {
		fn(cmd, &createSnapshotReq)
	}

	return cmd
}

// start create-synced-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSyncedTableOverrides []func(
	*cobra.Command,
	*postgres.CreateSyncedTableRequest,
)

func newCreateSyncedTable() *cobra.Command {
	cmd := &cobra.Command{}
	var createSyncedTableReq postgres.CreateSyncedTableRequest
	createSyncedTableReq.SyncedTable = &postgres.SyncedTable{}

	var createSyncedTableJson flags.JsonFlag

	var createSyncedTableSkipWait bool
	var createSyncedTableTimeout time.Duration

	cmd.Flags().BoolVar(&createSyncedTableSkipWait, "no-wait", createSyncedTableSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createSyncedTableTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createSyncedTableJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSyncedTableReq.SyncedTable.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-synced-table SYNCED_TABLE_ID"
	cmd.Short = `Create a Synced Database Table.`
	cmd.Long = `Create a Synced Database Table.

  Create a Synced Table.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    SYNCED_TABLE_ID: The ID to use for the Synced Table. This becomes the final component of
      the SyncedTable's resource name. ID is required and is the synced table
      name, containing (catalog, schema, table) tuple. Elements of the tuple are
      the UC entity names.

      Example: "{catalog}.{schema}.{table}"

      synced_table_id represents both of the following:

      1. An online VIEW virtual table in the Unity Catalog accessible via the
      Lakehouse Federation. 2. Postgres table named "{table}" in schema
      "{schema}" in the connected Postgres database`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createSyncedTableJson.Unmarshal(&createSyncedTableReq.SyncedTable)
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
		createSyncedTableReq.SyncedTableId = new(args[0])

		sdkReq, err := postgres.CreateSyncedTableRequestToSDK(&createSyncedTableReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createSyncedTableSkipWait:
			wait, err := w.Postgres.CreateSyncedTable(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.CreateSyncedTable(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-synced-table to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createSyncedTableTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.SyncedTableFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSyncedTableOverrides {
		fn(cmd, &createSyncedTableReq)
	}

	return cmd
}

// start delete-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteBranchOverrides []func(
	*cobra.Command,
	*postgres.DeleteBranchRequest,
)

func newDeleteBranch() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteBranchReq postgres.DeleteBranchRequest

	var deleteBranchSkipWait bool
	var deleteBranchTimeout time.Duration

	cmd.Flags().BoolVar(&deleteBranchSkipWait, "no-wait", deleteBranchSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteBranchTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	flags.BoolPtrVar(cmd.Flags(), &deleteBranchReq.Purge, "purge", `If true, permanently delete the branch; if false, soft delete.`)

	cmd.Use = "delete-branch NAME"
	cmd.Short = `Delete a Branch.`
	cmd.Long = `Delete a Branch.

  Deletes the specified database branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the branch to delete. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteBranchReq.Name = new(args[0])

		sdkReq, err := postgres.DeleteBranchRequestToSDK(&deleteBranchReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case deleteBranchSkipWait:
			wait, err := w.Postgres.DeleteBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-branch to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteBranchTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteBranchOverrides {
		fn(cmd, &deleteBranchReq)
	}

	return cmd
}

// start delete-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteCatalogOverrides []func(
	*cobra.Command,
	*sdkpostgres.DeleteCatalogRequest,
)

func newDeleteCatalog() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteCatalogReq sdkpostgres.DeleteCatalogRequest

	var deleteCatalogSkipWait bool
	var deleteCatalogTimeout time.Duration

	cmd.Flags().BoolVar(&deleteCatalogSkipWait, "no-wait", deleteCatalogSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteCatalogTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-catalog NAME"
	cmd.Short = `Delete a Database Catalog.`
	cmd.Long = `Delete a Database Catalog.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the catalog to delete.

      Format: "catalogs/{catalog_id}".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteCatalogReq.Name = new(args[0])

		sdkReq := &deleteCatalogReq
		// Determine which mode to execute based on flags.
		switch {
		case deleteCatalogSkipWait:
			wait, err := w.Postgres.DeleteCatalog(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteCatalog(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-catalog to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteCatalogTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteCatalogOverrides {
		fn(cmd, &deleteCatalogReq)
	}

	return cmd
}

// start delete-cdf-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteCdfConfigOverrides []func(
	*cobra.Command,
	*postgres.DeleteCdfConfigRequest,
)

func newDeleteCdfConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteCdfConfigReq postgres.DeleteCdfConfigRequest

	var deleteCdfConfigSkipWait bool
	var deleteCdfConfigTimeout time.Duration

	cmd.Flags().BoolVar(&deleteCdfConfigSkipWait, "no-wait", deleteCdfConfigSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteCdfConfigTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	flags.BoolPtrVar(cmd.Flags(), &deleteCdfConfigReq.Force, "force", `When true, also drops the replicated Delta tables in Unity Catalog.`)

	cmd.Use = "delete-cdf-config NAME"
	cmd.Short = `*Beta* Delete a change data feed configuration.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a change data feed configuration.

  Delete a CDF configuration and stop materializing the change data feed. When
  force=true, also drops the Delta tables in Unity Catalog. When force=false
  (default), the existing tables are preserved at their last state.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the CdfConfig to delete. Format:
      projects/{project}/branches/{branch}/databases/{database}/cdf-configs/{cdf_config}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteCdfConfigReq.Name = new(args[0])

		sdkReq, err := postgres.DeleteCdfConfigRequestToSDK(&deleteCdfConfigReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case deleteCdfConfigSkipWait:
			wait, err := w.Postgres.DeleteCdfConfig(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteCdfConfig(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-cdf-config to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteCdfConfigTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteCdfConfigOverrides {
		fn(cmd, &deleteCdfConfigReq)
	}

	return cmd
}

// start delete-data-api command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDataApiOverrides []func(
	*cobra.Command,
	*sdkpostgres.DeleteDataApiRequest,
)

func newDeleteDataApi() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDataApiReq sdkpostgres.DeleteDataApiRequest

	var deleteDataApiSkipWait bool
	var deleteDataApiTimeout time.Duration

	cmd.Flags().BoolVar(&deleteDataApiSkipWait, "no-wait", deleteDataApiSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteDataApiTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-data-api NAME"
	cmd.Short = `Disable Data API for a database.`
	cmd.Long = `Disable Data API for a database.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Resource name:
      projects/{project_id}/branches/{branch_id}/databases/{database_id}/data-api`

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

		deleteDataApiReq.Name = new(args[0])

		sdkReq := &deleteDataApiReq
		// Determine which mode to execute based on flags.
		switch {
		case deleteDataApiSkipWait:
			wait, err := w.Postgres.DeleteDataApi(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteDataApi(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-data-api to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteDataApiTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDataApiOverrides {
		fn(cmd, &deleteDataApiReq)
	}

	return cmd
}

// start delete-database command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseOverrides []func(
	*cobra.Command,
	*sdkpostgres.DeleteDatabaseRequest,
)

func newDeleteDatabase() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDatabaseReq sdkpostgres.DeleteDatabaseRequest

	var deleteDatabaseSkipWait bool
	var deleteDatabaseTimeout time.Duration

	cmd.Flags().BoolVar(&deleteDatabaseSkipWait, "no-wait", deleteDatabaseSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteDatabaseTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-database NAME"
	cmd.Short = `Delete a Database.`
	cmd.Long = `Delete a Database.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the postgres database. Format:
      projects/{project_id}/branches/{branch_id}/databases/{database_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteDatabaseReq.Name = new(args[0])

		sdkReq := &deleteDatabaseReq
		// Determine which mode to execute based on flags.
		switch {
		case deleteDatabaseSkipWait:
			wait, err := w.Postgres.DeleteDatabase(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteDatabase(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-database to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteDatabaseTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseOverrides {
		fn(cmd, &deleteDatabaseReq)
	}

	return cmd
}

// start delete-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteEndpointOverrides []func(
	*cobra.Command,
	*sdkpostgres.DeleteEndpointRequest,
)

func newDeleteEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteEndpointReq sdkpostgres.DeleteEndpointRequest

	var deleteEndpointSkipWait bool
	var deleteEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&deleteEndpointSkipWait, "no-wait", deleteEndpointSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteEndpointTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-endpoint NAME"
	cmd.Short = `Delete an Endpoint.`
	cmd.Long = `Delete an Endpoint.

  Deletes the specified compute endpoint.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the endpoint to delete. Format:
      projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteEndpointReq.Name = new(args[0])

		sdkReq := &deleteEndpointReq
		// Determine which mode to execute based on flags.
		switch {
		case deleteEndpointSkipWait:
			wait, err := w.Postgres.DeleteEndpoint(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteEndpoint(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-endpoint to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteEndpointTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteEndpointOverrides {
		fn(cmd, &deleteEndpointReq)
	}

	return cmd
}

// start delete-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteProjectOverrides []func(
	*cobra.Command,
	*postgres.DeleteProjectRequest,
)

func newDeleteProject() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteProjectReq postgres.DeleteProjectRequest

	var deleteProjectSkipWait bool
	var deleteProjectTimeout time.Duration

	cmd.Flags().BoolVar(&deleteProjectSkipWait, "no-wait", deleteProjectSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteProjectTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	flags.BoolPtrVar(cmd.Flags(), &deleteProjectReq.Purge, "purge", `If true, permanently deletes the project (hard delete).`)

	cmd.Use = "delete-project NAME"
	cmd.Short = `Delete a Project.`
	cmd.Long = `Delete a Project.

  Deletes the specified database project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the project to delete. Format:
      projects/{project_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteProjectReq.Name = new(args[0])

		sdkReq, err := postgres.DeleteProjectRequestToSDK(&deleteProjectReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case deleteProjectSkipWait:
			wait, err := w.Postgres.DeleteProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-project to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteProjectTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteProjectOverrides {
		fn(cmd, &deleteProjectReq)
	}

	return cmd
}

// start delete-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRoleOverrides []func(
	*cobra.Command,
	*postgres.DeleteRoleRequest,
)

func newDeleteRole() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteRoleReq postgres.DeleteRoleRequest

	var deleteRoleSkipWait bool
	var deleteRoleTimeout time.Duration

	cmd.Flags().BoolVar(&deleteRoleSkipWait, "no-wait", deleteRoleSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteRoleTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	flags.StringPtrVar(cmd.Flags(), &deleteRoleReq.ReassignOwnedTo, "reassign-owned-to", `Reassign objects.`)

	cmd.Use = "delete-role NAME"
	cmd.Short = `Delete a Postgres Role from a Branch.`
	cmd.Long = `Delete a Postgres Role from a Branch.

  Deletes the specified Postgres role.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the role to delete. Format:
      projects/{project_id}/branches/{branch_id}/roles/{role_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteRoleReq.Name = new(args[0])

		sdkReq, err := postgres.DeleteRoleRequestToSDK(&deleteRoleReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case deleteRoleSkipWait:
			wait, err := w.Postgres.DeleteRole(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteRole(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-role to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteRoleTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteRoleOverrides {
		fn(cmd, &deleteRoleReq)
	}

	return cmd
}

// start delete-snapshot command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSnapshotOverrides []func(
	*cobra.Command,
	*sdkpostgres.DeleteSnapshotRequest,
)

func newDeleteSnapshot() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSnapshotReq sdkpostgres.DeleteSnapshotRequest

	var deleteSnapshotSkipWait bool
	var deleteSnapshotTimeout time.Duration

	cmd.Flags().BoolVar(&deleteSnapshotSkipWait, "no-wait", deleteSnapshotSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteSnapshotTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-snapshot NAME"
	cmd.Short = `*Beta* Delete a Snapshot.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a Snapshot.

  Deletes the specified snapshot.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the snapshot to delete. Format:
      projects/{project_id}/snapshots/{snapshot_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteSnapshotReq.Name = new(args[0])

		sdkReq := &deleteSnapshotReq
		// Determine which mode to execute based on flags.
		switch {
		case deleteSnapshotSkipWait:
			wait, err := w.Postgres.DeleteSnapshot(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteSnapshot(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-snapshot to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteSnapshotTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSnapshotOverrides {
		fn(cmd, &deleteSnapshotReq)
	}

	return cmd
}

// start delete-synced-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSyncedTableOverrides []func(
	*cobra.Command,
	*sdkpostgres.DeleteSyncedTableRequest,
)

func newDeleteSyncedTable() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSyncedTableReq sdkpostgres.DeleteSyncedTableRequest

	var deleteSyncedTableSkipWait bool
	var deleteSyncedTableTimeout time.Duration

	cmd.Flags().BoolVar(&deleteSyncedTableSkipWait, "no-wait", deleteSyncedTableSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteSyncedTableTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-synced-table NAME"
	cmd.Short = `Delete a Synced Database Table.`
	cmd.Long = `Delete a Synced Database Table.

  Delete a Synced Table.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The Full resource name of the synced table, of the format
      "synced_tables/{catalog}.{schema}.{table}", where (catalog, schema, table)
      are the UC entity names.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteSyncedTableReq.Name = new(args[0])

		sdkReq := &deleteSyncedTableReq
		// Determine which mode to execute based on flags.
		switch {
		case deleteSyncedTableSkipWait:
			wait, err := w.Postgres.DeleteSyncedTable(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.DeleteSyncedTable(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-synced-table to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteSyncedTableTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSyncedTableOverrides {
		fn(cmd, &deleteSyncedTableReq)
	}

	return cmd
}

// start generate-database-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateDatabaseCredentialOverrides []func(
	*cobra.Command,
	*postgres.GenerateDatabaseCredentialRequest,
)

func newGenerateDatabaseCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var generateDatabaseCredentialReq postgres.GenerateDatabaseCredentialRequest

	var generateDatabaseCredentialJson flags.JsonFlag

	cmd.Flags().Var(&generateDatabaseCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: claims
	var expireTimeParam string
	cmd.Flags().StringVar(&expireTimeParam, "expire-time", expireTimeParam, `Timestamp in UTC of when this credential should expire.`)
	var ttlParam string
	cmd.Flags().StringVar(&ttlParam, "ttl", ttlParam, `The requested time-to-live for the generated credential token.`)

	cmd.Use = "generate-database-credential ENDPOINT"
	cmd.Short = `Generate OAuth credentials for a Postgres database.`
	cmd.Long = `Generate OAuth credentials for a Postgres database.

  Arguments:
    ENDPOINT: The endpoint resource name for which this credential will be generated.
      Format: projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'endpoint' in your JSON input")
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
		if !cmd.Flags().Changed("json") {
			generateDatabaseCredentialReq.Endpoint = new(args[0])
		}

		if expireTimeParam != "" {
			var expireTimeField sdkcoretypes.Time
			err = json.Unmarshal([]byte(strconv.Quote(expireTimeParam)), &expireTimeField)
			if err != nil {
				return fmt.Errorf("invalid EXPIRE_TIME: %s", expireTimeParam)
			}
			generateDatabaseCredentialReq.ExpireTime = &expireTimeField
		}

		if ttlParam != "" {
			var ttlField sdkcoretypes.Duration
			err = json.Unmarshal([]byte(strconv.Quote(ttlParam)), &ttlField)
			if err != nil {
				return fmt.Errorf("invalid TTL: %s", ttlParam)
			}
			generateDatabaseCredentialReq.Ttl = &ttlField
		}

		sdkReq, err := postgres.GenerateDatabaseCredentialRequestToSDK(&generateDatabaseCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.Postgres.GenerateDatabaseCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.DatabaseCredentialFromSDK(response)
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

// start get-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBranchOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetBranchRequest,
)

func newGetBranch() *cobra.Command {
	cmd := &cobra.Command{}
	var getBranchReq sdkpostgres.GetBranchRequest

	cmd.Use = "get-branch NAME"
	cmd.Short = `Get a Branch.`
	cmd.Long = `Get a Branch.

  Retrieves information about the specified database branch.

  Arguments:
    NAME: The full resource path of the branch to retrieve. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getBranchReq.Name = new(args[0])

		sdkReq := &getBranchReq
		response, err := w.Postgres.GetBranch(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.BranchFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBranchOverrides {
		fn(cmd, &getBranchReq)
	}

	return cmd
}

// start get-catalog command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCatalogOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetCatalogRequest,
)

func newGetCatalog() *cobra.Command {
	cmd := &cobra.Command{}
	var getCatalogReq sdkpostgres.GetCatalogRequest

	cmd.Use = "get-catalog NAME"
	cmd.Short = `Get a Database Catalog.`
	cmd.Long = `Get a Database Catalog.

  Arguments:
    NAME: The full resource path of the catalog to retrieve.

      Format: "catalogs/{catalog_id}".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getCatalogReq.Name = new(args[0])

		sdkReq := &getCatalogReq
		response, err := w.Postgres.GetCatalog(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.CatalogFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCatalogOverrides {
		fn(cmd, &getCatalogReq)
	}

	return cmd
}

// start get-cdf-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCdfConfigOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetCdfConfigRequest,
)

func newGetCdfConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var getCdfConfigReq sdkpostgres.GetCdfConfigRequest

	cmd.Use = "get-cdf-config NAME"
	cmd.Short = `*Beta* Get a change data feed configuration.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a change data feed configuration.

  Get a single Lakebase CDF configuration, including the source Postgres schema
  and target Unity Catalog schema.

  Arguments:
    NAME: The resource name of the CdfConfig to retrieve. Format:
      projects/{project}/branches/{branch}/databases/{database}/cdf-configs/{cdf_config}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getCdfConfigReq.Name = new(args[0])

		sdkReq := &getCdfConfigReq
		response, err := w.Postgres.GetCdfConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.CdfConfigFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCdfConfigOverrides {
		fn(cmd, &getCdfConfigReq)
	}

	return cmd
}

// start get-cdf-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCdfStatusOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetCdfStatusRequest,
)

func newGetCdfStatus() *cobra.Command {
	cmd := &cobra.Command{}
	var getCdfStatusReq sdkpostgres.GetCdfStatusRequest

	cmd.Use = "get-cdf-status NAME"
	cmd.Short = `*Beta* Get a change data feed status.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a change data feed status.

  Get the CDF status of a single table within a Lakebase CDF configuration,
  including its current state and the last committed position in the feed.

  Arguments:
    NAME: The resource name of the CdfStatus to retrieve. Format:
      projects/{project}/branches/{branch}/databases/{database}/cdf-configs/{cdf_config}/cdf-statuses/{cdf_status}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getCdfStatusReq.Name = new(args[0])

		sdkReq := &getCdfStatusReq
		response, err := w.Postgres.GetCdfStatus(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.CdfStatusFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCdfStatusOverrides {
		fn(cmd, &getCdfStatusReq)
	}

	return cmd
}

// start get-data-api command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDataApiOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetDataApiRequest,
)

func newGetDataApi() *cobra.Command {
	cmd := &cobra.Command{}
	var getDataApiReq sdkpostgres.GetDataApiRequest

	cmd.Use = "get-data-api NAME"
	cmd.Short = `Get Data API configuration.`
	cmd.Long = `Get Data API configuration.

  Get Data API configuration for a database.

  Arguments:
    NAME: Resource name:
      projects/{project_id}/branches/{branch_id}/databases/{database_id}/data-api`

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

		getDataApiReq.Name = new(args[0])

		sdkReq := &getDataApiReq
		response, err := w.Postgres.GetDataApi(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.DataApiFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDataApiOverrides {
		fn(cmd, &getDataApiReq)
	}

	return cmd
}

// start get-database command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetDatabaseRequest,
)

func newGetDatabase() *cobra.Command {
	cmd := &cobra.Command{}
	var getDatabaseReq sdkpostgres.GetDatabaseRequest

	cmd.Use = "get-database NAME"
	cmd.Short = `Get a Database.`
	cmd.Long = `Get a Database.

  Arguments:
    NAME: The name of the Database to retrieve. Format:
      projects/{project_id}/branches/{branch_id}/databases/{database_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getDatabaseReq.Name = new(args[0])

		sdkReq := &getDatabaseReq
		response, err := w.Postgres.GetDatabase(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.DatabaseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseOverrides {
		fn(cmd, &getDatabaseReq)
	}

	return cmd
}

// start get-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEndpointOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetEndpointRequest,
)

func newGetEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var getEndpointReq sdkpostgres.GetEndpointRequest

	cmd.Use = "get-endpoint NAME"
	cmd.Short = `Get an Endpoint.`
	cmd.Long = `Get an Endpoint.

  Retrieves information about the specified compute endpoint, including its
  connection details and operational state.

  Arguments:
    NAME: The full resource path of the endpoint to retrieve. Format:
      projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getEndpointReq.Name = new(args[0])

		sdkReq := &getEndpointReq
		response, err := w.Postgres.GetEndpoint(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.EndpointFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEndpointOverrides {
		fn(cmd, &getEndpointReq)
	}

	return cmd
}

// start get-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOperationOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetBranchRequest,
)

func newGetOperation() *cobra.Command {
	cmd := &cobra.Command{}
	var getOperationReq sdkpostgres.GetBranchRequest

	cmd.Use = "get-operation NAME"
	cmd.Short = `Get an Operation.`
	cmd.Long = `Get an Operation.

  Retrieves the status of a long-running operation.

  Arguments:
    NAME: The name of the operation resource.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getOperationReq.Name = new(args[0])

		sdkReq := &getOperationReq
		response, err := w.Postgres.GetBranch(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.BranchFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOperationOverrides {
		fn(cmd, &getOperationReq)
	}

	return cmd
}

// start get-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getProjectOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetProjectRequest,
)

func newGetProject() *cobra.Command {
	cmd := &cobra.Command{}
	var getProjectReq sdkpostgres.GetProjectRequest

	cmd.Use = "get-project NAME"
	cmd.Short = `Get a Project.`
	cmd.Long = `Get a Project.

  Retrieves information about the specified database project.

  Arguments:
    NAME: The full resource path of the project to retrieve. Format:
      projects/{project_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getProjectReq.Name = new(args[0])

		sdkReq := &getProjectReq
		response, err := w.Postgres.GetProject(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.ProjectFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getProjectOverrides {
		fn(cmd, &getProjectReq)
	}

	return cmd
}

// start get-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRoleOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetRoleRequest,
)

func newGetRole() *cobra.Command {
	cmd := &cobra.Command{}
	var getRoleReq sdkpostgres.GetRoleRequest

	cmd.Use = "get-role NAME"
	cmd.Short = `Get a Postgres Role for a Branch.`
	cmd.Long = `Get a Postgres Role for a Branch.

  Retrieves information about the specified Postgres role, including its
  authentication method and permissions.

  Arguments:
    NAME: The full resource path of the role to retrieve. Format:
      projects/{project_id}/branches/{branch_id}/roles/{role_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getRoleReq.Name = new(args[0])

		sdkReq := &getRoleReq
		response, err := w.Postgres.GetRole(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.RoleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRoleOverrides {
		fn(cmd, &getRoleReq)
	}

	return cmd
}

// start get-snapshot command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSnapshotOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetSnapshotRequest,
)

func newGetSnapshot() *cobra.Command {
	cmd := &cobra.Command{}
	var getSnapshotReq sdkpostgres.GetSnapshotRequest

	cmd.Use = "get-snapshot NAME"
	cmd.Short = `*Beta* Get a Snapshot.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a Snapshot.

  Retrieves information about the specified snapshot.

  Arguments:
    NAME: The resource name of the snapshot to retrieve. Format:
      projects/{project_id}/snapshots/{snapshot_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getSnapshotReq.Name = new(args[0])

		sdkReq := &getSnapshotReq
		response, err := w.Postgres.GetSnapshot(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.SnapshotFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSnapshotOverrides {
		fn(cmd, &getSnapshotReq)
	}

	return cmd
}

// start get-snapshot-schedule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSnapshotScheduleOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetSnapshotScheduleRequest,
)

func newGetSnapshotSchedule() *cobra.Command {
	cmd := &cobra.Command{}
	var getSnapshotScheduleReq sdkpostgres.GetSnapshotScheduleRequest

	cmd.Use = "get-snapshot-schedule NAME"
	cmd.Short = `*Beta* Get a Snapshot Schedule.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a Snapshot Schedule.

  Retrieves the snapshot schedule for a branch. A branch with no configured
  schedule returns an empty schedule (not NOT_FOUND).

  Arguments:
    NAME: The resource name of the branch's snapshot schedule. Format:
      projects/{project_id}/branches/{branch_id}/snapshot-schedule`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getSnapshotScheduleReq.Name = new(args[0])

		sdkReq := &getSnapshotScheduleReq
		response, err := w.Postgres.GetSnapshotSchedule(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.SnapshotScheduleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSnapshotScheduleOverrides {
		fn(cmd, &getSnapshotScheduleReq)
	}

	return cmd
}

// start get-synced-table command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSyncedTableOverrides []func(
	*cobra.Command,
	*sdkpostgres.GetSyncedTableRequest,
)

func newGetSyncedTable() *cobra.Command {
	cmd := &cobra.Command{}
	var getSyncedTableReq sdkpostgres.GetSyncedTableRequest

	cmd.Use = "get-synced-table NAME"
	cmd.Short = `Get a Synced Database Table.`
	cmd.Long = `Get a Synced Database Table.

  Get a Synced Table.

  Arguments:
    NAME: The Full resource name of the synced table. Format:
      "synced_tables/{catalog}.{schema}.{table}", where (catalog, schema, table)
      are the entity names in the Unity Catalog.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getSyncedTableReq.Name = new(args[0])

		sdkReq := &getSyncedTableReq
		response, err := w.Postgres.GetSyncedTable(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := postgres.SyncedTableFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSyncedTableOverrides {
		fn(cmd, &getSyncedTableReq)
	}

	return cmd
}

// start list-branches command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listBranchesOverrides []func(
	*cobra.Command,
	*postgres.ListBranchesRequest,
)

func newListBranches() *cobra.Command {
	cmd := &cobra.Command{}
	var listBranchesReq postgres.ListBranchesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listBranchesLimit int

	flags.IntPtrVar(cmd.Flags(), &listBranchesReq.PageSize, "page-size", `Upper bound for items returned.`)
	flags.BoolPtrVar(cmd.Flags(), &listBranchesReq.ShowDeleted, "show-deleted", `Whether to include soft-deleted branches in the response.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listBranchesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listBranchesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-branches PARENT"
	cmd.Short = `List Branches.`
	cmd.Long = `List Branches.

  Returns a paginated list of database branches in the project.

  Arguments:
    PARENT: The Project that owns this collection of branches. Format:
      projects/{project_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listBranchesReq.Parent = new(args[0])

		sdkReq, err := postgres.ListBranchesRequestToSDK(&listBranchesReq)
		if err != nil {
			return err
		}
		if listBranchesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listBranchesLimit)
		}
		if listBranchesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listBranchesLimit)
		}
		response := w.Postgres.ListBranchesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.BranchFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listBranchesOverrides {
		fn(cmd, &listBranchesReq)
	}

	return cmd
}

// start list-cdf-configs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listCdfConfigsOverrides []func(
	*cobra.Command,
	*postgres.ListCdfConfigsRequest,
)

func newListCdfConfigs() *cobra.Command {
	cmd := &cobra.Command{}
	var listCdfConfigsReq postgres.ListCdfConfigsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listCdfConfigsLimit int

	flags.IntPtrVar(cmd.Flags(), &listCdfConfigsReq.PageSize, "page-size", `Maximum number of CdfConfigs to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listCdfConfigsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listCdfConfigsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-cdf-configs PARENT"
	cmd.Short = `*Beta* List change data feed configurations.`
	cmd.Long = `This command is in Beta and may change without notice.

List change data feed configurations.

  List all CDF configurations for a Lakebase database. Each configuration maps a
  Postgres schema to a Unity Catalog schema where the change data feed is
  materialized.

  Arguments:
    PARENT: The parent database to list CdfConfigs for. Format:
      projects/{project}/branches/{branch}/databases/{database}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listCdfConfigsReq.Parent = new(args[0])

		sdkReq, err := postgres.ListCdfConfigsRequestToSDK(&listCdfConfigsReq)
		if err != nil {
			return err
		}
		if listCdfConfigsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listCdfConfigsLimit)
		}
		if listCdfConfigsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listCdfConfigsLimit)
		}
		response := w.Postgres.ListCdfConfigsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.CdfConfigFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listCdfConfigsOverrides {
		fn(cmd, &listCdfConfigsReq)
	}

	return cmd
}

// start list-cdf-statuses command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listCdfStatusesOverrides []func(
	*cobra.Command,
	*postgres.ListCdfStatusesRequest,
)

func newListCdfStatuses() *cobra.Command {
	cmd := &cobra.Command{}
	var listCdfStatusesReq postgres.ListCdfStatusesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listCdfStatusesLimit int

	flags.IntPtrVar(cmd.Flags(), &listCdfStatusesReq.PageSize, "page-size", `Maximum number of CdfStatuses to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listCdfStatusesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listCdfStatusesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-cdf-statuses PARENT"
	cmd.Short = `*Beta* List change data feed statuses.`
	cmd.Long = `This command is in Beta and may change without notice.

List change data feed statuses.

  List the per-table CDF statuses within a Lakebase CDF configuration. Each
  status shows whether a table's change data feed is snapshotting, streaming, or
  skipped.

  Arguments:
    PARENT: The parent CdfConfig to list CdfStatuses for. Format:
      projects/{project}/branches/{branch}/databases/{database}/cdf-configs/{cdf_config}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listCdfStatusesReq.Parent = new(args[0])

		sdkReq, err := postgres.ListCdfStatusesRequestToSDK(&listCdfStatusesReq)
		if err != nil {
			return err
		}
		if listCdfStatusesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listCdfStatusesLimit)
		}
		if listCdfStatusesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listCdfStatusesLimit)
		}
		response := w.Postgres.ListCdfStatusesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.CdfStatusFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listCdfStatusesOverrides {
		fn(cmd, &listCdfStatusesReq)
	}

	return cmd
}

// start list-databases command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatabasesOverrides []func(
	*cobra.Command,
	*postgres.ListDatabasesRequest,
)

func newListDatabases() *cobra.Command {
	cmd := &cobra.Command{}
	var listDatabasesReq postgres.ListDatabasesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDatabasesLimit int

	flags.IntPtrVar(cmd.Flags(), &listDatabasesReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDatabasesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listDatabasesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-databases PARENT"
	cmd.Short = `List postgres databases in a branch.`
	cmd.Long = `List postgres databases in a branch.

  List Databases.

  Arguments:
    PARENT: The Branch that owns this collection of databases. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listDatabasesReq.Parent = new(args[0])

		sdkReq, err := postgres.ListDatabasesRequestToSDK(&listDatabasesReq)
		if err != nil {
			return err
		}
		if listDatabasesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDatabasesLimit)
		}
		if listDatabasesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDatabasesLimit)
		}
		response := w.Postgres.ListDatabasesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.DatabaseFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatabasesOverrides {
		fn(cmd, &listDatabasesReq)
	}

	return cmd
}

// start list-endpoints command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listEndpointsOverrides []func(
	*cobra.Command,
	*postgres.ListEndpointsRequest,
)

func newListEndpoints() *cobra.Command {
	cmd := &cobra.Command{}
	var listEndpointsReq postgres.ListEndpointsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listEndpointsLimit int

	flags.IntPtrVar(cmd.Flags(), &listEndpointsReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listEndpointsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listEndpointsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-endpoints PARENT"
	cmd.Short = `List Endpoints.`
	cmd.Long = `List Endpoints.

  Returns a paginated list of compute endpoints in the branch.

  Arguments:
    PARENT: The Branch that owns this collection of endpoints. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listEndpointsReq.Parent = new(args[0])

		sdkReq, err := postgres.ListEndpointsRequestToSDK(&listEndpointsReq)
		if err != nil {
			return err
		}
		if listEndpointsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listEndpointsLimit)
		}
		if listEndpointsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listEndpointsLimit)
		}
		response := w.Postgres.ListEndpointsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.EndpointFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listEndpointsOverrides {
		fn(cmd, &listEndpointsReq)
	}

	return cmd
}

// start list-projects command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listProjectsOverrides []func(
	*cobra.Command,
	*postgres.ListProjectsRequest,
)

func newListProjects() *cobra.Command {
	cmd := &cobra.Command{}
	var listProjectsReq postgres.ListProjectsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listProjectsLimit int

	flags.IntPtrVar(cmd.Flags(), &listProjectsReq.PageSize, "page-size", `Upper bound for items returned.`)
	flags.BoolPtrVar(cmd.Flags(), &listProjectsReq.ShowDeleted, "show-deleted", `Whether to include soft-deleted projects in the response.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listProjectsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listProjectsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-projects"
	cmd.Short = `List Projects.`
	cmd.Long = `List Projects.

  Returns a paginated list of database projects in the workspace that the user
  has permission to access.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := postgres.ListProjectsRequestToSDK(&listProjectsReq)
		if err != nil {
			return err
		}
		if listProjectsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listProjectsLimit)
		}
		if listProjectsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listProjectsLimit)
		}
		response := w.Postgres.ListProjectsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.ProjectFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listProjectsOverrides {
		fn(cmd, &listProjectsReq)
	}

	return cmd
}

// start list-roles command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listRolesOverrides []func(
	*cobra.Command,
	*postgres.ListRolesRequest,
)

func newListRoles() *cobra.Command {
	cmd := &cobra.Command{}
	var listRolesReq postgres.ListRolesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listRolesLimit int

	flags.IntPtrVar(cmd.Flags(), &listRolesReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listRolesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listRolesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-roles PARENT"
	cmd.Short = `List Postgres Roles for a Branch.`
	cmd.Long = `List Postgres Roles for a Branch.

  Returns a paginated list of Postgres roles in the branch.

  Arguments:
    PARENT: The Branch that owns this collection of roles. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listRolesReq.Parent = new(args[0])

		sdkReq, err := postgres.ListRolesRequestToSDK(&listRolesReq)
		if err != nil {
			return err
		}
		if listRolesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listRolesLimit)
		}
		if listRolesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listRolesLimit)
		}
		response := w.Postgres.ListRolesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.RoleFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listRolesOverrides {
		fn(cmd, &listRolesReq)
	}

	return cmd
}

// start list-snapshots command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSnapshotsOverrides []func(
	*cobra.Command,
	*postgres.ListSnapshotsRequest,
)

func newListSnapshots() *cobra.Command {
	cmd := &cobra.Command{}
	var listSnapshotsReq postgres.ListSnapshotsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSnapshotsLimit int

	flags.IntPtrVar(cmd.Flags(), &listSnapshotsReq.PageSize, "page-size", `Maximum number of snapshots to return per page.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSnapshotsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSnapshotsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-snapshots PARENT"
	cmd.Short = `*Beta* List Snapshots.`
	cmd.Long = `This command is in Beta and may change without notice.

List Snapshots.

  Returns a paginated list of snapshots in the project.

  Arguments:
    PARENT: The project that owns the snapshots. Format: projects/{project_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listSnapshotsReq.Parent = new(args[0])

		sdkReq, err := postgres.ListSnapshotsRequestToSDK(&listSnapshotsReq)
		if err != nil {
			return err
		}
		if listSnapshotsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSnapshotsLimit)
		}
		if listSnapshotsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSnapshotsLimit)
		}
		response := w.Postgres.ListSnapshotsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, postgres.SnapshotFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSnapshotsOverrides {
		fn(cmd, &listSnapshotsReq)
	}

	return cmd
}

// start undelete-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var undeleteBranchOverrides []func(
	*cobra.Command,
	*postgres.UndeleteBranchRequest,
)

func newUndeleteBranch() *cobra.Command {
	cmd := &cobra.Command{}
	var undeleteBranchReq postgres.UndeleteBranchRequest

	var undeleteBranchSkipWait bool
	var undeleteBranchTimeout time.Duration

	cmd.Flags().BoolVar(&undeleteBranchSkipWait, "no-wait", undeleteBranchSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&undeleteBranchTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "undelete-branch NAME"
	cmd.Short = `Undelete a Branch.`
	cmd.Long = `Undelete a Branch.

  Undeletes the specified database branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the branch to undelete. Format:
      projects/{project_id}/branches/{branch_id}`

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

		undeleteBranchReq.Name = new(args[0])

		sdkReq, err := postgres.UndeleteBranchRequestToSDK(&undeleteBranchReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case undeleteBranchSkipWait:
			wait, err := w.Postgres.UndeleteBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UndeleteBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for undelete-branch to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(undeleteBranchTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range undeleteBranchOverrides {
		fn(cmd, &undeleteBranchReq)
	}

	return cmd
}

// start undelete-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var undeleteProjectOverrides []func(
	*cobra.Command,
	*postgres.UndeleteProjectRequest,
)

func newUndeleteProject() *cobra.Command {
	cmd := &cobra.Command{}
	var undeleteProjectReq postgres.UndeleteProjectRequest

	var undeleteProjectSkipWait bool
	var undeleteProjectTimeout time.Duration

	cmd.Flags().BoolVar(&undeleteProjectSkipWait, "no-wait", undeleteProjectSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&undeleteProjectTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "undelete-project NAME"
	cmd.Short = `*Beta* Undelete a Project.`
	cmd.Long = `This command is in Beta and may change without notice.

Undelete a Project.

  Undeletes a soft-deleted project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the project to undelete. Format:
      projects/{project_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		undeleteProjectReq.Name = new(args[0])

		sdkReq, err := postgres.UndeleteProjectRequestToSDK(&undeleteProjectReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case undeleteProjectSkipWait:
			wait, err := w.Postgres.UndeleteProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UndeleteProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for undelete-project to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(undeleteProjectTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range undeleteProjectOverrides {
		fn(cmd, &undeleteProjectReq)
	}

	return cmd
}

// start update-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateBranchOverrides []func(
	*cobra.Command,
	*postgres.UpdateBranchRequest,
)

func newUpdateBranch() *cobra.Command {
	cmd := &cobra.Command{}
	var updateBranchReq postgres.UpdateBranchRequest
	updateBranchReq.Branch = &postgres.Branch{}

	var updateBranchJson flags.JsonFlag

	var updateBranchSkipWait bool
	var updateBranchTimeout time.Duration

	cmd.Flags().BoolVar(&updateBranchSkipWait, "no-wait", updateBranchSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateBranchTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateBranchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateBranchReq.Branch.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-branch NAME UPDATE_MASK"
	cmd.Short = `Update a Branch.`
	cmd.Long = `Update a Branch.

  Updates the specified database branch. You can set this branch as the
  project's default branch, or protect/unprotect it.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Output only. The full resource path of the branch. Format:
      projects/{project_id}/branches/{branch_id}
    UPDATE_MASK: The list of fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := updateBranchJson.Unmarshal(&updateBranchReq.Branch)
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
		updateBranchReq.Branch.Name = new(args[0])
		if args[1] != "" {
			updateBranchReq.UpdateMask = new(args[1])
		}

		sdkReq, err := postgres.UpdateBranchRequestToSDK(&updateBranchReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateBranchSkipWait:
			wait, err := w.Postgres.UpdateBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UpdateBranch(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-branch to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateBranchTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.BranchFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateBranchOverrides {
		fn(cmd, &updateBranchReq)
	}

	return cmd
}

// start update-data-api command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDataApiOverrides []func(
	*cobra.Command,
	*postgres.UpdateDataApiRequest,
)

func newUpdateDataApi() *cobra.Command {
	cmd := &cobra.Command{}
	var updateDataApiReq postgres.UpdateDataApiRequest
	updateDataApiReq.DataApi = &postgres.DataApi{}

	var updateDataApiJson flags.JsonFlag

	var updateDataApiSkipWait bool
	var updateDataApiTimeout time.Duration

	cmd.Flags().BoolVar(&updateDataApiSkipWait, "no-wait", updateDataApiSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateDataApiTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateDataApiJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateDataApiReq.DataApi.Name, "name", `Resource name: projects/{project_id}/branches/{branch_id}/databases/{database_id}/data-api.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-data-api NAME UPDATE_MASK"
	cmd.Short = `Update Data API configuration.`
	cmd.Long = `Update Data API configuration.

  Update Data API configuration for a database.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Resource name:
      projects/{project_id}/branches/{branch_id}/databases/{database_id}/data-api
    UPDATE_MASK: The list of fields to update.`

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
			diags := updateDataApiJson.Unmarshal(&updateDataApiReq.DataApi)
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
		updateDataApiReq.DataApi.Name = new(args[0])
		if args[1] != "" {
			updateDataApiReq.UpdateMask = new(args[1])
		}

		sdkReq, err := postgres.UpdateDataApiRequestToSDK(&updateDataApiReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateDataApiSkipWait:
			wait, err := w.Postgres.UpdateDataApi(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UpdateDataApi(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-data-api to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateDataApiTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.DataApiFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDataApiOverrides {
		fn(cmd, &updateDataApiReq)
	}

	return cmd
}

// start update-database command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatabaseOverrides []func(
	*cobra.Command,
	*postgres.UpdateDatabaseRequest,
)

func newUpdateDatabase() *cobra.Command {
	cmd := &cobra.Command{}
	var updateDatabaseReq postgres.UpdateDatabaseRequest
	updateDatabaseReq.Database = &postgres.Database{}

	var updateDatabaseJson flags.JsonFlag

	var updateDatabaseSkipWait bool
	var updateDatabaseTimeout time.Duration

	cmd.Flags().BoolVar(&updateDatabaseSkipWait, "no-wait", updateDatabaseSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateDatabaseTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateDatabaseJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateDatabaseReq.Database.Name, "name", `The resource name of the database.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-database NAME UPDATE_MASK"
	cmd.Short = `Update a Database.`
	cmd.Long = `Update a Database.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the database. Format:
      projects/{project_id}/branches/{branch_id}/databases/{database_id}
    UPDATE_MASK: The list of fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := updateDatabaseJson.Unmarshal(&updateDatabaseReq.Database)
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
		updateDatabaseReq.Database.Name = new(args[0])
		if args[1] != "" {
			updateDatabaseReq.UpdateMask = new(args[1])
		}

		sdkReq, err := postgres.UpdateDatabaseRequestToSDK(&updateDatabaseReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateDatabaseSkipWait:
			wait, err := w.Postgres.UpdateDatabase(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UpdateDatabase(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-database to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateDatabaseTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.DatabaseFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatabaseOverrides {
		fn(cmd, &updateDatabaseReq)
	}

	return cmd
}

// start update-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateEndpointOverrides []func(
	*cobra.Command,
	*postgres.UpdateEndpointRequest,
)

func newUpdateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}
	var updateEndpointReq postgres.UpdateEndpointRequest
	updateEndpointReq.Endpoint = &postgres.Endpoint{}

	var updateEndpointJson flags.JsonFlag

	var updateEndpointSkipWait bool
	var updateEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&updateEndpointSkipWait, "no-wait", updateEndpointSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateEndpointTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateEndpointReq.Endpoint.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-endpoint NAME UPDATE_MASK"
	cmd.Short = `Update an Endpoint.`
	cmd.Long = `Update an Endpoint.

  Updates the specified compute endpoint. You can update autoscaling limits,
  suspend timeout, or enable/disable the compute endpoint.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Output only. The full resource path of the endpoint. Format:
      projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}
    UPDATE_MASK: The list of fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := updateEndpointJson.Unmarshal(&updateEndpointReq.Endpoint)
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
		updateEndpointReq.Endpoint.Name = new(args[0])
		if args[1] != "" {
			updateEndpointReq.UpdateMask = new(args[1])
		}

		sdkReq, err := postgres.UpdateEndpointRequestToSDK(&updateEndpointReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateEndpointSkipWait:
			wait, err := w.Postgres.UpdateEndpoint(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UpdateEndpoint(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-endpoint to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateEndpointTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.EndpointFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateEndpointOverrides {
		fn(cmd, &updateEndpointReq)
	}

	return cmd
}

// start update-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateProjectOverrides []func(
	*cobra.Command,
	*postgres.UpdateProjectRequest,
)

func newUpdateProject() *cobra.Command {
	cmd := &cobra.Command{}
	var updateProjectReq postgres.UpdateProjectRequest
	updateProjectReq.Project = &postgres.Project{}

	var updateProjectJson flags.JsonFlag

	var updateProjectSkipWait bool
	var updateProjectTimeout time.Duration

	cmd.Flags().BoolVar(&updateProjectSkipWait, "no-wait", updateProjectSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateProjectTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateProjectJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: initial_branch_spec
	// TODO: complex arg: initial_endpoint_spec
	flags.StringPtrVar(cmd.Flags(), &updateProjectReq.Project.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-project NAME UPDATE_MASK"
	cmd.Short = `Update a Project.`
	cmd.Long = `Update a Project.

  Updates the specified database project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Output only. The full resource path of the project. Format:
      projects/{project_id}
    UPDATE_MASK: The list of fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := updateProjectJson.Unmarshal(&updateProjectReq.Project)
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
		updateProjectReq.Project.Name = new(args[0])
		if args[1] != "" {
			updateProjectReq.UpdateMask = new(args[1])
		}

		sdkReq, err := postgres.UpdateProjectRequestToSDK(&updateProjectReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateProjectSkipWait:
			wait, err := w.Postgres.UpdateProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UpdateProject(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-project to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateProjectTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.ProjectFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateProjectOverrides {
		fn(cmd, &updateProjectReq)
	}

	return cmd
}

// start update-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRoleOverrides []func(
	*cobra.Command,
	*postgres.UpdateRoleRequest,
)

func newUpdateRole() *cobra.Command {
	cmd := &cobra.Command{}
	var updateRoleReq postgres.UpdateRoleRequest
	updateRoleReq.Role = &postgres.Role{}

	var updateRoleJson flags.JsonFlag

	var updateRoleSkipWait bool
	var updateRoleTimeout time.Duration

	cmd.Flags().BoolVar(&updateRoleSkipWait, "no-wait", updateRoleSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateRoleTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateRoleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateRoleReq.Role.Name, "name", `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-role NAME UPDATE_MASK"
	cmd.Short = `Update a Postgres Role for a Branch.`
	cmd.Long = `Update a Postgres Role for a Branch.

  Update a role for a branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Output only. The full resource path of the role. Format:
      projects/{project_id}/branches/{branch_id}/roles/{role_id}
    UPDATE_MASK: The list of fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
			diags := updateRoleJson.Unmarshal(&updateRoleReq.Role)
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
		updateRoleReq.Role.Name = new(args[0])
		if args[1] != "" {
			updateRoleReq.UpdateMask = new(args[1])
		}

		sdkReq, err := postgres.UpdateRoleRequestToSDK(&updateRoleReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateRoleSkipWait:
			wait, err := w.Postgres.UpdateRole(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UpdateRole(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-role to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateRoleTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.RoleFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateRoleOverrides {
		fn(cmd, &updateRoleReq)
	}

	return cmd
}

// start update-snapshot-schedule command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSnapshotScheduleOverrides []func(
	*cobra.Command,
	*postgres.UpdateSnapshotScheduleRequest,
)

func newUpdateSnapshotSchedule() *cobra.Command {
	cmd := &cobra.Command{}
	var updateSnapshotScheduleReq postgres.UpdateSnapshotScheduleRequest
	updateSnapshotScheduleReq.SnapshotSchedule = &postgres.SnapshotSchedule{}

	var updateSnapshotScheduleJson flags.JsonFlag

	var updateSnapshotScheduleSkipWait bool
	var updateSnapshotScheduleTimeout time.Duration

	cmd.Flags().BoolVar(&updateSnapshotScheduleSkipWait, "no-wait", updateSnapshotScheduleSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateSnapshotScheduleTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateSnapshotScheduleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateSnapshotScheduleReq.SnapshotSchedule.Name, "name", `The resource name of the branch's snapshot schedule.`)
	// TODO: array: schedule

	cmd.Use = "update-snapshot-schedule NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update a Snapshot Schedule.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a Snapshot Schedule.

  Sets the snapshot schedule for a branch. The schedule field is replaced
  wholesale; an empty schedule disables automatic snapshots.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the branch's snapshot schedule. Format:
      projects/{project_id}/branches/{branch_id}/snapshot-schedule
    UPDATE_MASK: Fields to update. The only updatable path is schedule, which replaces
      the entire set of cadences.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
			diags := updateSnapshotScheduleJson.Unmarshal(&updateSnapshotScheduleReq.SnapshotSchedule)
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
		updateSnapshotScheduleReq.SnapshotSchedule.Name = new(args[0])
		if args[1] != "" {
			updateSnapshotScheduleReq.UpdateMask = new(args[1])
		}

		sdkReq, err := postgres.UpdateSnapshotScheduleRequestToSDK(&updateSnapshotScheduleReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateSnapshotScheduleSkipWait:
			wait, err := w.Postgres.UpdateSnapshotSchedule(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Postgres.UpdateSnapshotSchedule(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-snapshot-schedule to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateSnapshotScheduleTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := postgres.SnapshotScheduleFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSnapshotScheduleOverrides {
		fn(cmd, &updateSnapshotScheduleReq)
	}

	return cmd
}

// end service Postgres
