package clusters

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "clusters",
	Short: `The Clusters API allows you to create, start, edit, list, terminate, and delete clusters.`, // TODO: fix FirstSentence logic and append dot to summary
}

var changeOwnerReq clusters.ChangeClusterOwner

func init() {
	Cmd.AddCommand(changeOwnerCmd)
	// TODO: short flags

	changeOwnerCmd.Flags().StringVar(&changeOwnerReq.ClusterId, "cluster-id", "", `<needs content added>.`)
	changeOwnerCmd.Flags().StringVar(&changeOwnerReq.OwnerUsername, "owner-username", "", `New owner of the cluster_id after this RPC.`)

}

var changeOwnerCmd = &cobra.Command{
	Use:   "change-owner",
	Short: `Change cluster owner Change the owner of the cluster.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.ChangeOwner(ctx, changeOwnerReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var createReq clusters.CreateCluster

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().BoolVar(&createReq.ApplyPolicyDefaultValues, "apply-policy-default-values", false, `Note: This field won't be true for webapp requests.`)
	// TODO: complex arg: autoscale
	createCmd.Flags().IntVar(&createReq.AutoterminationMinutes, "autotermination-minutes", 0, `Automatically terminates the cluster after it is inactive for this time in minutes.`)
	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: complex arg: cluster_log_conf
	createCmd.Flags().StringVar(&createReq.ClusterName, "cluster-name", "", `Cluster name requested by the user.`)
	// TODO: complex arg: cluster_source
	// TODO: complex arg: custom_tags
	createCmd.Flags().StringVar(&createReq.DriverInstancePoolId, "driver-instance-pool-id", "", `The optional ID of the instance pool for the driver of the cluster belongs.`)
	createCmd.Flags().StringVar(&createReq.DriverNodeTypeId, "driver-node-type-id", "", `The node type of the Spark driver.`)
	createCmd.Flags().StringVar(&createReq.EffectiveSparkVersion, "effective-spark-version", "", `The key of the spark version running in the dataplane.`)
	createCmd.Flags().BoolVar(&createReq.EnableElasticDisk, "enable-elastic-disk", false, `Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	createCmd.Flags().BoolVar(&createReq.EnableLocalDiskEncryption, "enable-local-disk-encryption", false, `Whether to enable LUKS on cluster VMs' local disks.`)
	// TODO: complex arg: gcp_attributes
	createCmd.Flags().StringVar(&createReq.InstancePoolId, "instance-pool-id", "", `The optional ID of the instance pool to which the cluster belongs.`)
	createCmd.Flags().StringVar(&createReq.NodeTypeId, "node-type-id", "", `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	createCmd.Flags().IntVar(&createReq.NumWorkers, "num-workers", 0, `Number of worker nodes that this cluster should have.`)
	createCmd.Flags().StringVar(&createReq.PolicyId, "policy-id", "", `The ID of the cluster policy used to create the cluster if applicable.`)
	// TODO: complex arg: runtime_engine
	// TODO: complex arg: spark_conf
	// TODO: complex arg: spark_env_vars
	createCmd.Flags().StringVar(&createReq.SparkVersion, "spark-version", "", `The Spark version of the cluster, e.g.`)
	// TODO: complex arg: ssh_public_keys
	// TODO: complex arg: workload_type

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create new cluster Creates a new Spark cluster.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Clusters.Create(ctx, createReq)
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

var deleteReq clusters.DeleteCluster

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.ClusterId, "cluster-id", "", `The cluster to be terminated.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Terminate cluster Terminates the Spark cluster with the specified ID.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var editReq clusters.EditCluster

func init() {
	Cmd.AddCommand(editCmd)
	// TODO: short flags

	editCmd.Flags().BoolVar(&editReq.ApplyPolicyDefaultValues, "apply-policy-default-values", false, `Note: This field won't be true for webapp requests.`)
	// TODO: complex arg: autoscale
	editCmd.Flags().IntVar(&editReq.AutoterminationMinutes, "autotermination-minutes", 0, `Automatically terminates the cluster after it is inactive for this time in minutes.`)
	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	editCmd.Flags().StringVar(&editReq.ClusterId, "cluster-id", "", `<needs content added>.`)
	// TODO: complex arg: cluster_log_conf
	editCmd.Flags().StringVar(&editReq.ClusterName, "cluster-name", "", `Cluster name requested by the user.`)
	// TODO: complex arg: cluster_source
	// TODO: complex arg: custom_tags
	editCmd.Flags().StringVar(&editReq.DriverInstancePoolId, "driver-instance-pool-id", "", `The optional ID of the instance pool for the driver of the cluster belongs.`)
	editCmd.Flags().StringVar(&editReq.DriverNodeTypeId, "driver-node-type-id", "", `The node type of the Spark driver.`)
	editCmd.Flags().StringVar(&editReq.EffectiveSparkVersion, "effective-spark-version", "", `The key of the spark version running in the dataplane.`)
	editCmd.Flags().BoolVar(&editReq.EnableElasticDisk, "enable-elastic-disk", false, `Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	editCmd.Flags().BoolVar(&editReq.EnableLocalDiskEncryption, "enable-local-disk-encryption", false, `Whether to enable LUKS on cluster VMs' local disks.`)
	// TODO: complex arg: gcp_attributes
	editCmd.Flags().StringVar(&editReq.InstancePoolId, "instance-pool-id", "", `The optional ID of the instance pool to which the cluster belongs.`)
	editCmd.Flags().StringVar(&editReq.NodeTypeId, "node-type-id", "", `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	editCmd.Flags().IntVar(&editReq.NumWorkers, "num-workers", 0, `Number of worker nodes that this cluster should have.`)
	editCmd.Flags().StringVar(&editReq.PolicyId, "policy-id", "", `The ID of the cluster policy used to create the cluster if applicable.`)
	// TODO: complex arg: runtime_engine
	// TODO: complex arg: spark_conf
	// TODO: complex arg: spark_env_vars
	editCmd.Flags().StringVar(&editReq.SparkVersion, "spark-version", "", `The Spark version of the cluster, e.g.`)
	// TODO: complex arg: ssh_public_keys
	// TODO: complex arg: workload_type

}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: `Update cluster configuration Updates the configuration of a cluster to match the provided attributes and size.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.Edit(ctx, editReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var eventsReq clusters.GetEvents

func init() {
	Cmd.AddCommand(eventsCmd)
	// TODO: short flags

	eventsCmd.Flags().StringVar(&eventsReq.ClusterId, "cluster-id", "", `The ID of the cluster to retrieve events about.`)
	eventsCmd.Flags().Int64Var(&eventsReq.EndTime, "end-time", 0, `The end time in epoch milliseconds.`)
	// TODO: complex arg: event_types
	eventsCmd.Flags().Int64Var(&eventsReq.Limit, "limit", 0, `The maximum number of events to include in a page of events.`)
	eventsCmd.Flags().Int64Var(&eventsReq.Offset, "offset", 0, `The offset in the result set.`)
	// TODO: complex arg: order
	eventsCmd.Flags().Int64Var(&eventsReq.StartTime, "start-time", 0, `The start time in epoch milliseconds.`)

}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: `List cluster activity events Retrieves a list of events about the activity of a cluster.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Clusters.EventsAll(ctx, eventsReq)
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

var getReq clusters.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.ClusterId, "cluster-id", "", `The cluster about which to retrieve information.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get cluster info "Retrieves the information for a cluster given its identifier.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Clusters.Get(ctx, getReq)
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

var listReq clusters.List

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.CanUseClient, "can-use-client", "", `Filter clusters based on what type of client it can be used for.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List all clusters Returns information about all pinned clusters, currently active clusters, up to 70 of the most recently terminated interactive clusters in the past 7 days, and up to 30 of the most recently terminated job clusters in the past 7 days.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Clusters.ListAll(ctx, listReq)
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
	Cmd.AddCommand(listNodeTypesCmd)

}

var listNodeTypesCmd = &cobra.Command{
	Use:   "list-node-types",
	Short: `List node types Returns a list of supported Spark node types.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Clusters.ListNodeTypes(ctx)
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
	Cmd.AddCommand(listZonesCmd)

}

var listZonesCmd = &cobra.Command{
	Use:   "list-zones",
	Short: `List availability zones Returns a list of availability zones where clusters can be created in (For example, us-west-2a).`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Clusters.ListZones(ctx)
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

var permanentDeleteReq clusters.PermanentDeleteCluster

func init() {
	Cmd.AddCommand(permanentDeleteCmd)
	// TODO: short flags

	permanentDeleteCmd.Flags().StringVar(&permanentDeleteReq.ClusterId, "cluster-id", "", `The cluster to be deleted.`)

}

var permanentDeleteCmd = &cobra.Command{
	Use:   "permanent-delete",
	Short: `Permanently delete cluster Permanently deletes a Spark cluster.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.PermanentDelete(ctx, permanentDeleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var pinReq clusters.PinCluster

func init() {
	Cmd.AddCommand(pinCmd)
	// TODO: short flags

	pinCmd.Flags().StringVar(&pinReq.ClusterId, "cluster-id", "", `<needs content added>.`)

}

var pinCmd = &cobra.Command{
	Use:   "pin",
	Short: `Pin cluster Pinning a cluster ensures that the cluster will always be returned by the ListClusters API.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.Pin(ctx, pinReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var resizeReq clusters.ResizeCluster

func init() {
	Cmd.AddCommand(resizeCmd)
	// TODO: short flags

	// TODO: complex arg: autoscale
	resizeCmd.Flags().StringVar(&resizeReq.ClusterId, "cluster-id", "", `The cluster to be resized.`)
	resizeCmd.Flags().IntVar(&resizeReq.NumWorkers, "num-workers", 0, `Number of worker nodes that this cluster should have.`)

}

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: `Resize cluster Resizes a cluster to have a desired number of workers.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.Resize(ctx, resizeReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var restartReq clusters.RestartCluster

func init() {
	Cmd.AddCommand(restartCmd)
	// TODO: short flags

	restartCmd.Flags().StringVar(&restartReq.ClusterId, "cluster-id", "", `The cluster to be started.`)
	restartCmd.Flags().StringVar(&restartReq.RestartUser, "restart-user", "", `<needs content added>.`)

}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: `Restart cluster Restarts a Spark cluster with the supplied ID.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.Restart(ctx, restartReq)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	Cmd.AddCommand(sparkVersionsCmd)

}

var sparkVersionsCmd = &cobra.Command{
	Use:   "spark-versions",
	Short: `List available Spark versions Returns the list of available Spark versions.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Clusters.SparkVersions(ctx)
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

var startReq clusters.StartCluster

func init() {
	Cmd.AddCommand(startCmd)
	// TODO: short flags

	startCmd.Flags().StringVar(&startReq.ClusterId, "cluster-id", "", `The cluster to be started.`)

}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: `Start terminated cluster Starts a terminated Spark cluster with the supplied ID.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.Start(ctx, startReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var unpinReq clusters.UnpinCluster

func init() {
	Cmd.AddCommand(unpinCmd)
	// TODO: short flags

	unpinCmd.Flags().StringVar(&unpinReq.ClusterId, "cluster-id", "", `<needs content added>.`)

}

var unpinCmd = &cobra.Command{
	Use:   "unpin",
	Short: `Unpin cluster Unpinning a cluster will allow the cluster to eventually be removed from the ListClusters API.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Clusters.Unpin(ctx, unpinReq)
		if err != nil {
			return err
		}

		return nil
	},
}
