// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package instance_pools

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	instancepools "github.com/databricks/cli/libs/sdktypes/instancepools/v2"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkinstancepools "github.com/databricks/sdk-go/instancepools/v2"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance-pools",
		Short: `Instance Pools API are used to create, edit, delete and list instance pools by using ready-to-use cloud instances which reduces a cluster start and auto-scaling times.`,
		Long: `Instance Pools API are used to create, edit, delete and list instance pools by
  using ready-to-use cloud instances which reduces a cluster start and
  auto-scaling times.

  Databricks pools reduce cluster start and auto-scaling times by maintaining a
  set of idle, ready-to-use instances. When a cluster is attached to a pool,
  cluster nodes are created using the pool’s idle instances. If the pool has
  no idle instances, the pool expands by allocating a new instance from the
  instance provider in order to accommodate the cluster’s request. When a
  cluster releases an instance, it returns to the pool and is free for another
  cluster to use. Only clusters attached to a pool can use that pool’s idle
  instances.

  You can specify a different pool for the driver node and worker nodes, or use
  the same pool for both.

  Databricks does not charge DBUs while instances are idle in the pool. Instance
  provider billing does apply. See pricing.`,
		GroupID: "compute",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newEdit())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newList())
	cmd.AddCommand(newSetPermissions())
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
	*instancepools.CreateInstancePoolRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq instancepools.CreateInstancePoolRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: map via StringToStringVar: custom_tags
	// TODO: complex arg: disk_spec
	flags.BoolPtrVar(cmd.Flags(), &createReq.EnableElasticDisk, "enable-elastic-disk", `Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	// TODO: complex arg: gcp_attributes
	flags.IntPtrVar(cmd.Flags(), &createReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.MaxCapacity, "max-capacity", `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.MinIdleInstances, "min-idle-instances", `Minimum number of idle instances to keep in the instance pool.`)
	// TODO: complex arg: node_type_flexibility
	// TODO: array: preloaded_docker_images
	// TODO: array: preloaded_spark_versions
	flags.IntPtrVar(cmd.Flags(), &createReq.RemoteDiskThroughput, "remote-disk-throughput", `If set, what the configurable throughput (in Mb/s) for the remote disk is.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.TotalInitialRemoteDiskSize, "total-initial-remote-disk-size", `If set, what the total initial volume size (in GB) of the remote disks should be.`)

	cmd.Use = "create INSTANCE_POOL_NAME NODE_TYPE_ID"
	cmd.Short = `Create a new instance pool.`
	cmd.Long = `Create a new instance pool.

  Creates a new instance pool using idle and ready-to-use cloud instances.

  Arguments:
    INSTANCE_POOL_NAME: Pool name requested by the user. Pool name must be unique. Length must be
      between 1 and 100 characters.
    NODE_TYPE_ID: This field encodes, through a single value, the resources available to
      each of the Spark nodes in this cluster. For example, the Spark nodes can
      be provisioned and optimized for memory or compute intensive workloads. A
      list of available node types can be retrieved by using the
      [clusters/listNodeTypes] API call.

      [clusters/listNodeTypes]: https://docs.databricks.com/api/workspace/clusters/listnodetypes`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'instance_pool_name', 'node_type_id' in your JSON input")
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
		if !cmd.Flags().Changed("json") {
			createReq.InstancePoolName = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createReq.NodeTypeId = new(args[1])
		}

		sdkReq, err := instancepools.CreateInstancePoolRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.InstancePools.CreateInstancePool(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := instancepools.CreateInstancePoolResponseFromSDK(response)
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*instancepools.DeleteInstancePoolRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq instancepools.DeleteInstancePoolRequest

	var deleteJson flags.JsonFlag

	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete INSTANCE_POOL_ID"
	cmd.Short = `Delete an instance pool.`
	cmd.Long = `Delete an instance pool.

  Deletes the instance pool permanently. The idle instances in the pool are
  terminated asynchronously.

  Arguments:
    INSTANCE_POOL_ID: The instance pool to be terminated.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'instance_pool_id' in your JSON input")
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
			diags := deleteJson.Unmarshal(&deleteReq)
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
			deleteReq.InstancePoolId = new(args[0])
		}

		sdkReq, err := instancepools.DeleteInstancePoolRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.InstancePools.DeleteInstancePool(ctx, *sdkReq)
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

// start edit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var editOverrides []func(
	*cobra.Command,
	*instancepools.EditInstancePoolRequest,
)

func newEdit() *cobra.Command {
	cmd := &cobra.Command{}
	var editReq instancepools.EditInstancePoolRequest

	var editJson flags.JsonFlag

	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: custom_tags
	flags.IntPtrVar(cmd.Flags(), &editReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.MaxCapacity, "max-capacity", `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.MinIdleInstances, "min-idle-instances", `Minimum number of idle instances to keep in the instance pool.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.RemoteDiskThroughput, "remote-disk-throughput", `If set, what the configurable throughput (in Mb/s) for the remote disk is.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.TotalInitialRemoteDiskSize, "total-initial-remote-disk-size", `If set, what the total initial volume size (in GB) of the remote disks should be.`)

	cmd.Use = "edit INSTANCE_POOL_ID INSTANCE_POOL_NAME NODE_TYPE_ID"
	cmd.Short = `Edit an existing instance pool.`
	cmd.Long = `Edit an existing instance pool.

  Modifies the configuration of an existing instance pool.

  Arguments:
    INSTANCE_POOL_ID: Instance pool ID
    INSTANCE_POOL_NAME: Pool name requested by the user. Pool name must be unique. Length must be
      between 1 and 100 characters.
    NODE_TYPE_ID: This field encodes, through a single value, the resources available to
      each of the Spark nodes in this cluster. For example, the Spark nodes can
      be provisioned and optimized for memory or compute intensive workloads. A
      list of available node types can be retrieved by using the
      [clusters/listNodeTypes] API call.

      [clusters/listNodeTypes]: https://docs.databricks.com/api/workspace/clusters/listnodetypes`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'instance_pool_id', 'instance_pool_name', 'node_type_id' in your JSON input")
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
		if !cmd.Flags().Changed("json") {
			editReq.InstancePoolId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			editReq.InstancePoolName = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			editReq.NodeTypeId = new(args[2])
		}

		sdkReq, err := instancepools.EditInstancePoolRequestToSDK(&editReq)
		if err != nil {
			return err
		}
		response, err := w.InstancePools.EditInstancePool(ctx, *sdkReq)
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
	*instancepools.GetInstancePoolRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq instancepools.GetInstancePoolRequest

	cmd.Use = "get INSTANCE_POOL_ID"
	cmd.Short = `Get instance pool information.`
	cmd.Long = `Get instance pool information.

  Retrieve the information for an instance pool based on its identifier.

  Arguments:
    INSTANCE_POOL_ID: The canonical unique identifier for the instance pool.`

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

		getReq.InstancePoolId = new(args[0])

		sdkReq, err := instancepools.GetInstancePoolRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.InstancePools.GetInstancePool(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := instancepools.GetInstancePoolResponseFromSDK(response)
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

	cmd.Use = "get-permission-levels INSTANCE_POOL_ID"
	cmd.Short = `Get instance pool permission levels.`
	cmd.Long = `Get instance pool permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    INSTANCE_POOL_ID: The instance pool for which to get or manage permissions.`

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

		getPermissionLevelsReq.RequestObjectId = new(args[0])

		getPermissionLevelsReq.RequestObjectType = new("instance-pools")

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

	cmd.Use = "get-permissions INSTANCE_POOL_ID"
	cmd.Short = `Get instance pool permissions.`
	cmd.Long = `Get instance pool permissions.

  Gets the permissions of an instance pool. Instance pools can inherit
  permissions from their root object.

  Arguments:
    INSTANCE_POOL_ID: The instance pool for which to get or manage permissions.`

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

		getPermissionsReq.RequestObjectId = new(args[0])

		getPermissionsReq.RequestObjectType = new("instance-pools")

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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list"
	cmd.Short = `List instance pool info.`
	cmd.Long = `List instance pool info.

  Gets a list of instance pools with their statistics.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkinstancepools.ListInstancePoolsRequest{}

		response, err := w.InstancePools.ListInstancePools(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}

		out, err := instancepools.ListInstancePoolsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out.InstancePools)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
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

	cmd.Use = "set-permissions INSTANCE_POOL_ID"
	cmd.Short = `Set instance pool permissions.`
	cmd.Long = `Set instance pool permissions.

  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    INSTANCE_POOL_ID: The instance pool for which to get or manage permissions.`

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
		setPermissionsReq.RequestObjectId = new(args[0])

		setPermissionsReq.RequestObjectType = new("instance-pools")

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

	cmd.Use = "update-permissions INSTANCE_POOL_ID"
	cmd.Short = `Update instance pool permissions.`
	cmd.Long = `Update instance pool permissions.

  Updates the permissions on an instance pool. Instance pools can inherit
  permissions from their root object.

  Arguments:
    INSTANCE_POOL_ID: The instance pool for which to get or manage permissions.`

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
		updatePermissionsReq.RequestObjectId = new(args[0])

		updatePermissionsReq.RequestObjectType = new("instance-pools")

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

// end service InstancePools
