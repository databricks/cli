// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package warehouses

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

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	warehouses "github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkwarehouses "github.com/databricks/sdk-go/warehouses/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateDefaultWarehouseOverride())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newDeleteDefaultWarehouseOverride())
	cmd.AddCommand(newEdit())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetDefaultWarehouseOverride())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetWorkspaceWarehouseConfig())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListDefaultWarehouseOverrides())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newSetWorkspaceWarehouseConfig())
	cmd.AddCommand(newStart())
	cmd.AddCommand(newStop())
	cmd.AddCommand(newUpdateDefaultWarehouseOverride())
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
	*warehouses.CreateWarehouseRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq warehouses.CreateWarehouseRequest

	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.IntPtrVar(cmd.Flags(), &createReq.AutoStopMins, "auto-stop-mins", `The amount of time in minutes that a SQL warehouse must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	flags.StringPtrVar(cmd.Flags(), &createReq.ClusterSize, "cluster-size", `Size of the clusters allocated for this warehouse.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.CreatorName, "creator-name", `warehouse creator name.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.EnablePhoton, "enable-photon", `Configures whether the warehouse should use Photon optimized clusters.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.EnableServerlessCompute, "enable-serverless-compute", `Configures whether the warehouse should use serverless compute.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.InstanceProfileArn, "instance-profile-arn", `Deprecated.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.MaxNumClusters, "max-num-clusters", `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.MinNumClusters, "min-num-clusters", `Minimum number of available clusters that will be maintained for this SQL warehouse.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Name, "name", `Logical name for the cluster.`)
	flags.EnumVar(cmd.Flags(), &createReq.SpotInstancePolicy, "spot-instance-policy", `Configurations whether the endpoint should use spot instances. Supported values: [COST_OPTIMIZED, POLICY_UNSPECIFIED, RELIABILITY_OPTIMIZED]`, "SpotInstancePolicy", "COST_OPTIMIZED", "POLICY_UNSPECIFIED", "RELIABILITY_OPTIMIZED")
	flags.IntPtrVar(cmd.Flags(), &createReq.StatementTimeout, "statement-timeout", `Warehouse statement timeout in seconds.`)
	// TODO: complex arg: tags
	flags.EnumVar(cmd.Flags(), &createReq.WarehouseType, "warehouse-type", `Warehouse type: PRO or CLASSIC. Supported values: [CLASSIC, PRO, TYPE_UNSPECIFIED]`, "CreateWarehouseRequestWarehouseType", "CLASSIC", "PRO", "TYPE_UNSPECIFIED")

	cmd.Use = "create"
	cmd.Short = `Create a warehouse.`
	cmd.Long = `Create a warehouse.

  Creates a new SQL warehouse.`

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

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		sdkReq, err := warehouses.CreateWarehouseRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		wait, err := w.Warehouses.CreateWarehouse(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, map[string]any{"id": wait.GetId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := warehouses.GetWarehouseResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start create-default-warehouse-override command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDefaultWarehouseOverrideOverrides []func(
	*cobra.Command,
	*warehouses.CreateDefaultWarehouseOverrideRequest,
)

func newCreateDefaultWarehouseOverride() *cobra.Command {
	cmd := &cobra.Command{}
	var createDefaultWarehouseOverrideReq warehouses.CreateDefaultWarehouseOverrideRequest
	createDefaultWarehouseOverrideReq.DefaultWarehouseOverride = &warehouses.DefaultWarehouseOverride{}

	var createDefaultWarehouseOverrideJson flags.JsonFlag

	cmd.Flags().Var(&createDefaultWarehouseOverrideJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createDefaultWarehouseOverrideReq.DefaultWarehouseOverride.Name, "name", `The resource name of the default warehouse override.`)
	flags.StringPtrVar(cmd.Flags(), &createDefaultWarehouseOverrideReq.DefaultWarehouseOverride.WarehouseId, "warehouse-id", `The specific warehouse ID when type is CUSTOM.`)

	cmd.Use = "create-default-warehouse-override DEFAULT_WAREHOUSE_OVERRIDE_ID TYPE"
	cmd.Short = `*Beta* Create default warehouse override.`
	cmd.Long = `This command is in Beta and may change without notice.

Create default warehouse override.

  Creates a new default warehouse override for a user. Users can create their
  own override. Admins can create overrides for any user.

  Arguments:
    DEFAULT_WAREHOUSE_OVERRIDE_ID: Required. The ID to use for the override, which will become the final
      component of the override's resource name. Can be a numeric user ID or the
      literal string "me" for the current user.
    TYPE: The type of override behavior.
      Supported values: [CUSTOM, LAST_SELECTED]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only DEFAULT_WAREHOUSE_OVERRIDE_ID as positional arguments. Provide 'type' in your JSON input")
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
			diags := createDefaultWarehouseOverrideJson.Unmarshal(&createDefaultWarehouseOverrideReq.DefaultWarehouseOverride)
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
		createDefaultWarehouseOverrideReq.DefaultWarehouseOverrideId = new(args[0])
		if !cmd.Flags().Changed("json") {
			createDefaultWarehouseOverrideReq.DefaultWarehouseOverride.Type = warehouses.DefaultWarehouseOverrideType(args[1])
		}

		sdkReq, err := warehouses.CreateDefaultWarehouseOverrideRequestToSDK(&createDefaultWarehouseOverrideReq)
		if err != nil {
			return err
		}
		response, err := w.Warehouses.CreateDefaultWarehouseOverride(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := warehouses.DefaultWarehouseOverrideFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDefaultWarehouseOverrideOverrides {
		fn(cmd, &createDefaultWarehouseOverrideReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*sdkwarehouses.DeleteWarehouseRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkwarehouses.DeleteWarehouseRequest

	cmd.Use = "delete ID"
	cmd.Short = `Delete a warehouse.`
	cmd.Long = `Delete a warehouse.

  Deletes a SQL warehouse.

  Arguments:
    ID: Required. Id of the SQL warehouse.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have required")
		}
		deleteReq.Id = new(args[0])

		sdkReq := &deleteReq
		response, err := w.Warehouses.DeleteWarehouse(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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

// start delete-default-warehouse-override command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDefaultWarehouseOverrideOverrides []func(
	*cobra.Command,
	*sdkwarehouses.DeleteDefaultWarehouseOverrideRequest,
)

func newDeleteDefaultWarehouseOverride() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteDefaultWarehouseOverrideReq sdkwarehouses.DeleteDefaultWarehouseOverrideRequest

	cmd.Use = "delete-default-warehouse-override NAME"
	cmd.Short = `*Beta* Delete default warehouse override.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete default warehouse override.

  Deletes the default warehouse override for a user. Users can delete their own
  override. Admins can delete overrides for any user. After deletion, the
  workspace default warehouse will be used.

  Arguments:
    NAME: Required. The resource name of the default warehouse override to delete.
      Format: default-warehouse-overrides/{default_warehouse_override_id} The
      default_warehouse_override_id can be a numeric user ID or the literal
      string "me" for the current user.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have required")
		}
		deleteDefaultWarehouseOverrideReq.Name = new(args[0])

		sdkReq := &deleteDefaultWarehouseOverrideReq
		err = w.Warehouses.DeleteDefaultWarehouseOverride(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDefaultWarehouseOverrideOverrides {
		fn(cmd, &deleteDefaultWarehouseOverrideReq)
	}

	return cmd
}

// start edit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var editOverrides []func(
	*cobra.Command,
	*warehouses.EditWarehouseRequest,
)

func newEdit() *cobra.Command {
	cmd := &cobra.Command{}
	var editReq warehouses.EditWarehouseRequest

	var editJson flags.JsonFlag

	var editSkipWait bool
	var editTimeout time.Duration

	cmd.Flags().BoolVar(&editSkipWait, "no-wait", editSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&editTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.IntPtrVar(cmd.Flags(), &editReq.AutoStopMins, "auto-stop-mins", `The amount of time in minutes that a SQL warehouse must be idle (i.e., no RUNNING queries) before it is automatically stopped.`)
	// TODO: complex arg: channel
	flags.StringPtrVar(cmd.Flags(), &editReq.ClusterSize, "cluster-size", `Size of the clusters allocated for this warehouse.`)
	flags.StringPtrVar(cmd.Flags(), &editReq.CreatorName, "creator-name", `warehouse creator name.`)
	flags.BoolPtrVar(cmd.Flags(), &editReq.EnablePhoton, "enable-photon", `Configures whether the warehouse should use Photon optimized clusters.`)
	flags.BoolPtrVar(cmd.Flags(), &editReq.EnableServerlessCompute, "enable-serverless-compute", `Configures whether the warehouse should use serverless compute.`)
	flags.StringPtrVar(cmd.Flags(), &editReq.InstanceProfileArn, "instance-profile-arn", `Deprecated.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.MaxNumClusters, "max-num-clusters", `Maximum number of clusters that the autoscaler will create to handle concurrent queries.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.MinNumClusters, "min-num-clusters", `Minimum number of available clusters that will be maintained for this SQL warehouse.`)
	flags.StringPtrVar(cmd.Flags(), &editReq.Name, "name", `Logical name for the cluster.`)
	flags.EnumVar(cmd.Flags(), &editReq.SpotInstancePolicy, "spot-instance-policy", `Configurations whether the endpoint should use spot instances. Supported values: [COST_OPTIMIZED, POLICY_UNSPECIFIED, RELIABILITY_OPTIMIZED]`, "SpotInstancePolicy", "COST_OPTIMIZED", "POLICY_UNSPECIFIED", "RELIABILITY_OPTIMIZED")
	flags.IntPtrVar(cmd.Flags(), &editReq.StatementTimeout, "statement-timeout", `Warehouse statement timeout in seconds.`)
	// TODO: complex arg: tags
	flags.EnumVar(cmd.Flags(), &editReq.WarehouseType, "warehouse-type", `Warehouse type: PRO or CLASSIC. Supported values: [CLASSIC, PRO, TYPE_UNSPECIFIED]`, "EditWarehouseRequestWarehouseType", "CLASSIC", "PRO", "TYPE_UNSPECIFIED")

	cmd.Use = "edit ID"
	cmd.Short = `Update a warehouse.`
	cmd.Long = `Update a warehouse.

  Updates the configuration for a SQL warehouse.

  Arguments:
    ID: Required. Id of the warehouse to configure.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := editJson.Unmarshal(&editReq)
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
		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have required")
		}
		editReq.Id = new(args[0])

		sdkReq, err := warehouses.EditWarehouseRequestToSDK(&editReq)
		if err != nil {
			return err
		}
		wait, err := w.Warehouses.EditWarehouse(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if editSkipWait {
			return cmdio.Render(ctx, map[string]any{"id": wait.GetId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for edit to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(editTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := warehouses.GetWarehouseResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkwarehouses.GetWarehouseRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkwarehouses.GetWarehouseRequest

	cmd.Use = "get ID"
	cmd.Short = `Get warehouse info.`
	cmd.Long = `Get warehouse info.

  Gets the information for a single SQL warehouse.

  Arguments:
    ID: Required. Id of the SQL warehouse.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have required")
		}
		getReq.Id = new(args[0])

		sdkReq := &getReq
		response, err := w.Warehouses.GetWarehouse(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := warehouses.GetWarehouseResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start get-default-warehouse-override command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDefaultWarehouseOverrideOverrides []func(
	*cobra.Command,
	*sdkwarehouses.GetDefaultWarehouseOverrideRequest,
)

func newGetDefaultWarehouseOverride() *cobra.Command {
	cmd := &cobra.Command{}
	var getDefaultWarehouseOverrideReq sdkwarehouses.GetDefaultWarehouseOverrideRequest

	cmd.Use = "get-default-warehouse-override NAME"
	cmd.Short = `*Beta* Get default warehouse override.`
	cmd.Long = `This command is in Beta and may change without notice.

Get default warehouse override.

  Returns the default warehouse override for a user. Users can fetch their own
  override. Admins can fetch overrides for any user. If no override exists, the
  UI will fallback to the workspace default warehouse.

  Arguments:
    NAME: Required. The resource name of the default warehouse override to retrieve.
      Format: default-warehouse-overrides/{default_warehouse_override_id} The
      default_warehouse_override_id can be a numeric user ID or the literal
      string "me" for the current user.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No NAME argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have required")
		}
		getDefaultWarehouseOverrideReq.Name = new(args[0])

		sdkReq := &getDefaultWarehouseOverrideReq
		response, err := w.Warehouses.GetDefaultWarehouseOverride(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := warehouses.DefaultWarehouseOverrideFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDefaultWarehouseOverrideOverrides {
		fn(cmd, &getDefaultWarehouseOverrideReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*sdkaccessmanagement.ListPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionLevelsReq sdkaccessmanagement.ListPermissionLevelsRequest

	cmd.Use = "get-permission-levels WAREHOUSE_ID"
	cmd.Short = `Get SQL warehouse permission levels.`
	cmd.Long = `Get SQL warehouse permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    WAREHOUSE_ID: The SQL warehouse for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have the sql warehouse for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: warehouse_id

		getPermissionLevelsReq.RequestObjectType = new("warehouses")

		sdkReq := &getPermissionLevelsReq
		response, err := w.AccessManagement.ListPermissionLevels(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListPermissionLevelsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.GetObjectPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionsReq sdkaccessmanagement.GetObjectPermissionsRequest

	cmd.Use = "get-permissions WAREHOUSE_ID"
	cmd.Short = `Get SQL warehouse permissions.`
	cmd.Long = `Get SQL warehouse permissions.

  Gets the permissions of a SQL warehouse. SQL warehouses can inherit
  permissions from their root object.

  Arguments:
    WAREHOUSE_ID: The SQL warehouse for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have the sql warehouse for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: warehouse_id

		getPermissionsReq.RequestObjectType = new("warehouses")

		sdkReq := &getPermissionsReq
		response, err := w.AccessManagement.GetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkwarehouses.GetWorkspaceWarehouseConfigRequest{}
		response, err := w.Warehouses.GetWorkspaceWarehouseConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := warehouses.GetWorkspaceWarehouseConfigResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*warehouses.ListWarehousesRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq warehouses.ListWarehousesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `The max number of warehouses to return.`)
	flags.Int64PtrVar(cmd.Flags(), &listReq.RunAsUserId, "run-as-user-id", `Deprecated: this field is ignored by the server.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List warehouses.`
	cmd.Long = `List warehouses.

  Lists all SQL warehouses that a user has access to.`

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

		sdkReq, err := warehouses.ListWarehousesRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Warehouses.ListWarehousesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, warehouses.EndpointInfoFromSDK))
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

// start list-default-warehouse-overrides command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDefaultWarehouseOverridesOverrides []func(
	*cobra.Command,
	*warehouses.ListDefaultWarehouseOverridesRequest,
)

func newListDefaultWarehouseOverrides() *cobra.Command {
	cmd := &cobra.Command{}
	var listDefaultWarehouseOverridesReq warehouses.ListDefaultWarehouseOverridesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDefaultWarehouseOverridesLimit int

	flags.IntPtrVar(cmd.Flags(), &listDefaultWarehouseOverridesReq.PageSize, "page-size", `The maximum number of overrides to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDefaultWarehouseOverridesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listDefaultWarehouseOverridesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-default-warehouse-overrides"
	cmd.Short = `*Beta* List default warehouse overrides.`
	cmd.Long = `This command is in Beta and may change without notice.

List default warehouse overrides.

  Lists all default warehouse overrides in the workspace. Only workspace
  administrators can list all overrides.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := warehouses.ListDefaultWarehouseOverridesRequestToSDK(&listDefaultWarehouseOverridesReq)
		if err != nil {
			return err
		}
		if listDefaultWarehouseOverridesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDefaultWarehouseOverridesLimit)
		}
		if listDefaultWarehouseOverridesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDefaultWarehouseOverridesLimit)
		}
		response := w.Warehouses.ListDefaultWarehouseOverridesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, warehouses.DefaultWarehouseOverrideFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDefaultWarehouseOverridesOverrides {
		fn(cmd, &listDefaultWarehouseOverridesReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.SetObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var setPermissionsReq accessmanagement.SetObjectPermissionsRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have the sql warehouse for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: warehouse_id

		setPermissionsReq.RequestObjectType = new("warehouses")

		sdkReq, err := accessmanagement.SetObjectPermissionsRequestToSDK(&setPermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.SetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*warehouses.SetWorkspaceWarehouseConfigRequest,
)

func newSetWorkspaceWarehouseConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var setWorkspaceWarehouseConfigReq warehouses.SetWorkspaceWarehouseConfigRequest

	var setWorkspaceWarehouseConfigJson flags.JsonFlag

	cmd.Flags().Var(&setWorkspaceWarehouseConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: channel
	// TODO: complex arg: config_param
	// TODO: array: data_access_config
	flags.BoolPtrVar(cmd.Flags(), &setWorkspaceWarehouseConfigReq.EnableServerlessCompute, "enable-serverless-compute", `Deprecated: only setting this to true is allowed.`)
	// TODO: array: enabled_warehouse_types
	// TODO: complex arg: global_param
	flags.StringPtrVar(cmd.Flags(), &setWorkspaceWarehouseConfigReq.GoogleServiceAccount, "google-service-account", `GCP only: Google Service Account used to pass to cluster to access Google Cloud Storage.`)
	flags.StringPtrVar(cmd.Flags(), &setWorkspaceWarehouseConfigReq.InstanceProfileArn, "instance-profile-arn", `AWS Only: The instance profile used to pass an IAM role to the SQL warehouses.`)
	flags.EnumVar(cmd.Flags(), &setWorkspaceWarehouseConfigReq.SecurityPolicy, "security-policy", `Security policy for warehouses. Supported values: [DATA_ACCESS_CONTROL, NONE, PASSTHROUGH]`, "SetWorkspaceWarehouseConfigRequestSecurityPolicy", "DATA_ACCESS_CONTROL", "NONE", "PASSTHROUGH")
	// TODO: complex arg: sql_configuration_parameters

	cmd.Use = "set-workspace-warehouse-config"
	cmd.Short = `Set the workspace configuration.`
	cmd.Long = `Set the workspace configuration.

  Sets the workspace level configuration that is shared by all SQL warehouses in
  a workspace.`

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

		if cmd.Flags().Changed("json") {
			diags := setWorkspaceWarehouseConfigJson.Unmarshal(&setWorkspaceWarehouseConfigReq)
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

		sdkReq, err := warehouses.SetWorkspaceWarehouseConfigRequestToSDK(&setWorkspaceWarehouseConfigReq)
		if err != nil {
			return err
		}
		response, err := w.Warehouses.SetWorkspaceWarehouseConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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
	*warehouses.StartRequest,
)

func newStart() *cobra.Command {
	cmd := &cobra.Command{}
	var startReq warehouses.StartRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have required")
		}
		startReq.Id = new(args[0])

		sdkReq, err := warehouses.StartRequestToSDK(&startReq)
		if err != nil {
			return err
		}
		wait, err := w.Warehouses.StartWarehouse(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if startSkipWait {
			return cmdio.Render(ctx, map[string]any{"id": wait.GetId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for start to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(startTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := warehouses.GetWarehouseResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*warehouses.StopRequest,
)

func newStop() *cobra.Command {
	cmd := &cobra.Command{}
	var stopReq warehouses.StopRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have required")
		}
		stopReq.Id = new(args[0])

		sdkReq, err := warehouses.StopRequestToSDK(&stopReq)
		if err != nil {
			return err
		}
		wait, err := w.Warehouses.StopWarehouse(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if stopSkipWait {
			return cmdio.Render(ctx, map[string]any{"id": wait.GetId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for stop to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(stopTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := warehouses.GetWarehouseResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start update-default-warehouse-override command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDefaultWarehouseOverrideOverrides []func(
	*cobra.Command,
	*warehouses.UpdateDefaultWarehouseOverrideRequest,
)

func newUpdateDefaultWarehouseOverride() *cobra.Command {
	cmd := &cobra.Command{}
	var updateDefaultWarehouseOverrideReq warehouses.UpdateDefaultWarehouseOverrideRequest
	updateDefaultWarehouseOverrideReq.DefaultWarehouseOverride = &warehouses.DefaultWarehouseOverride{}

	var updateDefaultWarehouseOverrideJson flags.JsonFlag

	cmd.Flags().Var(&updateDefaultWarehouseOverrideJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &updateDefaultWarehouseOverrideReq.AllowMissing, "allow-missing", `If set to true, and the override is not found, a new override will be created.`)
	flags.StringPtrVar(cmd.Flags(), &updateDefaultWarehouseOverrideReq.DefaultWarehouseOverride.Name, "name", `The resource name of the default warehouse override.`)
	flags.StringPtrVar(cmd.Flags(), &updateDefaultWarehouseOverrideReq.DefaultWarehouseOverride.WarehouseId, "warehouse-id", `The specific warehouse ID when type is CUSTOM.`)

	cmd.Use = "update-default-warehouse-override NAME UPDATE_MASK TYPE"
	cmd.Short = `*Beta* Update default warehouse override.`
	cmd.Long = `This command is in Beta and may change without notice.

Update default warehouse override.

  Updates an existing default warehouse override for a user. Users can update
  their own override. Admins can update overrides for any user.

  Arguments:
    NAME: The resource name of the default warehouse override. Format:
      default-warehouse-overrides/{default_warehouse_override_id}
    UPDATE_MASK: Required. Field mask specifying which fields to update. Only the fields
      specified in the mask will be updated. Use "*" to update all fields. When
      allow_missing is true, this field is ignored and all fields are applied.
    TYPE: The type of override behavior.
      Supported values: [CUSTOM, LAST_SELECTED]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'type' in your JSON input")
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
			diags := updateDefaultWarehouseOverrideJson.Unmarshal(&updateDefaultWarehouseOverrideReq.DefaultWarehouseOverride)
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
		updateDefaultWarehouseOverrideReq.DefaultWarehouseOverride.Name = new(args[0])
		if args[1] != "" {
			updateDefaultWarehouseOverrideReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateDefaultWarehouseOverrideReq.DefaultWarehouseOverride.Type = warehouses.DefaultWarehouseOverrideType(args[2])
		}

		sdkReq, err := warehouses.UpdateDefaultWarehouseOverrideRequestToSDK(&updateDefaultWarehouseOverrideReq)
		if err != nil {
			return err
		}
		response, err := w.Warehouses.UpdateDefaultWarehouseOverride(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := warehouses.DefaultWarehouseOverrideFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDefaultWarehouseOverrideOverrides {
		fn(cmd, &updateDefaultWarehouseOverrideReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.UpdateObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq accessmanagement.UpdateObjectPermissionsRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No WAREHOUSE_ID argument specified. Loading names for Warehouses drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Warehouses.ListWarehousesIter(ctx, sdkwarehouses.ListWarehousesRequest{}), []string{"Name"}, []string{"Id"})
			sp.Close()
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
			return errors.New("expected to have the sql warehouse for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: warehouse_id

		updatePermissionsReq.RequestObjectType = new("warehouses")

		sdkReq, err := accessmanagement.UpdateObjectPermissionsRequestToSDK(&updatePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.UpdateObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
