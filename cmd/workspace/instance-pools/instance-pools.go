// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package instance_pools

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "compute",
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
	*compute.CreateInstancePool,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq compute.CreateInstancePool
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: map via StringToStringVar: custom_tags
	// TODO: complex arg: disk_spec
	cmd.Flags().BoolVar(&createReq.EnableElasticDisk, "enable-elastic-disk", createReq.EnableElasticDisk, `Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	// TODO: complex arg: gcp_attributes
	cmd.Flags().IntVar(&createReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", createReq.IdleInstanceAutoterminationMinutes, `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	// TODO: complex arg: instance_pool_fleet_attributes
	cmd.Flags().IntVar(&createReq.MaxCapacity, "max-capacity", createReq.MaxCapacity, `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.`)
	cmd.Flags().IntVar(&createReq.MinIdleInstances, "min-idle-instances", createReq.MinIdleInstances, `Minimum number of idle instances to keep in the instance pool.`)
	// TODO: array: preloaded_docker_images
	// TODO: array: preloaded_spark_versions

	cmd.Use = "create INSTANCE_POOL_NAME NODE_TYPE_ID"
	cmd.Short = `Create a new instance pool.`
	cmd.Long = `Create a new instance pool.
  
  Creates a new instance pool using idle and ready-to-use cloud instances.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

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
			createReq.InstancePoolName = args[0]
			createReq.NodeTypeId = args[1]
		}

		response, err := w.InstancePools.Create(ctx, createReq)
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
	*compute.DeleteInstancePool,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq compute.DeleteInstancePool
	var deleteJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete INSTANCE_POOL_ID"
	cmd.Short = `Delete an instance pool.`
	cmd.Long = `Delete an instance pool.
  
  Deletes the instance pool permanently. The idle instances in the pool are
  terminated asynchronously.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No INSTANCE_POOL_ID argument specified. Loading names for Instance Pools drop-down."
				names, err := w.InstancePools.InstancePoolAndStatsInstancePoolNameToInstancePoolIdMap(ctx)
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Instance Pools drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The instance pool to be terminated")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the instance pool to be terminated")
			}
			deleteReq.InstancePoolId = args[0]
		}

		err = w.InstancePools.Delete(ctx, deleteReq)
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

// start edit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var editOverrides []func(
	*cobra.Command,
	*compute.EditInstancePool,
)

func newEdit() *cobra.Command {
	cmd := &cobra.Command{}

	var editReq compute.EditInstancePool
	var editJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: map via StringToStringVar: custom_tags
	// TODO: complex arg: disk_spec
	cmd.Flags().BoolVar(&editReq.EnableElasticDisk, "enable-elastic-disk", editReq.EnableElasticDisk, `Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	// TODO: complex arg: gcp_attributes
	cmd.Flags().IntVar(&editReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", editReq.IdleInstanceAutoterminationMinutes, `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	// TODO: complex arg: instance_pool_fleet_attributes
	cmd.Flags().IntVar(&editReq.MaxCapacity, "max-capacity", editReq.MaxCapacity, `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.`)
	cmd.Flags().IntVar(&editReq.MinIdleInstances, "min-idle-instances", editReq.MinIdleInstances, `Minimum number of idle instances to keep in the instance pool.`)
	// TODO: array: preloaded_docker_images
	// TODO: array: preloaded_spark_versions

	cmd.Use = "edit INSTANCE_POOL_ID INSTANCE_POOL_NAME NODE_TYPE_ID"
	cmd.Short = `Edit an existing instance pool.`
	cmd.Long = `Edit an existing instance pool.
  
  Modifies the configuration of an existing instance pool.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(3)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = editJson.Unmarshal(&editReq)
			if err != nil {
				return err
			}
		} else {
			editReq.InstancePoolId = args[0]
			editReq.InstancePoolName = args[1]
			editReq.NodeTypeId = args[2]
		}

		err = w.InstancePools.Edit(ctx, editReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newEdit())
	})
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*compute.GetInstancePoolRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq compute.GetInstancePoolRequest

	// TODO: short flags

	cmd.Use = "get INSTANCE_POOL_ID"
	cmd.Short = `Get instance pool information.`
	cmd.Long = `Get instance pool information.
  
  Retrieve the information for an instance pool based on its identifier.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No INSTANCE_POOL_ID argument specified. Loading names for Instance Pools drop-down."
			names, err := w.InstancePools.InstancePoolAndStatsInstancePoolNameToInstancePoolIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Instance Pools drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The canonical unique identifier for the instance pool")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the canonical unique identifier for the instance pool")
		}
		getReq.InstancePoolId = args[0]

		response, err := w.InstancePools.Get(ctx, getReq)
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

// start get-instance-pool-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getInstancePoolPermissionLevelsOverrides []func(
	*cobra.Command,
	*compute.GetInstancePoolPermissionLevelsRequest,
)

func newGetInstancePoolPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getInstancePoolPermissionLevelsReq compute.GetInstancePoolPermissionLevelsRequest

	// TODO: short flags

	cmd.Use = "get-instance-pool-permission-levels INSTANCE_POOL_ID"
	cmd.Short = `Get instance pool permission levels.`
	cmd.Long = `Get instance pool permission levels.
  
  Gets the permission levels that a user can have on an object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No INSTANCE_POOL_ID argument specified. Loading names for Instance Pools drop-down."
			names, err := w.InstancePools.InstancePoolAndStatsInstancePoolNameToInstancePoolIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Instance Pools drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The instance pool for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the instance pool for which to get or manage permissions")
		}
		getInstancePoolPermissionLevelsReq.InstancePoolId = args[0]

		response, err := w.InstancePools.GetInstancePoolPermissionLevels(ctx, getInstancePoolPermissionLevelsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getInstancePoolPermissionLevelsOverrides {
		fn(cmd, &getInstancePoolPermissionLevelsReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetInstancePoolPermissionLevels())
	})
}

// start get-instance-pool-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getInstancePoolPermissionsOverrides []func(
	*cobra.Command,
	*compute.GetInstancePoolPermissionsRequest,
)

func newGetInstancePoolPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getInstancePoolPermissionsReq compute.GetInstancePoolPermissionsRequest

	// TODO: short flags

	cmd.Use = "get-instance-pool-permissions INSTANCE_POOL_ID"
	cmd.Short = `Get instance pool permissions.`
	cmd.Long = `Get instance pool permissions.
  
  Gets the permissions of an instance pool. Instance pools can inherit
  permissions from their root object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No INSTANCE_POOL_ID argument specified. Loading names for Instance Pools drop-down."
			names, err := w.InstancePools.InstancePoolAndStatsInstancePoolNameToInstancePoolIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Instance Pools drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The instance pool for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the instance pool for which to get or manage permissions")
		}
		getInstancePoolPermissionsReq.InstancePoolId = args[0]

		response, err := w.InstancePools.GetInstancePoolPermissions(ctx, getInstancePoolPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getInstancePoolPermissionsOverrides {
		fn(cmd, &getInstancePoolPermissionsReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetInstancePoolPermissions())
	})
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `List instance pool info.`
	cmd.Long = `List instance pool info.
  
  Gets a list of instance pools with their statistics.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.InstancePools.ListAll(ctx)
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
		fn(cmd)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newList())
	})
}

// start set-instance-pool-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setInstancePoolPermissionsOverrides []func(
	*cobra.Command,
	*compute.InstancePoolPermissionsRequest,
)

func newSetInstancePoolPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setInstancePoolPermissionsReq compute.InstancePoolPermissionsRequest
	var setInstancePoolPermissionsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&setInstancePoolPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-instance-pool-permissions INSTANCE_POOL_ID"
	cmd.Short = `Set instance pool permissions.`
	cmd.Long = `Set instance pool permissions.
  
  Sets permissions on an instance pool. Instance pools can inherit permissions
  from their root object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = setInstancePoolPermissionsJson.Unmarshal(&setInstancePoolPermissionsReq)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No INSTANCE_POOL_ID argument specified. Loading names for Instance Pools drop-down."
			names, err := w.InstancePools.InstancePoolAndStatsInstancePoolNameToInstancePoolIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Instance Pools drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The instance pool for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the instance pool for which to get or manage permissions")
		}
		setInstancePoolPermissionsReq.InstancePoolId = args[0]

		response, err := w.InstancePools.SetInstancePoolPermissions(ctx, setInstancePoolPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setInstancePoolPermissionsOverrides {
		fn(cmd, &setInstancePoolPermissionsReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newSetInstancePoolPermissions())
	})
}

// start update-instance-pool-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateInstancePoolPermissionsOverrides []func(
	*cobra.Command,
	*compute.InstancePoolPermissionsRequest,
)

func newUpdateInstancePoolPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updateInstancePoolPermissionsReq compute.InstancePoolPermissionsRequest
	var updateInstancePoolPermissionsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateInstancePoolPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-instance-pool-permissions INSTANCE_POOL_ID"
	cmd.Short = `Update instance pool permissions.`
	cmd.Long = `Update instance pool permissions.
  
  Updates the permissions on an instance pool. Instance pools can inherit
  permissions from their root object.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateInstancePoolPermissionsJson.Unmarshal(&updateInstancePoolPermissionsReq)
			if err != nil {
				return err
			}
		}
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No INSTANCE_POOL_ID argument specified. Loading names for Instance Pools drop-down."
			names, err := w.InstancePools.InstancePoolAndStatsInstancePoolNameToInstancePoolIdMap(ctx)
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Instance Pools drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The instance pool for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the instance pool for which to get or manage permissions")
		}
		updateInstancePoolPermissionsReq.InstancePoolId = args[0]

		response, err := w.InstancePools.UpdateInstancePoolPermissions(ctx, updateInstancePoolPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateInstancePoolPermissionsOverrides {
		fn(cmd, &updateInstancePoolPermissionsReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdateInstancePoolPermissions())
	})
}

// end service InstancePools
