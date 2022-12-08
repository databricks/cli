package instance_pools

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/instancepools"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "instance-pools",
	Short: `Instance Pools API are used to create, edit, delete and list instance pools by using ready-to-use cloud instances which reduces a cluster start and auto-scaling times.`,
}

var createReq instancepools.CreateInstancePool

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: map via StringToStringVar: custom_tags
	// TODO: complex arg: disk_spec
	createCmd.Flags().BoolVar(&createReq.EnableElasticDisk, "enable-elastic-disk", false, `Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	createCmd.Flags().IntVar(&createReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", 0, `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	// TODO: complex arg: instance_pool_fleet_attributes
	createCmd.Flags().StringVar(&createReq.InstancePoolName, "instance-pool-name", "", `Pool name requested by the user.`)
	createCmd.Flags().IntVar(&createReq.MaxCapacity, "max-capacity", 0, `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle instances.`)
	createCmd.Flags().IntVar(&createReq.MinIdleInstances, "min-idle-instances", 0, `Minimum number of idle instances to keep in the instance pool.`)
	createCmd.Flags().StringVar(&createReq.NodeTypeId, "node-type-id", "", `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	// TODO: array: preloaded_docker_images
	// TODO: array: preloaded_spark_versions

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new instance pool.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.InstancePools.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq instancepools.DeleteInstancePool

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.InstancePoolId, "instance-pool-id", "", `The instance pool to be terminated.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete an instance pool.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.InstancePools.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var editReq instancepools.EditInstancePool

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags

	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: map via StringToStringVar: custom_tags
	// TODO: complex arg: disk_spec
	editCmd.Flags().BoolVar(&editReq.EnableElasticDisk, "enable-elastic-disk", false, `Autoscaling Local Storage: when enabled, this instances in this pool will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	editCmd.Flags().IntVar(&editReq.IdleInstanceAutoterminationMinutes, "idle-instance-autotermination-minutes", 0, `Automatically terminates the extra instances in the pool cache after they are inactive for this time in minutes if min_idle_instances requirement is already met.`)
	editCmd.Flags().StringVar(&editReq.InstancePoolId, "instance-pool-id", "", ``)
	editCmd.Flags().StringVar(&editReq.InstancePoolName, "instance-pool-name", "", `Pool name requested by the user.`)
	editCmd.Flags().IntVar(&editReq.MaxCapacity, "max-capacity", 0, `Maximum number of outstanding instances to keep in the pool, including both instances used by clusters and idle ones.`)
	editCmd.Flags().IntVar(&editReq.MinIdleInstances, "min-idle-instances", 0, `Minimum number of idle instances to keep in the instance pool.`)
	editCmd.Flags().StringVar(&editReq.NodeTypeId, "node-type-id", "", `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	// TODO: array: preloaded_docker_images
	// TODO: array: preloaded_spark_versions

}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: `Edit an existing instance pool.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.InstancePools.Edit(ctx, editReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq instancepools.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.InstancePoolId, "instance-pool-id", "", `The canonical unique identifier for the instance pool.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get instance pool information.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.InstancePools.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List instance pool info.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.InstancePools.ListAll(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}
