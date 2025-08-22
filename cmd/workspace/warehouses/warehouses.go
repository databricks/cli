// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package warehouses

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "warehouses",
		Short: `A SQL warehouse is a compute resource that lets you run SQL commands on data objects within Databricks SQL.`,
		Long: `A SQL warehouse is a compute resource that lets you run SQL commands on data
  objects within Databricks SQL. Compute resources are infrastructure resources
  that provide processing capabilities in the cloud.`,
		GroupID: "sql",
		Annotations: map[string]string{
			"package": "sql",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newEdit())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetWorkspaceWarehouseConfig())
	cmd.AddCommand(newList())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newSetWorkspaceWarehouseConfig())
	cmd.AddCommand(newStart())
	cmd.AddCommand(newStop())
	cmd.AddCommand(newUpdatePermissions())

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
	*sql.CreateWarehouseRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq sql.CreateWarehouseRequest
	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&createReq.AutoStopMins, "auto-stop-mins", createReq.AutoStopMins, `The amount of time in minutes that a SQL warehouse must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	cmd.Flags().StringVar(&createReq.ClusterSize, "cluster-size", createReq.ClusterSize, `Size of the clusters allocated for this warehouse.`)
	cmd.Flags().StringVar(&createReq.CreatorName, "creator-name", createReq.CreatorName, `warehouse creator name.`)
	cmd.Flags().BoolVar(&createReq.EnablePhoton, "enable-photon", createReq.EnablePhoton, `Configures whether the warehouse should use Photon optimized clusters.`)
	cmd.Flags().BoolVar(&createReq.EnableServerlessCompute, "enable-serverless-compute", createReq.EnableServerlessCompute, `Configures whether the warehouse should use serverless compute.`)
	cmd.Flags().StringVar(&createReq.InstanceProfileArn, "instance-profile-arn", createReq.InstanceProfileArn, `Deprecated.`)
	cmd.Flags().IntVar(&createReq.MaxNumClusters, "max-num-clusters", createReq.MaxNumClusters, `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	cmd.Flags().IntVar(&createReq.MinNumClusters, "min-num-clusters", createReq.MinNumClusters, `Minimum number of available clusters that will be maintained for this SQL warehouse.`)
	cmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Logical name for the cluster.`)
	cmd.Flags().Var(&createReq.SpotInstancePolicy, "spot-instance-policy", `Supported values: [COST_OPTIMIZED, POLICY_UNSPECIFIED, RELIABILITY_OPTIMIZED]`)
	// TODO: complex arg: tags
	cmd.Flags().Var(&createReq.WarehouseType, "warehouse-type", `Supported values: [CLASSIC, PRO, TYPE_UNSPECIFIED]`)

	cmd.Use = "create"
	cmd.Short = `Create a warehouse.`
	cmd.Long = `Create a warehouse.
  
  Creates a new SQL warehouse.`

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
		}

		wait, err := w.Warehouses.Create(ctx, createReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *sql.GetWarehouseResponse) {
			if i.Health == nil {
				return
			}
			status := i.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Health != nil {
				statusMessage = i.Health.Summary
			}
			spinner <- statusMessage
		}).GetWithTimeout(createTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
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
	*sql.DeleteWarehouseRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq sql.DeleteWarehouseRequest

	cmd.Use = "delete ID"
	cmd.Short = `Delete a warehouse.`
	cmd.Long = `Delete a warehouse.
  
  Deletes a SQL warehouse.

  Arguments:
    ID: Required. Id of the SQL warehouse.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		deleteReq.Id = args[0]

		err = w.Warehouses.Delete(ctx, deleteReq)
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

// start edit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var editOverrides []func(
	*cobra.Command,
	*sql.EditWarehouseRequest,
)

func newEdit() *cobra.Command {
	cmd := &cobra.Command{}

	var editReq sql.EditWarehouseRequest
	var editJson flags.JsonFlag

	var editSkipWait bool
	var editTimeout time.Duration

	cmd.Flags().BoolVar(&editSkipWait, "no-wait", editSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&editTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&editReq.AutoStopMins, "auto-stop-mins", editReq.AutoStopMins, `The amount of time in minutes that a SQL warehouse must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	cmd.Flags().StringVar(&editReq.ClusterSize, "cluster-size", editReq.ClusterSize, `Size of the clusters allocated for this warehouse.`)
	cmd.Flags().StringVar(&editReq.CreatorName, "creator-name", editReq.CreatorName, `warehouse creator name.`)
	cmd.Flags().BoolVar(&editReq.EnablePhoton, "enable-photon", editReq.EnablePhoton, `Configures whether the warehouse should use Photon optimized clusters.`)
	cmd.Flags().BoolVar(&editReq.EnableServerlessCompute, "enable-serverless-compute", editReq.EnableServerlessCompute, `Configures whether the warehouse should use serverless compute.`)
	cmd.Flags().StringVar(&editReq.InstanceProfileArn, "instance-profile-arn", editReq.InstanceProfileArn, `Deprecated.`)
	cmd.Flags().IntVar(&editReq.MaxNumClusters, "max-num-clusters", editReq.MaxNumClusters, `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	cmd.Flags().IntVar(&editReq.MinNumClusters, "min-num-clusters", editReq.MinNumClusters, `Minimum number of available clusters that will be maintained for this SQL warehouse.`)
	cmd.Flags().StringVar(&editReq.Name, "name", editReq.Name, `Logical name for the cluster.`)
	cmd.Flags().Var(&editReq.SpotInstancePolicy, "spot-instance-policy", `Supported values: [COST_OPTIMIZED, POLICY_UNSPECIFIED, RELIABILITY_OPTIMIZED]`)
	// TODO: complex arg: tags
	cmd.Flags().Var(&editReq.WarehouseType, "warehouse-type", `Supported values: [CLASSIC, PRO, TYPE_UNSPECIFIED]`)

	cmd.Use = "edit ID"
	cmd.Short = `Update a warehouse.`
	cmd.Long = `Update a warehouse.
  
  Updates the configuration for a SQL warehouse.

  Arguments:
    ID: Required. Id of the warehouse to configure.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := editJson.Unmarshal(&editReq)
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
			promptSpinner <- "No ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		editReq.Id = args[0]

		wait, err := w.Warehouses.Edit(ctx, editReq)
		if err != nil {
			return err
		}
		if editSkipWait {
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *sql.GetWarehouseResponse) {
			if i.Health == nil {
				return
			}
			status := i.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Health != nil {
				statusMessage = i.Health.Summary
			}
			spinner <- statusMessage
		}).GetWithTimeout(editTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range editOverrides {
		fn(cmd, &editReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*sql.GetWarehouseRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq sql.GetWarehouseRequest

	cmd.Use = "get ID"
	cmd.Short = `Get warehouse info.`
	cmd.Long = `Get warehouse info.
  
  Gets the information for a single SQL warehouse.

  Arguments:
    ID: Required. Id of the SQL warehouse.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		getReq.Id = args[0]

		response, err := w.Warehouses.Get(ctx, getReq)
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

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*sql.GetWarehousePermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq sql.GetWarehousePermissionLevelsRequest

	cmd.Use = "get-permission-levels WAREHOUSE_ID"
	cmd.Short = `Get SQL warehouse permission levels.`
	cmd.Long = `Get SQL warehouse permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    WAREHOUSE_ID: The SQL warehouse for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The SQL warehouse for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the sql warehouse for which to get or manage permissions")
		}
		getPermissionLevelsReq.WarehouseId = args[0]

		response, err := w.Warehouses.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd, &getPermissionLevelsReq)
	}

	return cmd
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*sql.GetWarehousePermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq sql.GetWarehousePermissionsRequest

	cmd.Use = "get-permissions WAREHOUSE_ID"
	cmd.Short = `Get SQL warehouse permissions.`
	cmd.Long = `Get SQL warehouse permissions.
  
  Gets the permissions of a SQL warehouse. SQL warehouses can inherit
  permissions from their root object.

  Arguments:
    WAREHOUSE_ID: The SQL warehouse for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The SQL warehouse for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the sql warehouse for which to get or manage permissions")
		}
		getPermissionsReq.WarehouseId = args[0]

		response, err := w.Warehouses.GetPermissions(ctx, getPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd, &getPermissionsReq)
	}

	return cmd
}

// start get-workspace-warehouse-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceWarehouseConfigOverrides []func(
	*cobra.Command,
)

func newGetWorkspaceWarehouseConfig() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-workspace-warehouse-config"
	cmd.Short = `Get the workspace configuration.`
	cmd.Long = `Get the workspace configuration.
  
  Gets the workspace level configuration that is shared by all SQL warehouses in
  a workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.Warehouses.GetWorkspaceWarehouseConfig(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceWarehouseConfigOverrides {
		fn(cmd)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*sql.ListWarehousesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq sql.ListWarehousesRequest

	cmd.Flags().IntVar(&listReq.RunAsUserId, "run-as-user-id", listReq.RunAsUserId, `Service Principal which will be used to fetch the list of warehouses.`)

	cmd.Use = "list"
	cmd.Short = `List warehouses.`
	cmd.Long = `List warehouses.
  
  Lists all SQL warehouses that a user has manager permissions on.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Warehouses.List(ctx, listReq)
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

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*sql.WarehousePermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq sql.WarehousePermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions WAREHOUSE_ID"
	cmd.Short = `Set SQL warehouse permissions.`
	cmd.Long = `Set SQL warehouse permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    WAREHOUSE_ID: The SQL warehouse for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
			promptSpinner <- "No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The SQL warehouse for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the sql warehouse for which to get or manage permissions")
		}
		setPermissionsReq.WarehouseId = args[0]

		response, err := w.Warehouses.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start set-workspace-warehouse-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setWorkspaceWarehouseConfigOverrides []func(
	*cobra.Command,
	*sql.SetWorkspaceWarehouseConfigRequest,
)

func newSetWorkspaceWarehouseConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var setWorkspaceWarehouseConfigReq sql.SetWorkspaceWarehouseConfigRequest
	var setWorkspaceWarehouseConfigJson flags.JsonFlag

	cmd.Flags().Var(&setWorkspaceWarehouseConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: channel
	// TODO: complex arg: config_param
	// TODO: array: data_access_config
	// TODO: array: enabled_warehouse_types
	// TODO: complex arg: global_param
	cmd.Flags().StringVar(&setWorkspaceWarehouseConfigReq.GoogleServiceAccount, "google-service-account", setWorkspaceWarehouseConfigReq.GoogleServiceAccount, `GCP only: Google Service Account used to pass to cluster to access Google Cloud Storage.`)
	cmd.Flags().StringVar(&setWorkspaceWarehouseConfigReq.InstanceProfileArn, "instance-profile-arn", setWorkspaceWarehouseConfigReq.InstanceProfileArn, `AWS Only: Instance profile used to pass IAM role to the cluster.`)
	cmd.Flags().Var(&setWorkspaceWarehouseConfigReq.SecurityPolicy, "security-policy", `Security policy for warehouses. Supported values: [DATA_ACCESS_CONTROL, NONE, PASSTHROUGH]`)
	// TODO: complex arg: sql_configuration_parameters

	cmd.Use = "set-workspace-warehouse-config"
	cmd.Short = `Set the workspace configuration.`
	cmd.Long = `Set the workspace configuration.
  
  Sets the workspace level configuration that is shared by all SQL warehouses in
  a workspace.`

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
			diags := setWorkspaceWarehouseConfigJson.Unmarshal(&setWorkspaceWarehouseConfigReq)
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

		err = w.Warehouses.SetWorkspaceWarehouseConfig(ctx, setWorkspaceWarehouseConfigReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setWorkspaceWarehouseConfigOverrides {
		fn(cmd, &setWorkspaceWarehouseConfigReq)
	}

	return cmd
}

// start start command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startOverrides []func(
	*cobra.Command,
	*sql.StartRequest,
)

func newStart() *cobra.Command {
	cmd := &cobra.Command{}

	var startReq sql.StartRequest

	var startSkipWait bool
	var startTimeout time.Duration

	cmd.Flags().BoolVar(&startSkipWait, "no-wait", startSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&startTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Use = "start ID"
	cmd.Short = `Start a warehouse.`
	cmd.Long = `Start a warehouse.
  
  Starts a SQL warehouse.

  Arguments:
    ID: Required. Id of the SQL warehouse.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		startReq.Id = args[0]

		wait, err := w.Warehouses.Start(ctx, startReq)
		if err != nil {
			return err
		}
		if startSkipWait {
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *sql.GetWarehouseResponse) {
			if i.Health == nil {
				return
			}
			status := i.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Health != nil {
				statusMessage = i.Health.Summary
			}
			spinner <- statusMessage
		}).GetWithTimeout(startTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range startOverrides {
		fn(cmd, &startReq)
	}

	return cmd
}

// start stop command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var stopOverrides []func(
	*cobra.Command,
	*sql.StopRequest,
)

func newStop() *cobra.Command {
	cmd := &cobra.Command{}

	var stopReq sql.StopRequest

	var stopSkipWait bool
	var stopTimeout time.Duration

	cmd.Flags().BoolVar(&stopSkipWait, "no-wait", stopSkipWait, `do not wait to reach STOPPED state`)
	cmd.Flags().DurationVar(&stopTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach STOPPED state`)

	cmd.Use = "stop ID"
	cmd.Short = `Stop a warehouse.`
	cmd.Long = `Stop a warehouse.
  
  Stops a SQL warehouse.

  Arguments:
    ID: Required. Id of the SQL warehouse.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "Required")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have required")
		}
		stopReq.Id = args[0]

		wait, err := w.Warehouses.Stop(ctx, stopReq)
		if err != nil {
			return err
		}
		if stopSkipWait {
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *sql.GetWarehouseResponse) {
			if i.Health == nil {
				return
			}
			status := i.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Health != nil {
				statusMessage = i.Health.Summary
			}
			spinner <- statusMessage
		}).GetWithTimeout(stopTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range stopOverrides {
		fn(cmd, &stopReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*sql.WarehousePermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq sql.WarehousePermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions WAREHOUSE_ID"
	cmd.Short = `Update SQL warehouse permissions.`
	cmd.Long = `Update SQL warehouse permissions.
  
  Updates the permissions on a SQL warehouse. SQL warehouses can inherit
  permissions from their root object.

  Arguments:
    WAREHOUSE_ID: The SQL warehouse for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
			promptSpinner <- "No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down."
			names, err := w.Warehouses.EndpointInfoNameToIdMap(ctx, sql.ListWarehousesRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Warehouses drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The SQL warehouse for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the sql warehouse for which to get or manage permissions")
		}
		updatePermissionsReq.WarehouseId = args[0]

		response, err := w.Warehouses.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service Warehouses
