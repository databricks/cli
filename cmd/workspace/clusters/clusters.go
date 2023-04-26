// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clusters

import (
	"fmt"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
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

// start change-owner command

var changeOwnerReq compute.ChangeClusterOwner

func init() {
	Cmd.AddCommand(changeOwnerCmd)
	// TODO: short flags

}

var changeOwnerCmd = &cobra.Command{
	Use:   "change-owner CLUSTER_ID OWNER_USERNAME",
	Short: `Change cluster owner.`,
	Long: `Change cluster owner.
  
  Change the owner of the cluster. You must be an admin to perform this
  operation.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		changeOwnerReq.ClusterId = args[0]
		changeOwnerReq.OwnerUsername = args[1]

		err = w.Clusters.ChangeOwner(ctx, changeOwnerReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start create command

var createReq compute.CreateCluster
var createJson flags.JsonFlag
var createSkipWait bool
var createTimeout time.Duration

func init() {
	Cmd.AddCommand(createCmd)

	createCmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach RUNNING state`)
	createCmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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
  the cluster will be in a PENDING state. The cluster will be usable once it
  enters a RUNNING state.
  
  Note: Databricks may not be able to acquire some of the requested nodes, due
  to cloud provider limitations (account limits, spot price, etc.) or transient
  network issues.
  
  If Databricks acquires at least 85% of the requested on-demand nodes, cluster
  creation will succeed. Otherwise the cluster will terminate with an
  informative error message.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = createJson.Unmarshal(&createReq)
		if err != nil {
			return err
		}
		createReq.SparkVersion = args[0]

		if createSkipWait {
			response, err := w.Clusters.Create(ctx, createReq)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Clusters.CreateAndWait(ctx, createReq,
			retries.Timeout[compute.ClusterInfo](createTimeout),
			func(i *retries.Info[compute.ClusterInfo]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.StateMessage
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start delete command

var deleteReq compute.DeleteCluster

var deleteSkipWait bool
var deleteTimeout time.Duration

func init() {
	Cmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVar(&deleteSkipWait, "no-wait", deleteSkipWait, `do not wait to reach TERMINATED state`)
	deleteCmd.Flags().DurationVar(&deleteTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED state`)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete CLUSTER_ID",
	Short: `Terminate cluster.`,
	Long: `Terminate cluster.
  
  Terminates the Spark cluster with the specified ID. The cluster is removed
  asynchronously. Once the termination has completed, the cluster will be in a
  TERMINATED state. If the cluster is already in a TERMINATING or
  TERMINATED state, nothing will happen.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The cluster to be terminated")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster to be terminated")
		}
		deleteReq.ClusterId = args[0]

		if deleteSkipWait {
			err = w.Clusters.Delete(ctx, deleteReq)
			if err != nil {
				return err
			}
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Clusters.DeleteAndWait(ctx, deleteReq,
			retries.Timeout[compute.ClusterInfo](deleteTimeout),
			func(i *retries.Info[compute.ClusterInfo]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.StateMessage
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start edit command

var editReq compute.EditCluster
var editJson flags.JsonFlag
var editSkipWait bool
var editTimeout time.Duration

func init() {
	Cmd.AddCommand(editCmd)

	editCmd.Flags().BoolVar(&editSkipWait, "no-wait", editSkipWait, `do not wait to reach RUNNING state`)
	editCmd.Flags().DurationVar(&editTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags
	editCmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	editCmd.Flags().BoolVar(&editReq.ApplyPolicyDefaultValues, "apply-policy-default-values", editReq.ApplyPolicyDefaultValues, `Note: This field won't be true for webapp requests.`)
	// TODO: complex arg: autoscale
	editCmd.Flags().IntVar(&editReq.AutoterminationMinutes, "autotermination-minutes", editReq.AutoterminationMinutes, `Automatically terminates the cluster after it is inactive for this time in minutes.`)
	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
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

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = editJson.Unmarshal(&editReq)
		if err != nil {
			return err
		}
		editReq.ClusterId = args[0]
		editReq.SparkVersion = args[1]

		if editSkipWait {
			err = w.Clusters.Edit(ctx, editReq)
			if err != nil {
				return err
			}
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Clusters.EditAndWait(ctx, editReq,
			retries.Timeout[compute.ClusterInfo](editTimeout),
			func(i *retries.Info[compute.ClusterInfo]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.StateMessage
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start events command

var eventsReq compute.GetEvents
var eventsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(eventsCmd)
	// TODO: short flags
	eventsCmd.Flags().Var(&eventsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

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

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = eventsJson.Unmarshal(&eventsReq)
		if err != nil {
			return err
		}
		eventsReq.ClusterId = args[0]

		response, err := w.Clusters.EventsAll(ctx, eventsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get command

var getReq compute.GetClusterRequest

var getSkipWait bool
var getTimeout time.Duration

func init() {
	Cmd.AddCommand(getCmd)

	getCmd.Flags().BoolVar(&getSkipWait, "no-wait", getSkipWait, `do not wait to reach RUNNING state`)
	getCmd.Flags().DurationVar(&getTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get CLUSTER_ID",
	Short: `Get cluster info.`,
	Long: `Get cluster info.
  
  "Retrieves the information for a cluster given its identifier. Clusters can be
  described while they are running, or up to 60 days after they are terminated.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The cluster about which to retrieve information")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster about which to retrieve information")
		}
		getReq.ClusterId = args[0]

		response, err := w.Clusters.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq compute.ListClustersRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().StringVar(&listReq.CanUseClient, "can-use-client", listReq.CanUseClient, `Filter clusters based on what type of client it can be used for.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List all clusters.`,
	Long: `List all clusters.
  
  Return information about all pinned clusters, active clusters, up to 200 of
  the most recently terminated all-purpose clusters in the past 30 days, and up
  to 30 of the most recently terminated job clusters in the past 30 days.
  
  For example, if there is 1 pinned cluster, 4 active clusters, 45 terminated
  all-purpose clusters in the past 30 days, and 50 terminated job clusters in
  the past 30 days, then this API returns the 1 pinned cluster, 4 active
  clusters, all 45 terminated all-purpose clusters, and the 30 most recently
  terminated job clusters.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response, err := w.Clusters.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list-node-types command

func init() {
	Cmd.AddCommand(listNodeTypesCmd)

}

var listNodeTypesCmd = &cobra.Command{
	Use:   "list-node-types",
	Short: `List node types.`,
	Long: `List node types.
  
  Returns a list of supported Spark node types. These node types can be used to
  launch a cluster.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Clusters.ListNodeTypes(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list-zones command

func init() {
	Cmd.AddCommand(listZonesCmd)

}

var listZonesCmd = &cobra.Command{
	Use:   "list-zones",
	Short: `List availability zones.`,
	Long: `List availability zones.
  
  Returns a list of availability zones where clusters can be created in (For
  example, us-west-2a). These zones can be used to launch a cluster.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Clusters.ListZones(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start permanent-delete command

var permanentDeleteReq compute.PermanentDeleteCluster

func init() {
	Cmd.AddCommand(permanentDeleteCmd)
	// TODO: short flags

}

var permanentDeleteCmd = &cobra.Command{
	Use:   "permanent-delete CLUSTER_ID",
	Short: `Permanently delete cluster.`,
	Long: `Permanently delete cluster.
  
  Permanently deletes a Spark cluster. This cluster is terminated and resources
  are asynchronously removed.
  
  In addition, users will no longer see permanently deleted clusters in the
  cluster list, and API users can no longer perform any action on permanently
  deleted clusters.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The cluster to be deleted")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster to be deleted")
		}
		permanentDeleteReq.ClusterId = args[0]

		err = w.Clusters.PermanentDelete(ctx, permanentDeleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start pin command

var pinReq compute.PinCluster

func init() {
	Cmd.AddCommand(pinCmd)
	// TODO: short flags

}

var pinCmd = &cobra.Command{
	Use:   "pin CLUSTER_ID",
	Short: `Pin cluster.`,
	Long: `Pin cluster.
  
  Pinning a cluster ensures that the cluster will always be returned by the
  ListClusters API. Pinning a cluster that is already pinned will have no
  effect. This API can only be called by workspace admins.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "<needs content added>")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have <needs content added>")
		}
		pinReq.ClusterId = args[0]

		err = w.Clusters.Pin(ctx, pinReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start resize command

var resizeReq compute.ResizeCluster
var resizeJson flags.JsonFlag
var resizeSkipWait bool
var resizeTimeout time.Duration

func init() {
	Cmd.AddCommand(resizeCmd)

	resizeCmd.Flags().BoolVar(&resizeSkipWait, "no-wait", resizeSkipWait, `do not wait to reach RUNNING state`)
	resizeCmd.Flags().DurationVar(&resizeTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags
	resizeCmd.Flags().Var(&resizeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: autoscale
	resizeCmd.Flags().IntVar(&resizeReq.NumWorkers, "num-workers", resizeReq.NumWorkers, `Number of worker nodes that this cluster should have.`)

}

var resizeCmd = &cobra.Command{
	Use:   "resize",
	Short: `Resize cluster.`,
	Long: `Resize cluster.
  
  Resizes a cluster to have a desired number of workers. This will fail unless
  the cluster is in a RUNNING state.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = resizeJson.Unmarshal(&resizeReq)
		if err != nil {
			return err
		}
		resizeReq.ClusterId = args[0]

		if resizeSkipWait {
			err = w.Clusters.Resize(ctx, resizeReq)
			if err != nil {
				return err
			}
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Clusters.ResizeAndWait(ctx, resizeReq,
			retries.Timeout[compute.ClusterInfo](resizeTimeout),
			func(i *retries.Info[compute.ClusterInfo]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.StateMessage
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start restart command

var restartReq compute.RestartCluster

var restartSkipWait bool
var restartTimeout time.Duration

func init() {
	Cmd.AddCommand(restartCmd)

	restartCmd.Flags().BoolVar(&restartSkipWait, "no-wait", restartSkipWait, `do not wait to reach RUNNING state`)
	restartCmd.Flags().DurationVar(&restartTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	restartCmd.Flags().StringVar(&restartReq.RestartUser, "restart-user", restartReq.RestartUser, `<needs content added>.`)

}

var restartCmd = &cobra.Command{
	Use:   "restart CLUSTER_ID",
	Short: `Restart cluster.`,
	Long: `Restart cluster.
  
  Restarts a Spark cluster with the supplied ID. If the cluster is not currently
  in a RUNNING state, nothing will happen.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The cluster to be started")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster to be started")
		}
		restartReq.ClusterId = args[0]

		if restartSkipWait {
			err = w.Clusters.Restart(ctx, restartReq)
			if err != nil {
				return err
			}
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Clusters.RestartAndWait(ctx, restartReq,
			retries.Timeout[compute.ClusterInfo](restartTimeout),
			func(i *retries.Info[compute.ClusterInfo]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.StateMessage
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start spark-versions command

func init() {
	Cmd.AddCommand(sparkVersionsCmd)

}

var sparkVersionsCmd = &cobra.Command{
	Use:   "spark-versions",
	Short: `List available Spark versions.`,
	Long: `List available Spark versions.
  
  Returns the list of available Spark versions. These versions can be used to
  launch a cluster.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Clusters.SparkVersions(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start start command

var startReq compute.StartCluster

var startSkipWait bool
var startTimeout time.Duration

func init() {
	Cmd.AddCommand(startCmd)

	startCmd.Flags().BoolVar(&startSkipWait, "no-wait", startSkipWait, `do not wait to reach RUNNING state`)
	startCmd.Flags().DurationVar(&startTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

}

var startCmd = &cobra.Command{
	Use:   "start CLUSTER_ID",
	Short: `Start terminated cluster.`,
	Long: `Start terminated cluster.
  
  Starts a terminated Spark cluster with the supplied ID. This works similar to
  createCluster except:
  
  * The previous cluster id and attributes are preserved. * The cluster starts
  with the last specified cluster size. * If the previous cluster was an
  autoscaling cluster, the current cluster starts with the minimum number of
  nodes. * If the cluster is not currently in a TERMINATED state, nothing will
  happen. * Clusters launched to run a job cannot be started.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The cluster to be started")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the cluster to be started")
		}
		startReq.ClusterId = args[0]

		if startSkipWait {
			err = w.Clusters.Start(ctx, startReq)
			if err != nil {
				return err
			}
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Clusters.StartAndWait(ctx, startReq,
			retries.Timeout[compute.ClusterInfo](startTimeout),
			func(i *retries.Info[compute.ClusterInfo]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.StateMessage
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start unpin command

var unpinReq compute.UnpinCluster

func init() {
	Cmd.AddCommand(unpinCmd)
	// TODO: short flags

}

var unpinCmd = &cobra.Command{
	Use:   "unpin CLUSTER_ID",
	Short: `Unpin cluster.`,
	Long: `Unpin cluster.
  
  Unpinning a cluster will allow the cluster to eventually be removed from the
  ListClusters API. Unpinning a cluster that is not pinned will have no effect.
  This API can only be called by workspace admins.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Clusters.ClusterInfoClusterNameToClusterIdMap(ctx, compute.ListClustersRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "<needs content added>")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have <needs content added>")
		}
		unpinReq.ClusterId = args[0]

		err = w.Clusters.Unpin(ctx, unpinReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Clusters
