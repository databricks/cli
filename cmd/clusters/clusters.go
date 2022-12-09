package clusters

import (
	instance_profiles "github.com/databricks/bricks/cmd/clusters/instance-profiles"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "clusters",
	Short: `The Clusters API allows you to create, start, edit, list, terminate, and delete clusters.`,
	Long: `The Clusters API allows you to create, start, edit, list, terminate, and
  delete clusters.
  
  Databricks maps cluster node instance types to compute units known as DBUs.
  See the instance type pricing page for a list of the supported instance types
  and their corresponding DBUs.
  
  A Databricks cluster is a set of computation resources and configurations on
  which you run data engineering, data science, and data analytics workloads,
  such as production ETL pipelines, streaming analytics, ad-hoc analytics, and
  machine learning.
  
  You run these workloads as a set of commands in a notebook or as an automated
  job. Databricks makes a distinction between all-purpose clusters and job
  clusters. You use all-purpose clusters to analyze data collaboratively using
  interactive notebooks. You use job clusters to run fast and robust automated
  jobs.
  
  You can create an all-purpose cluster using the UI, CLI, or REST API. You can
  manually terminate and restart an all-purpose cluster. Multiple users can
  share such clusters to do collaborative interactive analysis.
  
  IMPORTANT: Databricks retains cluster configuration information for up to 200
  all-purpose clusters terminated in the last 30 days and up to 30 job clusters
  recently terminated by the job scheduler. To keep an all-purpose cluster
  configuration even after it has been terminated for more than 30 days, an
  administrator can pin a cluster to the cluster list.`,
}

var changeOwnerReq clusters.ChangeClusterOwner

func init() {
	Cmd.AddCommand(changeOwnerCmd)
	// TODO: short flags

	changeOwnerCmd.Flags().StringVar(&changeOwnerReq.ClusterId, "cluster-id", changeOwnerReq.ClusterId, `<needs content added>.`)
	changeOwnerCmd.Flags().StringVar(&changeOwnerReq.OwnerUsername, "owner-username", changeOwnerReq.OwnerUsername, `New owner of the cluster_id after this RPC.`)

}

var changeOwnerCmd = &cobra.Command{
	Use:   "change-owner",
	Short: `Change cluster owner.`,
	Long: `Change cluster owner.
  
  Change the owner of the cluster. You must be an admin to perform this
  operation.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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

	createCmd.Flags().BoolVar(&createReq.ApplyPolicyDefaultValues, "apply-policy-default-values", createReq.ApplyPolicyDefaultValues, `Note: This field won't be true for webapp requests.`)
	// TODO: complex arg: autoscale
	createCmd.Flags().IntVar(&createReq.AutoterminationMinutes, "autotermination-minutes", createReq.AutoterminationMinutes, `Automatically terminates the cluster after it is inactive for this time in minutes.`)
	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: complex arg: cluster_log_conf
	createCmd.Flags().StringVar(&createReq.ClusterName, "cluster-name", createReq.ClusterName, `Cluster name requested by the user.`)
	createCmd.Flags().Var(&createReq.ClusterSource, "cluster-source", `Determines whether the cluster was created by a user through the UI, created by the Databricks Jobs Scheduler, or through an API request.`)
	// TODO: map via StringToStringVar: custom_tags
	createCmd.Flags().StringVar(&createReq.DriverInstancePoolId, "driver-instance-pool-id", createReq.DriverInstancePoolId, `The optional ID of the instance pool for the driver of the cluster belongs.`)
	createCmd.Flags().StringVar(&createReq.DriverNodeTypeId, "driver-node-type-id", createReq.DriverNodeTypeId, `The node type of the Spark driver.`)
	createCmd.Flags().BoolVar(&createReq.EnableElasticDisk, "enable-elastic-disk", createReq.EnableElasticDisk, `Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	createCmd.Flags().BoolVar(&createReq.EnableLocalDiskEncryption, "enable-local-disk-encryption", createReq.EnableLocalDiskEncryption, `Whether to enable LUKS on cluster VMs' local disks.`)
	// TODO: complex arg: gcp_attributes
	createCmd.Flags().StringVar(&createReq.InstancePoolId, "instance-pool-id", createReq.InstancePoolId, `The optional ID of the instance pool to which the cluster belongs.`)
	createCmd.Flags().StringVar(&createReq.NodeTypeId, "node-type-id", createReq.NodeTypeId, `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	createCmd.Flags().IntVar(&createReq.NumWorkers, "num-workers", createReq.NumWorkers, `Number of worker nodes that this cluster should have.`)
	createCmd.Flags().StringVar(&createReq.PolicyId, "policy-id", createReq.PolicyId, `The ID of the cluster policy used to create the cluster if applicable.`)
	createCmd.Flags().Var(&createReq.RuntimeEngine, "runtime-engine", `Decides which runtime engine to be use, e.g.`)
	// TODO: map via StringToStringVar: spark_conf
	// TODO: map via StringToStringVar: spark_env_vars
	createCmd.Flags().StringVar(&createReq.SparkVersion, "spark-version", createReq.SparkVersion, `The Spark version of the cluster, e.g.`)
	// TODO: array: ssh_public_keys
	// TODO: complex arg: workload_type

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create new cluster.`,
	Long: `Create new cluster.
  
  Creates a new Spark cluster. This method will acquire new instances from the
  cloud provider if necessary. This method is asynchronous; the returned
  cluster_id can be used to poll the cluster status. When this method returns,
  the cluster will be in\na PENDING state. The cluster will be usable once it
  enters a RUNNING state.
  
  Note: Databricks may not be able to acquire some of the requested nodes, due
  to cloud provider limitations (account limits, spot price, etc.) or transient
  network issues.
  
  If Databricks acquires at least 85% of the requested on-demand nodes, cluster
  creation will succeed. Otherwise the cluster will terminate with an
  informative error message.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Clusters.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteReq clusters.DeleteCluster

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.ClusterId, "cluster-id", deleteReq.ClusterId, `The cluster to be terminated.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Terminate cluster.`,
	Long: `Terminate cluster.
  
  Terminates the Spark cluster with the specified ID. The cluster is removed
  asynchronously. Once the termination has completed, the cluster will be in a
  TERMINATED state. If the cluster is already in a TERMINATING or
  TERMINATED state, nothing will happen.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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

	editCmd.Flags().BoolVar(&editReq.ApplyPolicyDefaultValues, "apply-policy-default-values", editReq.ApplyPolicyDefaultValues, `Note: This field won't be true for webapp requests.`)
	// TODO: complex arg: autoscale
	editCmd.Flags().IntVar(&editReq.AutoterminationMinutes, "autotermination-minutes", editReq.AutoterminationMinutes, `Automatically terminates the cluster after it is inactive for this time in minutes.`)
	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	editCmd.Flags().StringVar(&editReq.ClusterId, "cluster-id", editReq.ClusterId, `ID of the cluser.`)
	// TODO: complex arg: cluster_log_conf
	editCmd.Flags().StringVar(&editReq.ClusterName, "cluster-name", editReq.ClusterName, `Cluster name requested by the user.`)
	editCmd.Flags().Var(&editReq.ClusterSource, "cluster-source", `Determines whether the cluster was created by a user through the UI, created by the Databricks Jobs Scheduler, or through an API request.`)
	// TODO: map via StringToStringVar: custom_tags
	editCmd.Flags().StringVar(&editReq.DriverInstancePoolId, "driver-instance-pool-id", editReq.DriverInstancePoolId, `The optional ID of the instance pool for the driver of the cluster belongs.`)
	editCmd.Flags().StringVar(&editReq.DriverNodeTypeId, "driver-node-type-id", editReq.DriverNodeTypeId, `The node type of the Spark driver.`)
	editCmd.Flags().BoolVar(&editReq.EnableElasticDisk, "enable-elastic-disk", editReq.EnableElasticDisk, `Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	editCmd.Flags().BoolVar(&editReq.EnableLocalDiskEncryption, "enable-local-disk-encryption", editReq.EnableLocalDiskEncryption, `Whether to enable LUKS on cluster VMs' local disks.`)
	// TODO: complex arg: gcp_attributes
	editCmd.Flags().StringVar(&editReq.InstancePoolId, "instance-pool-id", editReq.InstancePoolId, `The optional ID of the instance pool to which the cluster belongs.`)
	editCmd.Flags().StringVar(&editReq.NodeTypeId, "node-type-id", editReq.NodeTypeId, `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	editCmd.Flags().IntVar(&editReq.NumWorkers, "num-workers", editReq.NumWorkers, `Number of worker nodes that this cluster should have.`)
	editCmd.Flags().StringVar(&editReq.PolicyId, "policy-id", editReq.PolicyId, `The ID of the cluster policy used to create the cluster if applicable.`)
	editCmd.Flags().Var(&editReq.RuntimeEngine, "runtime-engine", `Decides which runtime engine to be use, e.g.`)
	// TODO: map via StringToStringVar: spark_conf
	// TODO: map via StringToStringVar: spark_env_vars
	editCmd.Flags().StringVar(&editReq.SparkVersion, "spark-version", editReq.SparkVersion, `The Spark version of the cluster, e.g.`)
	// TODO: array: ssh_public_keys
	// TODO: complex arg: workload_type

}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: `Update cluster configuration.`,
	Long: `Update cluster configuration.
  
  Updates the configuration of a cluster to match the provided attributes and
  size. A cluster can be updated if it is in a RUNNING or TERMINATED state.
  
  If a cluster is updated while in a RUNNING state, it will be restarted so
  that the new attributes can take effect.
  
  If a cluster is updated while in a TERMINATED state, it will remain
  TERMINATED. The next time it is started using the clusters/start API, the
  new attributes will take effect. Any attempt to update a cluster in any other
  state will be rejected with an INVALID_STATE error code.
  
  Clusters created by the Databricks Jobs service cannot be edited.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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

	eventsCmd.Flags().StringVar(&eventsReq.ClusterId, "cluster-id", eventsReq.ClusterId, `The ID of the cluster to retrieve events about.`)
	eventsCmd.Flags().Int64Var(&eventsReq.EndTime, "end-time", eventsReq.EndTime, `The end time in epoch milliseconds.`)
	// TODO: array: event_types
	eventsCmd.Flags().Int64Var(&eventsReq.Limit, "limit", eventsReq.Limit, `The maximum number of events to include in a page of events.`)
	eventsCmd.Flags().Int64Var(&eventsReq.Offset, "offset", eventsReq.Offset, `The offset in the result set.`)
	eventsCmd.Flags().Var(&eventsReq.Order, "order", `The order to list events in; either "ASC" or "DESC".`)
	eventsCmd.Flags().Int64Var(&eventsReq.StartTime, "start-time", eventsReq.StartTime, `The start time in epoch milliseconds.`)

}

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: `List cluster activity events.`,
	Long: `List cluster activity events.
  
  Retrieves a list of events about the activity of a cluster. This API is
  paginated. If there are more events to read, the response includes all the
  nparameters necessary to request the next page of events.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Clusters.EventsAll(ctx, eventsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var getReq clusters.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.ClusterId, "cluster-id", getReq.ClusterId, `The cluster about which to retrieve information.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get cluster info.`,
	Long: `Get cluster info.
  
  "Retrieves the information for a cluster given its identifier. Clusters can be
  described while they are running, or up to 60 days after they are terminated.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Clusters.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var listReq clusters.List

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.CanUseClient, "can-use-client", listReq.CanUseClient, `Filter clusters based on what type of client it can be used for.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List all clusters.`,
	Long: `List all clusters.
  
  Returns information about all pinned clusters, currently active clusters, up
  to 70 of the most recently terminated interactive clusters in the past 7 days,
  and up to 30 of the most recently terminated job clusters in the past 7 days.
  
  For example, if there is 1 pinned cluster, 4 active clusters, 45 terminated
  interactive clusters in the past 7 days, and 50 terminated job clusters\nin
  the past 7 days, then this API returns the 1 pinned cluster, 4 active
  clusters, all 45 terminated interactive clusters, and the 30 most recently
  terminated job clusters.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Clusters.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

func init() {
	Cmd.AddCommand(listNodeTypesCmd)

}

var listNodeTypesCmd = &cobra.Command{
	Use:   "list-node-types",
	Short: `List node types.`,
	Long: `List node types.
  
  Returns a list of supported Spark node types. These node types can be used to
  launch a cluster.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Clusters.ListNodeTypes(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

func init() {
	Cmd.AddCommand(listZonesCmd)

}

var listZonesCmd = &cobra.Command{
	Use:   "list-zones",
	Short: `List availability zones.`,
	Long: `List availability zones.
  
  Returns a list of availability zones where clusters can be created in (For
  example, us-west-2a). These zones can be used to launch a cluster.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Clusters.ListZones(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var permanentDeleteReq clusters.PermanentDeleteCluster

func init() {
	Cmd.AddCommand(permanentDeleteCmd)
	// TODO: short flags

	permanentDeleteCmd.Flags().StringVar(&permanentDeleteReq.ClusterId, "cluster-id", permanentDeleteReq.ClusterId, `The cluster to be deleted.`)

}

var permanentDeleteCmd = &cobra.Command{
	Use:   "permanent-delete",
	Short: `Permanently delete cluster.`,
	Long: `Permanently delete cluster.
  
  Permanently deletes a Spark cluster. This cluster is terminated and resources
  are asynchronously removed.
  
  In addition, users will no longer see permanently deleted clusters in the
  cluster list, and API users can no longer perform any action on permanently
  deleted clusters.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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

	pinCmd.Flags().StringVar(&pinReq.ClusterId, "cluster-id", pinReq.ClusterId, `<needs content added>.`)

}

var pinCmd = &cobra.Command{
	Use:   "pin",
	Short: `Pin cluster.`,
	Long: `Pin cluster.
  
  Pinning a cluster ensures that the cluster will always be returned by the
  ListClusters API. Pinning a cluster that is already pinned will have no
  effect. This API can only be called by workspace admins.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	resizeCmd.Flags().StringVar(&resizeReq.ClusterId, "cluster-id", resizeReq.ClusterId, `The cluster to be resized.`)
	resizeCmd.Flags().IntVar(&resizeReq.NumWorkers, "num-workers", resizeReq.NumWorkers, `Number of worker nodes that this cluster should have.`)

}

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: `Resize cluster.`,
	Long: `Resize cluster.
  
  Resizes a cluster to have a desired number of workers. This will fail unless
  the cluster is in a RUNNING state.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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

	restartCmd.Flags().StringVar(&restartReq.ClusterId, "cluster-id", restartReq.ClusterId, `The cluster to be started.`)
	restartCmd.Flags().StringVar(&restartReq.RestartUser, "restart-user", restartReq.RestartUser, `<needs content added>.`)

}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: `Restart cluster.`,
	Long: `Restart cluster.
  
  Restarts a Spark cluster with the supplied ID. If the cluster is not currently
  in a RUNNING state, nothing will happen.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Short: `List available Spark versions.`,
	Long: `List available Spark versions.
  
  Returns the list of available Spark versions. These versions can be used to
  launch a cluster.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Clusters.SparkVersions(ctx)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var startReq clusters.StartCluster

func init() {
	Cmd.AddCommand(startCmd)
	// TODO: short flags

	startCmd.Flags().StringVar(&startReq.ClusterId, "cluster-id", startReq.ClusterId, `The cluster to be started.`)

}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: `Start terminated cluster.`,
	Long: `Start terminated cluster.
  
  Starts a terminated Spark cluster with the supplied ID. This works similar to
  createCluster except:
  
  * The previous cluster id and attributes are preserved. * The cluster starts
  with the last specified cluster size. * If the previous cluster was an
  autoscaling cluster, the current cluster starts with the minimum number of
  nodes. * If the cluster is not currently in a TERMINATED state, nothing will
  happen. * Clusters launched to run a job cannot be started.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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

	unpinCmd.Flags().StringVar(&unpinReq.ClusterId, "cluster-id", unpinReq.ClusterId, `<needs content added>.`)

}

var unpinCmd = &cobra.Command{
	Use:   "unpin",
	Short: `Unpin cluster.`,
	Long: `Unpin cluster.
  
  Unpinning a cluster will allow the cluster to eventually be removed from the
  ListClusters API. Unpinning a cluster that is not pinned will have no effect.
  This API can only be called by workspace admins.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Clusters.Unpin(ctx, unpinReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Clusters

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(instance_profiles.Cmd)
}
