package dresources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

// clusterWaitTimeout bounds how long we poll for a cluster to reach its target
// state (RUNNING/TERMINATED) after create, edit, or start/stop. Provisioning can
// legitimately take longer than 15 minutes on capacity-constrained workspaces
// (a cluster stays PENDING with "Finding instances for new nodes"), so we allow
// 30 minutes before giving up. Terminal states still halt immediately.
const clusterWaitTimeout = 30 * time.Minute

// ClusterState is the state type for Cluster resources. It extends compute.ClusterSpec with
// lifecycle settings and the cluster ID.
// ClusterId is written to state by DoCreate/DoUpdate for informational purposes; it is not
// used in diff computation (neither PrepareState nor RemapState set it).
type ClusterState struct {
	compute.ClusterSpec

	Lifecycle *StateLifecycle `json:"lifecycle,omitempty"`
}

// Custom marshalers needed because embedded compute.ClusterSpec has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *ClusterState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s ClusterState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// ClusterRemote extends compute.ClusterDetails with a synthetic Lifecycle field so that
// RemoteType satisfies TestRemoteSuperset (every field in ClusterState exists in ClusterRemote).
// Lifecycle.Started is populated by DoRead from the cluster's running state.
//
// ApplyPolicyDefaultValues is promoted to the top level because the cluster GET API returns it
// only under .spec (a snapshot of the create/edit settings), never at the top level. DoRead
// copies it up so RemapState stays a dumb copy and the field participates in normal drift
// detection instead of being suppressed as missing_in_remote.
type ClusterRemote struct {
	compute.ClusterDetails
	ApplyPolicyDefaultValues bool            `json:"apply_policy_default_values,omitempty"`
	Lifecycle                *StateLifecycle `json:"lifecycle,omitempty"`
}

func (r *ClusterRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r ClusterRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

type ResourceCluster struct {
	client *databricks.WorkspaceClient
}

func (r *ResourceCluster) New(client *databricks.WorkspaceClient) any {
	return &ResourceCluster{
		client: client,
	}
}

func (r *ResourceCluster) PrepareState(input *resources.Cluster) *ClusterState {
	s := &ClusterState{
		ClusterSpec: input.ClusterSpec,
		Lifecycle:   nil,
	}
	if input.Lifecycle != nil && input.Lifecycle.Started != nil {
		s.Lifecycle = &StateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote ClusterRemote to ClusterState for diff comparison.
// Started is derived from cluster state so the planner can detect start/stop changes.
func (r *ResourceCluster) RemapState(input *ClusterRemote) *ClusterState {
	started := input.State == compute.StateRunning
	spec := &ClusterState{
		ClusterSpec: compute.ClusterSpec{
			ApplyPolicyDefaultValues:   input.ApplyPolicyDefaultValues,
			Autoscale:                  input.Autoscale,
			AutoterminationMinutes:     input.AutoterminationMinutes,
			AwsAttributes:              input.AwsAttributes,
			AzureAttributes:            input.AzureAttributes,
			ClusterLogConf:             input.ClusterLogConf,
			ClusterName:                input.ClusterName,
			CustomTags:                 input.CustomTags,
			DataSecurityMode:           input.DataSecurityMode,
			DependencyMode:             input.DependencyMode,
			DockerImage:                input.DockerImage,
			DriverInstancePoolId:       input.DriverInstancePoolId,
			DriverNodeTypeId:           input.DriverNodeTypeId,
			DriverNodeTypeFlexibility:  input.DriverNodeTypeFlexibility,
			EnableElasticDisk:          input.EnableElasticDisk,
			EnableLocalDiskEncryption:  input.EnableLocalDiskEncryption,
			GcpAttributes:              input.GcpAttributes,
			InitScripts:                input.InitScripts,
			InstancePoolId:             input.InstancePoolId,
			IsSingleNode:               input.IsSingleNode,
			Kind:                       input.Kind,
			NodeTypeId:                 input.NodeTypeId,
			NumWorkers:                 input.NumWorkers,
			PolicyId:                   input.PolicyId,
			RemoteDiskThroughput:       input.RemoteDiskThroughput,
			RuntimeEngine:              input.RuntimeEngine,
			SingleUserName:             input.SingleUserName,
			SparkConf:                  input.SparkConf,
			SparkEnvVars:               input.SparkEnvVars,
			SparkVersion:               input.SparkVersion,
			SshPublicKeys:              input.SshPublicKeys,
			TotalInitialRemoteDiskSize: input.TotalInitialRemoteDiskSize,
			UseMlRuntime:               input.UseMlRuntime,
			WorkloadType:               input.WorkloadType,
			WorkerNodeTypeFlexibility:  input.WorkerNodeTypeFlexibility,
			ForceSendFields:            utils.FilterFields[compute.ClusterSpec](input.ForceSendFields),
		},
		Lifecycle: &StateLifecycle{Started: &started},
	}
	return spec
}

func (r *ResourceCluster) DoRead(ctx context.Context, id string) (*ClusterRemote, error) {
	details, err := r.client.Clusters.GetByClusterId(ctx, id)
	if err != nil {
		return nil, err
	}
	remote := &ClusterRemote{
		ClusterDetails:           *details,
		ApplyPolicyDefaultValues: false,
		Lifecycle:                nil,
	}
	// The GET response carries apply_policy_default_values only under .spec (a snapshot of the
	// create/edit settings), not at the top level. Promote it so RemapState is a dumb copy.
	if details.Spec != nil {
		remote.ApplyPolicyDefaultValues = details.Spec.ApplyPolicyDefaultValues
	}

	switch details.State {
	case compute.StateRunning:
		started := true
		remote.Lifecycle = &StateLifecycle{Started: &started}
	case compute.StateTerminated:
		started := false
		remote.Lifecycle = &StateLifecycle{Started: &started}
	default:
		remote.Lifecycle = nil
	}
	return remote, nil
}

func (r *ResourceCluster) DoCreate(ctx context.Context, engine *Engine, config *ClusterState) (string, *ClusterRemote, error) {
	wait, err := r.client.Clusters.Create(ctx, makeCreateCluster(&config.ClusterSpec))
	if err != nil {
		return "", nil, err
	}
	id := wait.ClusterId

	// Save state immediately after the cluster is created so it is not orphaned
	// if the subsequent wait or terminate is interrupted.
	if err := engine.SaveState(id, config); err != nil {
		return "", nil, err
	}

	// Always wait for RUNNING first: clusters start in PENDING state and must be polled.
	_, err = r.client.Clusters.WaitGetClusterRunning(ctx, id, clusterWaitTimeout, nil)
	if err != nil {
		return "", nil, err
	}

	if config.Lifecycle != nil && config.Lifecycle.Started != nil && !*config.Lifecycle.Started {
		// started=false: terminate the cluster after it reaches RUNNING.
		// Note: Delete terminates the cluster; permanent removal is a separate API (permanent-delete).
		deleteWaiter, err := r.client.Clusters.Delete(ctx, compute.DeleteCluster{ClusterId: id})
		if err != nil {
			return "", nil, err
		}
		_, err = deleteWaiter.GetWithTimeout(clusterWaitTimeout)
		if err != nil {
			return "", nil, err
		}
	}

	return id, nil, nil
}

// hasClusterChanges reports whether the plan entry contains any Update changes
// to fields that belong to the Cluster Edit API (i.e., not lifecycle-only fields).
func hasClusterChanges(entry *PlanEntry) bool {
	return entry.Changes.HasChangeExcept("lifecycle", "lifecycle.started")
}

func (r *ResourceCluster) DoUpdate(ctx context.Context, _ *Engine, id string, config *ClusterState, entry *PlanEntry) (*ClusterRemote, error) {
	if hasClusterChanges(entry) {
		// Same retry as in TF provider logic
		// https://github.com/databricks/terraform-provider-databricks/blob/3eecd0f90cf99d7777e79a3d03c41f9b2aafb004/clusters/resource_cluster.go#L624
		_, err := retries.Poll(ctx, clusterWaitTimeout, func() (*compute.WaitGetClusterRunning[struct{}], *retries.Err) {
			wait, err := r.client.Clusters.Edit(ctx, makeEditCluster(id, &config.ClusterSpec))
			if err == nil {
				return wait, nil
			}

			// Only Running and Terminated clusters can be modified. In particular, autoscaling clusters cannot be modified
			// while the resizing is ongoing. We retry in this case. Scaling can take several minutes.
			if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.ErrorCode == "INVALID_STATE" {
				return nil, retries.Continues(fmt.Sprintf("cluster %s cannot be modified in its current state: %s", id, apiErr.Message))
			}
			return nil, retries.Halt(err)
		})
		if err != nil {
			return nil, err
		}
	}

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	desiredStarted := *config.Lifecycle.Started
	alreadyRunning := remoteClusterIsRunning(entry)
	if desiredStarted && !alreadyRunning {
		// lifecycle.started=true: fire Start and wait for RUNNING.
		_, err := r.client.Clusters.Start(ctx, compute.StartCluster{ClusterId: id})
		if err != nil {
			return nil, err
		}
		_, err = r.client.Clusters.WaitGetClusterRunning(ctx, id, clusterWaitTimeout, nil)
		return nil, err
	} else if !desiredStarted && alreadyRunning {
		// lifecycle.started=false: fire Delete and wait for TERMINATED.
		// Note: Delete terminates the cluster; permanent removal is a separate API (permanent-delete).
		_, err := r.client.Clusters.Delete(ctx, compute.DeleteCluster{ClusterId: id})
		if err != nil {
			return nil, err
		}
		_, err = r.client.Clusters.WaitGetClusterTerminated(ctx, id, clusterWaitTimeout, nil)
		return nil, err
	}

	return nil, nil
}

func (r *ResourceCluster) DoResize(ctx context.Context, id string, config *ClusterState, entry *PlanEntry) error {
	_, err := r.client.Clusters.Resize(ctx, compute.ResizeCluster{
		ClusterId:       id,
		NumWorkers:      config.NumWorkers,
		Autoscale:       config.Autoscale,
		ForceSendFields: utils.FilterFields[compute.ResizeCluster](config.ForceSendFields),
	})
	if err == nil {
		return nil
	}

	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || apiErr.ErrorCode != "INVALID_STATE" {
		return err
	}

	// Cluster is not running; fall back to the full clusters/edit path.
	// DoUpdate ignores its Engine argument, so passing nil here is safe.
	log.Debugf(ctx, "cluster %s: resize returned INVALID_STATE (%s), falling back to edit", id, err)
	_, err = r.DoUpdate(ctx, nil, id, config, entry)
	return err
}

func (r *ResourceCluster) DoDelete(ctx context.Context, id string, _ *ClusterState) error {
	return r.client.Clusters.PermanentDeleteByClusterId(ctx, id)
}

func (r *ResourceCluster) OverrideChangeDesc(ctx context.Context, p *structpath.PathNode, change *ChangeDesc, remoteState *ClusterRemote) error {
	path := p.Prefix(1).String()

	// Remaining overrides only apply to Update actions.
	if change.Action != deployplan.Update {
		return nil
	}

	switch path {
	case "data_security_mode":
		// We do change skip here in the same way TF provider does suppress diff if the alias is used.
		// https://github.com/databricks/terraform-provider-databricks/blob/main/clusters/resource_cluster.go#L109-L117
		if change.New == compute.DataSecurityModeDataSecurityModeStandard && change.Remote == compute.DataSecurityModeUserIsolation && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeDedicated && change.Remote == compute.DataSecurityModeSingleUser && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeAuto && (change.Remote == compute.DataSecurityModeSingleUser || change.Remote == compute.DataSecurityModeUserIsolation) && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		}

	case "num_workers", "autoscale":
		if remoteState != nil && remoteState.State == compute.StateRunning {
			change.Action = deployplan.Resize
		}
	}
	return nil
}

// remoteClusterIsRunning reads the cluster running state from the plan entry's remote state.
func remoteClusterIsRunning(entry *PlanEntry) bool {
	if entry.RemoteState == nil {
		return false
	}
	remote, ok := entry.RemoteState.(*ClusterRemote)
	if !ok {
		return false
	}
	return remote.State == compute.StateRunning
}

func makeCreateCluster(config *compute.ClusterSpec) compute.CreateCluster {
	create := compute.CreateCluster{
		ApplyPolicyDefaultValues:   config.ApplyPolicyDefaultValues,
		Autoscale:                  config.Autoscale,
		AutoterminationMinutes:     config.AutoterminationMinutes,
		AwsAttributes:              config.AwsAttributes,
		AzureAttributes:            config.AzureAttributes,
		ClusterLogConf:             config.ClusterLogConf,
		ClusterName:                config.ClusterName,
		CloneFrom:                  nil, // Not supported by DABs
		CustomTags:                 config.CustomTags,
		DataSecurityMode:           config.DataSecurityMode,
		DependencyMode:             config.DependencyMode,
		DockerImage:                config.DockerImage,
		DriverInstancePoolId:       config.DriverInstancePoolId,
		DriverNodeTypeId:           config.DriverNodeTypeId,
		DriverNodeTypeFlexibility:  config.DriverNodeTypeFlexibility,
		EnableElasticDisk:          config.EnableElasticDisk,
		EnableLocalDiskEncryption:  config.EnableLocalDiskEncryption,
		GcpAttributes:              config.GcpAttributes,
		InitScripts:                config.InitScripts,
		InstancePoolId:             config.InstancePoolId,
		IsSingleNode:               config.IsSingleNode,
		Kind:                       config.Kind,
		NodeTypeId:                 config.NodeTypeId,
		NumWorkers:                 config.NumWorkers,
		PolicyId:                   config.PolicyId,
		RemoteDiskThroughput:       config.RemoteDiskThroughput,
		RuntimeEngine:              config.RuntimeEngine,
		SingleUserName:             config.SingleUserName,
		SparkConf:                  config.SparkConf,
		SparkEnvVars:               config.SparkEnvVars,
		SparkVersion:               config.SparkVersion,
		SshPublicKeys:              config.SshPublicKeys,
		TotalInitialRemoteDiskSize: config.TotalInitialRemoteDiskSize,
		UseMlRuntime:               config.UseMlRuntime,
		WorkloadType:               config.WorkloadType,
		WorkerNodeTypeFlexibility:  config.WorkerNodeTypeFlexibility,
		ForceSendFields:            utils.FilterFields[compute.CreateCluster](config.ForceSendFields),
	}

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	// If NumWorkers is not nil, we don't need to set it to ForceSendFields as it will be sent anyway.
	if config.Autoscale == nil && config.NumWorkers == 0 {
		create.ForceSendFields = append(create.ForceSendFields, "NumWorkers")
	}

	return create
}

func makeEditCluster(id string, config *compute.ClusterSpec) compute.EditCluster {
	edit := compute.EditCluster{
		ClusterId:                  id,
		ApplyPolicyDefaultValues:   config.ApplyPolicyDefaultValues,
		Autoscale:                  config.Autoscale,
		AutoterminationMinutes:     config.AutoterminationMinutes,
		AwsAttributes:              config.AwsAttributes,
		AzureAttributes:            config.AzureAttributes,
		ClusterLogConf:             config.ClusterLogConf,
		ClusterName:                config.ClusterName,
		CustomTags:                 config.CustomTags,
		DataSecurityMode:           config.DataSecurityMode,
		DependencyMode:             config.DependencyMode,
		DockerImage:                config.DockerImage,
		DriverInstancePoolId:       config.DriverInstancePoolId,
		DriverNodeTypeId:           config.DriverNodeTypeId,
		DriverNodeTypeFlexibility:  config.DriverNodeTypeFlexibility,
		EnableElasticDisk:          config.EnableElasticDisk,
		EnableLocalDiskEncryption:  config.EnableLocalDiskEncryption,
		GcpAttributes:              config.GcpAttributes,
		InitScripts:                config.InitScripts,
		InstancePoolId:             config.InstancePoolId,
		IsSingleNode:               config.IsSingleNode,
		Kind:                       config.Kind,
		NodeTypeId:                 config.NodeTypeId,
		NumWorkers:                 config.NumWorkers,
		PolicyId:                   config.PolicyId,
		RemoteDiskThroughput:       config.RemoteDiskThroughput,
		RuntimeEngine:              config.RuntimeEngine,
		SingleUserName:             config.SingleUserName,
		SparkConf:                  config.SparkConf,
		SparkEnvVars:               config.SparkEnvVars,
		SparkVersion:               config.SparkVersion,
		SshPublicKeys:              config.SshPublicKeys,
		TotalInitialRemoteDiskSize: config.TotalInitialRemoteDiskSize,
		UseMlRuntime:               config.UseMlRuntime,
		WorkloadType:               config.WorkloadType,
		WorkerNodeTypeFlexibility:  config.WorkerNodeTypeFlexibility,
		ForceSendFields:            utils.FilterFields[compute.EditCluster](config.ForceSendFields),
	}

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	// If NumWorkers is not nil, we don't need to set it to ForceSendFields as it will be sent anyway.
	if config.Autoscale == nil && config.NumWorkers == 0 {
		edit.ForceSendFields = append(edit.ForceSendFields, "NumWorkers")
	}

	return edit
}
