// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package clusters

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
	clusters "github.com/databricks/cli/libs/sdktypes/clusters/v2"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkclusters "github.com/databricks/sdk-go/clusters/v2"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
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

  IMPORTANT: Databricks retains cluster configuration information for terminated
  clusters for 30 days. To keep an all-purpose cluster configuration even after
  it has been terminated for more than 30 days, an administrator can pin a
  cluster to the cluster list.`,
		GroupID: "compute",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newChangeOwner())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newEdit())
	cmd.AddCommand(newEvents())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListNodeTypes())
	cmd.AddCommand(newListZones())
	cmd.AddCommand(newPermanentDelete())
	cmd.AddCommand(newPin())
	cmd.AddCommand(newResize())
	cmd.AddCommand(newRestart())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newSparkVersions())
	cmd.AddCommand(newStart())
	cmd.AddCommand(newUnpin())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdatePermissions())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start change-owner command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var changeOwnerOverrides []func(
	*cobra.Command,
	*clusters.ChangeClusterOwnerRequest,
)

func newChangeOwner() *cobra.Command {
	cmd := &cobra.Command{}
	var changeOwnerReq clusters.ChangeClusterOwnerRequest

	var changeOwnerJson flags.JsonFlag

	cmd.Flags().Var(&changeOwnerJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "change-owner CLUSTER_ID OWNER_USERNAME"
	cmd.Short = `Change cluster owner.`
	cmd.Long = `Change cluster owner.

  Change the owner of the cluster. You must be an admin and the cluster must be
  terminated to perform this operation. The service principal application ID can
  be supplied as an argument to owner_username.

  Arguments:
    CLUSTER_ID:
    OWNER_USERNAME: New owner of the cluster_id after this RPC.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id', 'owner_username' in your JSON input")
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
			diags := changeOwnerJson.Unmarshal(&changeOwnerReq)
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
			changeOwnerReq.ClusterId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			changeOwnerReq.OwnerUsername = new(args[1])
		}

		sdkReq, err := clusters.ChangeClusterOwnerRequestToSDK(&changeOwnerReq)
		if err != nil {
			return err
		}
		response, err := w.Clusters.ChangeClusterOwner(ctx, *sdkReq)
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
	for _, fn := range changeOwnerOverrides {
		fn(cmd, &changeOwnerReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*clusters.CreateClusterRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq clusters.CreateClusterRequest

	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createReq.ApplyPolicyDefaultValues, "apply-policy-default-values", `When set to true, fixed and default values from the policy will be used for fields that are omitted.`)
	// TODO: complex arg: autoscale
	flags.IntPtrVar(cmd.Flags(), &createReq.AutoterminationMinutes, "autotermination-minutes", `Automatically terminates the cluster after it is inactive for this time in minutes.`)
	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: complex arg: clone_from
	// TODO: complex arg: cluster_log_conf
	flags.StringPtrVar(cmd.Flags(), &createReq.ClusterName, "cluster-name", `Cluster name requested by the user.`)
	// TODO: map via StringToStringVar: custom_tags
	flags.EnumVar(cmd.Flags(), &createReq.DataSecurityMode, "data-security-mode", `Supported values: [
  DATA_SECURITY_MODE_AUTO,
  DATA_SECURITY_MODE_DEDICATED,
  DATA_SECURITY_MODE_STANDARD,
  LEGACY_PASSTHROUGH,
  LEGACY_SINGLE_USER,
  LEGACY_SINGLE_USER_STANDARD,
  LEGACY_TABLE_ACL,
  NONE,
  SINGLE_USER,
  USER_ISOLATION,
]`, "DataSecurityMode", "DATA_SECURITY_MODE_AUTO", "DATA_SECURITY_MODE_DEDICATED", "DATA_SECURITY_MODE_STANDARD", "LEGACY_PASSTHROUGH", "LEGACY_SINGLE_USER", "LEGACY_SINGLE_USER_STANDARD", "LEGACY_TABLE_ACL", "NONE", "SINGLE_USER", "USER_ISOLATION")
	flags.EnumVar(cmd.Flags(), &createReq.DependencyMode, "dependency-mode", `Controls dependency configuration for the cluster. Supported values: [DEPENDENCY_MODE_AUTO, DEPENDENCY_MODE_CLUSTER_LIBRARIES, DEPENDENCY_MODE_ENVIRONMENTS]`, "DependencyMode", "DEPENDENCY_MODE_AUTO", "DEPENDENCY_MODE_CLUSTER_LIBRARIES", "DEPENDENCY_MODE_ENVIRONMENTS")
	// TODO: complex arg: docker_image
	flags.StringPtrVar(cmd.Flags(), &createReq.DriverInstancePoolId, "driver-instance-pool-id", `The optional ID of the instance pool for the driver of the cluster belongs.`)
	// TODO: complex arg: driver_node_type_flexibility
	flags.StringPtrVar(cmd.Flags(), &createReq.DriverNodeTypeId, "driver-node-type-id", `The node type of the Spark driver.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.EnableElasticDisk, "enable-elastic-disk", `Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.EnableLocalDiskEncryption, "enable-local-disk-encryption", `Whether to enable LUKS on cluster VMs' local disks.`)
	// TODO: complex arg: gcp_attributes
	// TODO: array: init_scripts
	flags.StringPtrVar(cmd.Flags(), &createReq.InstancePoolId, "instance-pool-id", `The optional ID of the instance pool to which the cluster belongs.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.IsSingleNode, "is-single-node", `This field can only be used when kind = CLASSIC_PREVIEW.`)
	flags.EnumVar(cmd.Flags(), &createReq.Kind, "kind", `Supported values: [CLASSIC_PREVIEW]`, "Kind", "CLASSIC_PREVIEW")
	flags.StringPtrVar(cmd.Flags(), &createReq.NodeTypeId, "node-type-id", `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.NumWorkers, "num-workers", `Number of worker nodes that this cluster should have.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.PolicyId, "policy-id", `The ID of the cluster policy used to create the cluster if applicable.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.RemoteDiskThroughput, "remote-disk-throughput", `If set, what the configurable throughput (in Mb/s) for the remote disk is.`)
	flags.EnumVar(cmd.Flags(), &createReq.RuntimeEngine, "runtime-engine", `Determines the cluster's runtime engine, either standard or Photon. Supported values: [NULL, PHOTON, STANDARD]`, "RuntimeEngine", "NULL", "PHOTON", "STANDARD")
	flags.StringPtrVar(cmd.Flags(), &createReq.SingleUserName, "single-user-name", `Single user name if data_security_mode is SINGLE_USER.`)
	// TODO: map via StringToStringVar: spark_conf
	// TODO: map via StringToStringVar: spark_env_vars
	// TODO: array: ssh_public_keys
	flags.IntPtrVar(cmd.Flags(), &createReq.TotalInitialRemoteDiskSize, "total-initial-remote-disk-size", `If set, what the total initial volume size (in GB) of the remote disks should be.`)
	flags.BoolPtrVar(cmd.Flags(), &createReq.UseMlRuntime, "use-ml-runtime", `This field can only be used when kind = CLASSIC_PREVIEW.`)
	// TODO: complex arg: worker_node_type_flexibility
	// TODO: complex arg: workload_type

	cmd.Use = "create SPARK_VERSION"
	cmd.Short = `Create new cluster.`
	cmd.Long = `Create new cluster.

  Creates a new Spark cluster. This method will acquire new instances from the
  cloud provider if necessary. This method is asynchronous; the returned
  cluster_id can be used to poll the cluster status. When this method
  returns, the cluster will be in a PENDING state. The cluster will be
  usable once it enters a RUNNING state. Note: Databricks may not be able to
  acquire some of the requested nodes, due to cloud provider limitations
  (account limits, spot price, etc.) or transient network issues.

  If Databricks acquires at least 85% of the requested on-demand nodes, cluster
  creation will succeed. Otherwise the cluster will terminate with an
  informative error message.

  Rather than authoring the cluster's JSON definition from scratch, Databricks
  recommends filling out the [create compute UI] and then copying the generated
  JSON definition from the UI.

  [create compute UI]: https://docs.databricks.com/compute/configure.html

  Arguments:
    SPARK_VERSION: The Spark version of the cluster, e.g. 3.3.x-scala2.11. A list of
      available Spark versions can be retrieved by using the
      [clusters/sparkVersions] API call.

      [clusters/sparkVersions]: https://docs.databricks.com/api/workspace/clusters/sparkversions`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'spark_version' in your JSON input")
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
			createReq.SparkVersion = new(args[0])
		}

		sdkReq, err := clusters.CreateClusterRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		wait, err := w.Clusters.CreateCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, map[string]any{"cluster_id": wait.GetClusterId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
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
	*clusters.DeleteClusterRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq clusters.DeleteClusterRequest

	var deleteJson flags.JsonFlag

	var deleteSkipWait bool
	var deleteTimeout time.Duration

	cmd.Flags().BoolVar(&deleteSkipWait, "no-wait", deleteSkipWait, `do not wait to reach TERMINATED state`)
	cmd.Flags().DurationVar(&deleteTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED state`)

	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete CLUSTER_ID"
	cmd.Short = `Terminate cluster.`
	cmd.Long = `Terminate cluster.

  Terminates the Spark cluster with the specified ID. The cluster is removed
  asynchronously. Once the termination has completed, the cluster will be in a
  TERMINATED state. If the cluster is already in a TERMINATING or
  TERMINATED state, nothing will happen.

  Arguments:
    CLUSTER_ID: The cluster to be terminated.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
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
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The cluster to be terminated")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have the cluster to be terminated")
			}
			deleteReq.ClusterId = new(args[0])
		}

		sdkReq, err := clusters.DeleteClusterRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		wait, err := w.Clusters.DeleteCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if deleteSkipWait {
			return cmdio.Render(ctx, map[string]any{"cluster_id": wait.GetClusterId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for delete to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(deleteTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*clusters.EditClusterRequest,
)

func newEdit() *cobra.Command {
	cmd := &cobra.Command{}
	var editReq clusters.EditClusterRequest

	var editJson flags.JsonFlag

	var editSkipWait bool
	var editTimeout time.Duration

	cmd.Flags().BoolVar(&editSkipWait, "no-wait", editSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&editTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&editJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &editReq.ApplyPolicyDefaultValues, "apply-policy-default-values", `When set to true, fixed and default values from the policy will be used for fields that are omitted.`)
	// TODO: complex arg: autoscale
	flags.IntPtrVar(cmd.Flags(), &editReq.AutoterminationMinutes, "autotermination-minutes", `Automatically terminates the cluster after it is inactive for this time in minutes.`)
	// TODO: complex arg: aws_attributes
	// TODO: complex arg: azure_attributes
	// TODO: complex arg: cluster_log_conf
	flags.StringPtrVar(cmd.Flags(), &editReq.ClusterName, "cluster-name", `Cluster name requested by the user.`)
	// TODO: map via StringToStringVar: custom_tags
	flags.EnumVar(cmd.Flags(), &editReq.DataSecurityMode, "data-security-mode", `Supported values: [
  DATA_SECURITY_MODE_AUTO,
  DATA_SECURITY_MODE_DEDICATED,
  DATA_SECURITY_MODE_STANDARD,
  LEGACY_PASSTHROUGH,
  LEGACY_SINGLE_USER,
  LEGACY_SINGLE_USER_STANDARD,
  LEGACY_TABLE_ACL,
  NONE,
  SINGLE_USER,
  USER_ISOLATION,
]`, "DataSecurityMode", "DATA_SECURITY_MODE_AUTO", "DATA_SECURITY_MODE_DEDICATED", "DATA_SECURITY_MODE_STANDARD", "LEGACY_PASSTHROUGH", "LEGACY_SINGLE_USER", "LEGACY_SINGLE_USER_STANDARD", "LEGACY_TABLE_ACL", "NONE", "SINGLE_USER", "USER_ISOLATION")
	flags.EnumVar(cmd.Flags(), &editReq.DependencyMode, "dependency-mode", `Controls dependency configuration for the cluster. Supported values: [DEPENDENCY_MODE_AUTO, DEPENDENCY_MODE_CLUSTER_LIBRARIES, DEPENDENCY_MODE_ENVIRONMENTS]`, "DependencyMode", "DEPENDENCY_MODE_AUTO", "DEPENDENCY_MODE_CLUSTER_LIBRARIES", "DEPENDENCY_MODE_ENVIRONMENTS")
	// TODO: complex arg: docker_image
	flags.StringPtrVar(cmd.Flags(), &editReq.DriverInstancePoolId, "driver-instance-pool-id", `The optional ID of the instance pool for the driver of the cluster belongs.`)
	// TODO: complex arg: driver_node_type_flexibility
	flags.StringPtrVar(cmd.Flags(), &editReq.DriverNodeTypeId, "driver-node-type-id", `The node type of the Spark driver.`)
	flags.BoolPtrVar(cmd.Flags(), &editReq.EnableElasticDisk, "enable-elastic-disk", `Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk space when its Spark workers are running low on disk space.`)
	flags.BoolPtrVar(cmd.Flags(), &editReq.EnableLocalDiskEncryption, "enable-local-disk-encryption", `Whether to enable LUKS on cluster VMs' local disks.`)
	// TODO: complex arg: gcp_attributes
	// TODO: array: init_scripts
	flags.StringPtrVar(cmd.Flags(), &editReq.InstancePoolId, "instance-pool-id", `The optional ID of the instance pool to which the cluster belongs.`)
	flags.BoolPtrVar(cmd.Flags(), &editReq.IsSingleNode, "is-single-node", `This field can only be used when kind = CLASSIC_PREVIEW.`)
	flags.EnumVar(cmd.Flags(), &editReq.Kind, "kind", `Supported values: [CLASSIC_PREVIEW]`, "Kind", "CLASSIC_PREVIEW")
	flags.StringPtrVar(cmd.Flags(), &editReq.NodeTypeId, "node-type-id", `This field encodes, through a single value, the resources available to each of the Spark nodes in this cluster.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.NumWorkers, "num-workers", `Number of worker nodes that this cluster should have.`)
	flags.StringPtrVar(cmd.Flags(), &editReq.PolicyId, "policy-id", `The ID of the cluster policy used to create the cluster if applicable.`)
	flags.IntPtrVar(cmd.Flags(), &editReq.RemoteDiskThroughput, "remote-disk-throughput", `If set, what the configurable throughput (in Mb/s) for the remote disk is.`)
	flags.EnumVar(cmd.Flags(), &editReq.RuntimeEngine, "runtime-engine", `Determines the cluster's runtime engine, either standard or Photon. Supported values: [NULL, PHOTON, STANDARD]`, "RuntimeEngine", "NULL", "PHOTON", "STANDARD")
	flags.StringPtrVar(cmd.Flags(), &editReq.SingleUserName, "single-user-name", `Single user name if data_security_mode is SINGLE_USER.`)
	// TODO: map via StringToStringVar: spark_conf
	// TODO: map via StringToStringVar: spark_env_vars
	// TODO: array: ssh_public_keys
	flags.IntPtrVar(cmd.Flags(), &editReq.TotalInitialRemoteDiskSize, "total-initial-remote-disk-size", `If set, what the total initial volume size (in GB) of the remote disks should be.`)
	flags.BoolPtrVar(cmd.Flags(), &editReq.UseMlRuntime, "use-ml-runtime", `This field can only be used when kind = CLASSIC_PREVIEW.`)
	// TODO: complex arg: worker_node_type_flexibility
	// TODO: complex arg: workload_type

	cmd.Use = "edit CLUSTER_ID SPARK_VERSION"
	cmd.Short = `Update cluster configuration.`
	cmd.Long = `Update cluster configuration.

  Updates the configuration of a cluster to match the provided attributes and
  size. A cluster can be updated if it is in a RUNNING or TERMINATED state.

  If a cluster is updated while in a RUNNING state, it will be restarted so
  that the new attributes can take effect.

  If a cluster is updated while in a TERMINATED state, it will remain
  TERMINATED. The next time it is started using the clusters/start API, the
  new attributes will take effect. Any attempt to update a cluster in any other
  state will be rejected with an INVALID_STATE error code.

  Clusters created by the Databricks Jobs service cannot be edited.

  Arguments:
    CLUSTER_ID: ID of the cluster
    SPARK_VERSION: The Spark version of the cluster, e.g. 3.3.x-scala2.11. A list of
      available Spark versions can be retrieved by using the
      [clusters/sparkVersions] API call.

      [clusters/sparkVersions]: https://docs.databricks.com/api/workspace/clusters/sparkversions`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id', 'spark_version' in your JSON input")
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
			editReq.ClusterId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			editReq.SparkVersion = new(args[1])
		}

		sdkReq, err := clusters.EditClusterRequestToSDK(&editReq)
		if err != nil {
			return err
		}
		wait, err := w.Clusters.EditCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if editSkipWait {
			return cmdio.Render(ctx, map[string]any{"cluster_id": wait.GetClusterId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for edit to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(editTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
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

// start events command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var eventsOverrides []func(
	*cobra.Command,
	*clusters.ListEventsRequest,
)

func newEvents() *cobra.Command {
	cmd := &cobra.Command{}
	var eventsReq clusters.ListEventsRequest

	var eventsJson flags.JsonFlag
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var eventsLimit int

	cmd.Flags().Var(&eventsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Int64PtrVar(cmd.Flags(), &eventsReq.EndTime, "end-time", `The end time in epoch milliseconds.`)
	// TODO: array: event_types
	flags.EnumVar(cmd.Flags(), &eventsReq.Order, "order", `The order to list events in; either "ASC" or "DESC". Supported values: [ASC, DESC]`, "GetEventsOrder", "ASC", "DESC")
	flags.IntPtrVar(cmd.Flags(), &eventsReq.PageSize, "page-size", `The maximum number of events to include in a page of events.`)
	flags.StringPtrVar(cmd.Flags(), &eventsReq.PageToken, "page-token", `Use next_page_token or prev_page_token returned from the previous request to list the next or previous page of events respectively.`)
	flags.Int64PtrVar(cmd.Flags(), &eventsReq.StartTime, "start-time", `The start time in epoch milliseconds.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&eventsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.Int64PtrVar(cmd.Flags(), &eventsReq.Offset, "offset", `Deprecated: use page_token in combination with page_size instead.`)
	cmd.Flags().Lookup("offset").Hidden = true

	cmd.Use = "events CLUSTER_ID"
	cmd.Short = `List cluster activity events.`
	cmd.Long = `List cluster activity events.

  Retrieves a list of events about the activity of a cluster. This API is
  paginated. If there are more events to read, the response includes all the
  parameters necessary to request the next page of events.

  Arguments:
    CLUSTER_ID: The ID of the cluster to retrieve events about.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := eventsJson.Unmarshal(&eventsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID of the cluster to retrieve events about")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have the id of the cluster to retrieve events about")
			}
			eventsReq.ClusterId = new(args[0])

		}

		sdkReq, err := clusters.ListEventsRequestToSDK(&eventsReq)
		if err != nil {
			return err
		}
		if eventsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", eventsLimit)
		}
		if eventsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, eventsLimit)
		}
		response := w.Clusters.ListEventsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, clusters.ClusterEventFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range eventsOverrides {
		fn(cmd, &eventsReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*clusters.GetClusterRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq clusters.GetClusterRequest

	cmd.Use = "get CLUSTER_ID"
	cmd.Short = `Get cluster info.`
	cmd.Long = `Get cluster info.

  Retrieves the information for a cluster given its identifier. Clusters can be
  described while they are running, or up to 60 days after they are terminated.

  Arguments:
    CLUSTER_ID: The cluster about which to retrieve information.`

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
			sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster about which to retrieve information")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have the cluster about which to retrieve information")
		}
		getReq.ClusterId = new(args[0])

		sdkReq, err := clusters.GetClusterRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.Clusters.GetCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
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

	cmd.Use = "get-permission-levels CLUSTER_ID"
	cmd.Short = `Get cluster permission levels.`
	cmd.Long = `Get cluster permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    CLUSTER_ID: The cluster for which to get or manage permissions.`

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
			sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have the cluster for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: cluster_id

		getPermissionLevelsReq.RequestObjectType = new("clusters")

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

	cmd.Use = "get-permissions CLUSTER_ID"
	cmd.Short = `Get cluster permissions.`
	cmd.Long = `Get cluster permissions.

  Gets the permissions of a cluster. Clusters can inherit permissions from their
  root object.

  Arguments:
    CLUSTER_ID: The cluster for which to get or manage permissions.`

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
			sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have the cluster for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: cluster_id

		getPermissionsReq.RequestObjectType = new("clusters")

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
	*clusters.ListClustersRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq clusters.ListClustersRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	// TODO: complex arg: filter_by
	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `Use this field to specify the maximum number of results to be returned by the server.`)
	// TODO: complex arg: sort_by

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List clusters.`
	cmd.Long = `List clusters.

  Return information about all pinned and active clusters, and all clusters
  terminated within the last 30 days. Clusters terminated prior to this period
  are not included.`

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

		sdkReq, err := clusters.ListClustersRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Clusters.ListClustersIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, clusters.ClusterInfoFromSDK))
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

// start list-node-types command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listNodeTypesOverrides []func(
	*cobra.Command,
)

func newListNodeTypes() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list-node-types"
	cmd.Short = `List node types.`
	cmd.Long = `List node types.

  Returns a list of supported Spark node types. These node types can be used to
  launch a cluster.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkclusters.ListNodeTypesRequest{}
		response, err := w.Clusters.ListNodeTypes(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := clusters.ListNodeTypesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listNodeTypesOverrides {
		fn(cmd)
	}

	return cmd
}

// start list-zones command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listZonesOverrides []func(
	*cobra.Command,
)

func newListZones() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list-zones"
	cmd.Short = `List availability zones.`
	cmd.Long = `List availability zones.

  Returns a list of availability zones where clusters can be created in (For
  example, us-west-2a). These zones can be used to launch a cluster.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkclusters.ListAvailableZonesRequest{}
		response, err := w.Clusters.ListAvailableZones(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := clusters.ListAvailableZonesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listZonesOverrides {
		fn(cmd)
	}

	return cmd
}

// start permanent-delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var permanentDeleteOverrides []func(
	*cobra.Command,
	*clusters.PermanentDeleteClusterRequest,
)

func newPermanentDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var permanentDeleteReq clusters.PermanentDeleteClusterRequest

	var permanentDeleteJson flags.JsonFlag

	cmd.Flags().Var(&permanentDeleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "permanent-delete CLUSTER_ID"
	cmd.Short = `Permanently delete cluster.`
	cmd.Long = `Permanently delete cluster.

  Permanently deletes a Spark cluster. This cluster is terminated and resources
  are asynchronously removed.

  In addition, users will no longer see permanently deleted clusters in the
  cluster list, and API users can no longer perform any action on permanently
  deleted clusters.

  Arguments:
    CLUSTER_ID: The cluster to be deleted.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := permanentDeleteJson.Unmarshal(&permanentDeleteReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The cluster to be deleted")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have the cluster to be deleted")
			}
			permanentDeleteReq.ClusterId = new(args[0])
		}

		sdkReq, err := clusters.PermanentDeleteClusterRequestToSDK(&permanentDeleteReq)
		if err != nil {
			return err
		}
		response, err := w.Clusters.PermanentDeleteCluster(ctx, *sdkReq)
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
	for _, fn := range permanentDeleteOverrides {
		fn(cmd, &permanentDeleteReq)
	}

	return cmd
}

// start pin command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var pinOverrides []func(
	*cobra.Command,
	*clusters.PinClusterRequest,
)

func newPin() *cobra.Command {
	cmd := &cobra.Command{}
	var pinReq clusters.PinClusterRequest

	var pinJson flags.JsonFlag

	cmd.Flags().Var(&pinJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "pin CLUSTER_ID"
	cmd.Short = `Pin cluster.`
	cmd.Long = `Pin cluster.

  Pinning a cluster ensures that the cluster will always be returned by the
  ListClusters API. Pinning a cluster that is already pinned will have no
  effect. This API can only be called by workspace admins.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := pinJson.Unmarshal(&pinReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have ")
			}
			pinReq.ClusterId = new(args[0])
		}

		sdkReq, err := clusters.PinClusterRequestToSDK(&pinReq)
		if err != nil {
			return err
		}
		response, err := w.Clusters.PinCluster(ctx, *sdkReq)
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
	for _, fn := range pinOverrides {
		fn(cmd, &pinReq)
	}

	return cmd
}

// start resize command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resizeOverrides []func(
	*cobra.Command,
	*clusters.ResizeClusterRequest,
)

func newResize() *cobra.Command {
	cmd := &cobra.Command{}
	var resizeReq clusters.ResizeClusterRequest

	var resizeJson flags.JsonFlag

	var resizeSkipWait bool
	var resizeTimeout time.Duration

	cmd.Flags().BoolVar(&resizeSkipWait, "no-wait", resizeSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&resizeTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&resizeJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: autoscale
	flags.IntPtrVar(cmd.Flags(), &resizeReq.NumWorkers, "num-workers", `Number of worker nodes that this cluster should have.`)

	cmd.Use = "resize CLUSTER_ID"
	cmd.Short = `Resize cluster.`
	cmd.Long = `Resize cluster.

  Resizes a cluster to have a desired number of workers. This will fail unless
  the cluster is in a RUNNING state.

  Arguments:
    CLUSTER_ID: The cluster to be resized.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := resizeJson.Unmarshal(&resizeReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The cluster to be resized")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have the cluster to be resized")
			}
			resizeReq.ClusterId = new(args[0])

		}

		sdkReq, err := clusters.ResizeClusterRequestToSDK(&resizeReq)
		if err != nil {
			return err
		}
		wait, err := w.Clusters.ResizeCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if resizeSkipWait {
			return cmdio.Render(ctx, map[string]any{"cluster_id": wait.GetClusterId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for resize to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(resizeTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resizeOverrides {
		fn(cmd, &resizeReq)
	}

	return cmd
}

// start restart command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restartOverrides []func(
	*cobra.Command,
	*clusters.RestartClusterRequest,
)

func newRestart() *cobra.Command {
	cmd := &cobra.Command{}
	var restartReq clusters.RestartClusterRequest

	var restartJson flags.JsonFlag

	var restartSkipWait bool
	var restartTimeout time.Duration

	cmd.Flags().BoolVar(&restartSkipWait, "no-wait", restartSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&restartTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&restartJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &restartReq.RestartUser, "restart-user", ``)

	cmd.Use = "restart CLUSTER_ID"
	cmd.Short = `Restart cluster.`
	cmd.Long = `Restart cluster.

  Restarts a Spark cluster with the supplied ID. If the cluster is not currently
  in a RUNNING state, nothing will happen.

  Arguments:
    CLUSTER_ID: The cluster to be started.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := restartJson.Unmarshal(&restartReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The cluster to be started")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have the cluster to be started")
			}
			restartReq.ClusterId = new(args[0])

		}

		sdkReq, err := clusters.RestartClusterRequestToSDK(&restartReq)
		if err != nil {
			return err
		}
		wait, err := w.Clusters.RestartCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if restartSkipWait {
			return cmdio.Render(ctx, map[string]any{"cluster_id": wait.GetClusterId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for restart to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(restartTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range restartOverrides {
		fn(cmd, &restartReq)
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

	cmd.Use = "set-permissions CLUSTER_ID"
	cmd.Short = `Set cluster permissions.`
	cmd.Long = `Set cluster permissions.

  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    CLUSTER_ID: The cluster for which to get or manage permissions.`

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
			sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have the cluster for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: cluster_id

		setPermissionsReq.RequestObjectType = new("clusters")

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

// start spark-versions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var sparkVersionsOverrides []func(
	*cobra.Command,
)

func newSparkVersions() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "spark-versions"
	cmd.Short = `List available Spark versions.`
	cmd.Long = `List available Spark versions.

  Returns the list of available Spark versions. These versions can be used to
  launch a cluster.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkclusters.GetSparkVersionsRequest{}
		response, err := w.Clusters.ListSparkVersions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := clusters.GetSparkVersionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range sparkVersionsOverrides {
		fn(cmd)
	}

	return cmd
}

// start start command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startOverrides []func(
	*cobra.Command,
	*clusters.StartClusterRequest,
)

func newStart() *cobra.Command {
	cmd := &cobra.Command{}
	var startReq clusters.StartClusterRequest

	var startJson flags.JsonFlag

	var startSkipWait bool
	var startTimeout time.Duration

	cmd.Flags().BoolVar(&startSkipWait, "no-wait", startSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&startTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&startJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "start CLUSTER_ID"
	cmd.Short = `Start terminated cluster.`
	cmd.Long = `Start terminated cluster.

  Starts a terminated Spark cluster with the supplied ID. This works similar to
  createCluster except: - The previous cluster id and attributes are
  preserved. - The cluster starts with the last specified cluster size. - If the
  previous cluster was an autoscaling cluster, the current cluster starts with
  the minimum number of nodes. - If the cluster is not currently in a
  TERMINATED state, nothing will happen. - Clusters launched to run a job
  cannot be started.

  Arguments:
    CLUSTER_ID: The cluster to be started.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := startJson.Unmarshal(&startReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The cluster to be started")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have the cluster to be started")
			}
			startReq.ClusterId = new(args[0])
		}

		sdkReq, err := clusters.StartClusterRequestToSDK(&startReq)
		if err != nil {
			return err
		}
		wait, err := w.Clusters.StartCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if startSkipWait {
			return cmdio.Render(ctx, map[string]any{"cluster_id": wait.GetClusterId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for start to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(startTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
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

// start unpin command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var unpinOverrides []func(
	*cobra.Command,
	*clusters.UnpinClusterRequest,
)

func newUnpin() *cobra.Command {
	cmd := &cobra.Command{}
	var unpinReq clusters.UnpinClusterRequest

	var unpinJson flags.JsonFlag

	cmd.Flags().Var(&unpinJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "unpin CLUSTER_ID"
	cmd.Short = `Unpin cluster.`
	cmd.Long = `Unpin cluster.

  Unpinning a cluster will allow the cluster to eventually be removed from the
  ListClusters API. Unpinning a cluster that is not pinned will have no effect.
  This API can only be called by workspace admins.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := unpinJson.Unmarshal(&unpinReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
				sp.Close()
				if err != nil {
					return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return errors.New("expected to have ")
			}
			unpinReq.ClusterId = new(args[0])
		}

		sdkReq, err := clusters.UnpinClusterRequestToSDK(&unpinReq)
		if err != nil {
			return err
		}
		response, err := w.Clusters.UnpinCluster(ctx, *sdkReq)
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
	for _, fn := range unpinOverrides {
		fn(cmd, &unpinReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*clusters.UpdateClusterRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq clusters.UpdateClusterRequest

	var updateJson flags.JsonFlag

	var updateSkipWait bool
	var updateTimeout time.Duration

	cmd.Flags().BoolVar(&updateSkipWait, "no-wait", updateSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&updateTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: cluster

	cmd.Use = "update CLUSTER_ID UPDATE_MASK"
	cmd.Short = `Update cluster configuration (partial).`
	cmd.Long = `Update cluster configuration (partial).

  Updates the configuration of a cluster to match the partial set of attributes
  and size. Denote which fields to update using the update_mask field in the
  request body. A cluster can be updated if it is in a RUNNING or TERMINATED
  state. If a cluster is updated while in a RUNNING state, it will be
  restarted so that the new attributes can take effect. If a cluster is updated
  while in a TERMINATED state, it will remain TERMINATED. The updated
  attributes will take effect the next time the cluster is started using the
  clusters/start API. Attempts to update a cluster in any other state will be
  rejected with an INVALID_STATE error code. Clusters created by the
  Databricks Jobs service cannot be updated.

  Arguments:
    CLUSTER_ID: ID of the cluster.
    UPDATE_MASK: Used to specify which cluster attributes and size fields to update. See
      https://google.aip.dev/161 for more details.

      The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'cluster_id', 'update_mask' in your JSON input")
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
			diags := updateJson.Unmarshal(&updateReq)
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
			updateReq.ClusterId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			updateReq.UpdateMask = new(args[1])
		}

		sdkReq, err := clusters.UpdateClusterRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		wait, err := w.Clusters.UpdateCluster(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if updateSkipWait {
			return cmdio.Render(ctx, map[string]any{"cluster_id": wait.GetClusterId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for update to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(updateTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := clusters.ClusterInfoFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
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

	cmd.Use = "update-permissions CLUSTER_ID"
	cmd.Short = `Update cluster permissions.`
	cmd.Long = `Update cluster permissions.

  Updates the permissions on a cluster. Clusters can inherit permissions from
  their root object.

  Arguments:
    CLUSTER_ID: The cluster for which to get or manage permissions.`

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
			sp.Update("No CLUSTER_ID argument specified. Loading names for Clusters drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Clusters.ListClustersIter(ctx, sdkclusters.ListClustersRequest{}), []string{"ClusterName"}, []string{"ClusterId"})
			sp.Close()
			if err != nil {
				return fmt.Errorf("failed to load names for Clusters drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The cluster for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return errors.New("expected to have the cluster for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: cluster_id

		updatePermissionsReq.RequestObjectType = new("clusters")

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

// end service Clusters
