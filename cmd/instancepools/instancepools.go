// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package instancepools

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/instancepools"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
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
}

// start create command

var createReq instancepools.CreateInstancePool
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: map via StringToStringVar: custom_tags
	// TODO: complex arg: disk_spec
	createCmd.Flags().BoolVar(&createReq.EnableElasticDisk, "enable-elastic-disk", createReq.EnableElasticDisk, `Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	createCmd.Flags().IntVar(&createReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", createReq.IdleInstanceAutoterminationMinutes, `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	// TODO: complex arg: instance_pool_fleet_attributes
	createCmd.Flags().IntVar(&createReq.MaxCapacity, "max-capacity", createReq.MaxCapacity, `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.`)
	createCmd.Flags().IntVar(&createReq.MinIdleInstances, "min-idle-instances", createReq.MinIdleInstances, `Minimum number of idle instances to keep in the instance pool.`)
	// TODO: array: preloaded_docker_images
	// TODO: array: preloaded_spark_versions

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new instance pool.`,
	Long: `Create a new instance pool.
  
  Creates a new instance pool using idle and ready-to-use cloud instances.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.InstancePools.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq instancepools.DeleteInstancePool

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete INSTANCE_POOL_ID",
	Short: `Delete an instance pool.`,
	Long: `Delete an instance pool.
  
  Deletes the instance pool permanently. The idle instances in the pool are
  terminated asynchronously.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		deleteReq.InstancePoolId = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.InstancePools.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start edit command

var editReq instancepools.EditInstancePool
var editJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags
	editCmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: map via StringToStringVar: custom_tags
	// TODO: complex arg: disk_spec
	editCmd.Flags().BoolVar(&editReq.EnableElasticDisk, "enable-elastic-disk", editReq.EnableElasticDisk, `Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	editCmd.Flags().IntVar(&editReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", editReq.IdleInstanceAutoterminationMinutes, `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	// TODO: complex arg: instance_pool_fleet_attributes
	editCmd.Flags().IntVar(&editReq.MaxCapacity, "max-capacity", editReq.MaxCapacity, `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.`)
	editCmd.Flags().IntVar(&editReq.MinIdleInstances, "min-idle-instances", editReq.MinIdleInstances, `Minimum number of idle instances to keep in the instance pool.`)
	// TODO: array: preloaded_docker_images
	// TODO: array: preloaded_spark_versions

}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: `Edit an existing instance pool.`,
	Long: `Edit an existing instance pool.
  
  Modifies the configuration of an existing instance pool.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = editJson.Unmarshall(&editReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.InstancePools.Edit(ctx, editReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq instancepools.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get INSTANCE_POOL_ID",
	Short: `Get instance pool information.`,
	Long: `Get instance pool information.
  
  Retrieve the information for an instance pool based on its identifier.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		getReq.InstancePoolId = args[0]
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.InstancePools.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List instance pool info.`,
	Long: `List instance pool info.
  
  Gets a list of instance pools with their statistics.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.InstancePools.ListAll(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service InstancePools

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
