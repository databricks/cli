package dresources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

// ClusterStateLifecycle holds lifecycle settings persisted in state.
type ClusterStateLifecycle struct {
	Started *bool `json:"started,omitempty"`
}

// ClusterState is the state type for Cluster resources. It extends compute.ClusterSpec with
// lifecycle settings and a transient ClusterId used by WaitAfterCreate and WaitAfterUpdate.
type ClusterState struct {
	compute.ClusterSpec

	// ClusterId is set by DoCreate/DoUpdate for use in WaitAfterCreate/WaitAfterUpdate; it is not persisted in state.
	ClusterId string `json:"-"`

	Lifecycle *ClusterStateLifecycle `json:"lifecycle,omitempty"`
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
type ClusterRemote struct {
	compute.ClusterDetails
	Lifecycle *ClusterStateLifecycle `json:"lifecycle,omitempty"`
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
		s.Lifecycle = &ClusterStateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote ClusterRemote to ClusterState for diff comparison.
// Started is derived from cluster state so the planner can detect start/stop changes.
func (r *ResourceCluster) RemapState(input *ClusterRemote) *ClusterState {
	started := input.State == compute.StateRunning
	spec := &ClusterState{
		ClusterSpec: compute.ClusterSpec{
			ApplyPolicyDefaultValues:   false,
			Autoscale:                  input.Autoscale,
			AutoterminationMinutes:     input.AutoterminationMinutes,
			AwsAttributes:              input.AwsAttributes,
			AzureAttributes:            input.AzureAttributes,
			ClusterLogConf:             input.ClusterLogConf,
			ClusterName:                input.ClusterName,
			CustomTags:                 input.CustomTags,
			DataSecurityMode:           input.DataSecurityMode,
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
		Lifecycle: &ClusterStateLifecycle{Started: &started},
	}
	if input.Spec != nil {
		spec.ApplyPolicyDefaultValues = input.Spec.ApplyPolicyDefaultValues
	}
	return spec
}

func (r *ResourceCluster) DoRead(ctx context.Context, id string) (*ClusterRemote, error) {
	details, err := r.client.Clusters.GetByClusterId(ctx, id)
	if err != nil {
		return nil, err
	}
	started := details.State == compute.StateRunning
	return &ClusterRemote{
		ClusterDetails: *details,
		Lifecycle:      &ClusterStateLifecycle{Started: &started},
	}, nil
}

func (r *ResourceCluster) DoCreate(ctx context.Context, config *ClusterState) (string, *ClusterRemote, error) {
	wait, err := r.client.Clusters.Create(ctx, makeCreateCluster(&config.ClusterSpec))
	if err != nil {
		return "", nil, err
	}
	// Store cluster ID in memory for WaitAfterCreate to use.
	// This does not affect state serialization because ClusterId uses json:"-".
	config.ClusterId = wait.ClusterId
	return wait.ClusterId, nil, nil
}

// lifecycleOnlyFields are ClusterState fields managed via the Start/Delete APIs, not the Cluster Edit API.
var lifecycleOnlyFields = map[string]bool{
	"lifecycle":         true,
	"lifecycle.started": true,
}

// hasClusterChanges reports whether the plan entry contains any Update changes
// to fields that belong to the Cluster Edit API (i.e., not lifecycle-only fields).
func hasClusterChanges(entry *PlanEntry) bool {
	for path, change := range entry.Changes {
		if change.Action == deployplan.Update && !lifecycleOnlyFields[truncateAtIndex(path)] {
			return true
		}
	}
	return false
}

func (r *ResourceCluster) DoUpdate(ctx context.Context, id string, config *ClusterState, entry *PlanEntry) (*ClusterRemote, error) {
	// Store cluster ID for WaitAfterUpdate to use.
	config.ClusterId = id

	if hasClusterChanges(entry) {
		// Same retry as in TF provider logic
		// https://github.com/databricks/terraform-provider-databricks/blob/3eecd0f90cf99d7777e79a3d03c41f9b2aafb004/clusters/resource_cluster.go#L624
		timeout := 15 * time.Minute
		_, err := retries.Poll(ctx, timeout, func() (*compute.WaitGetClusterRunning[struct{}], *retries.Err) {
			wait, err := r.client.Clusters.Edit(ctx, makeEditCluster(id, &config.ClusterSpec))
			if err == nil {
				return wait, nil
			}

			var apiErr *apierr.APIError
			// Only Running and Terminated clusters can be modified. In particular, autoscaling clusters cannot be modified
			// while the resizing is ongoing. We retry in this case. Scaling can take several minutes.
			if errors.As(err, &apiErr) && apiErr.ErrorCode == "INVALID_STATE" {
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
		// lifecycle.started=true: fire Start; WaitAfterUpdate polls for RUNNING.
		_, err := r.client.Clusters.Start(ctx, compute.StartCluster{ClusterId: id})
		return nil, err
	} else if !desiredStarted && alreadyRunning {
		// lifecycle.started=false: fire Delete; WaitAfterUpdate polls for TERMINATED.
		// Delete does not remove the cluster, it just sets the state to TERMINATED.
		_, err := r.client.Clusters.Delete(ctx, compute.DeleteCluster{ClusterId: id})
		return nil, err
	}

	return nil, nil
}

// WaitAfterUpdate waits for the cluster to reach the desired lifecycle state after DoUpdate.
func (r *ResourceCluster) WaitAfterUpdate(ctx context.Context, config *ClusterState) (*ClusterRemote, error) {
	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	if *config.Lifecycle.Started {
		_, err := r.client.Clusters.WaitGetClusterRunning(ctx, config.ClusterId, 15*time.Minute, nil)
		return nil, err
	}

	_, err := r.client.Clusters.WaitGetClusterTerminated(ctx, config.ClusterId, 15*time.Minute, nil)
	return nil, err
}

// WaitAfterCreate waits for the cluster to reach RUNNING state (clusters always start on creation).
// When lifecycle.started=false, it then terminates the cluster.
func (r *ResourceCluster) WaitAfterCreate(ctx context.Context, config *ClusterState) (*ClusterRemote, error) {
	// Always wait for RUNNING first: clusters start in PENDING state and must be polled.
	_, err := r.client.Clusters.WaitGetClusterRunning(ctx, config.ClusterId, 15*time.Minute, nil)
	if err != nil {
		return nil, err
	}

	if config.Lifecycle != nil && config.Lifecycle.Started != nil && !*config.Lifecycle.Started {
		// started=false: terminate the cluster after it reaches RUNNING.
		deleteWaiter, err := r.client.Clusters.Delete(ctx, compute.DeleteCluster{ClusterId: config.ClusterId})
		if err != nil {
			return nil, err
		}
		_, err = deleteWaiter.Get()
		return nil, err
	}

	return nil, nil
}

func (r *ResourceCluster) DoResize(ctx context.Context, id string, config *ClusterState) error {
	_, err := r.client.Clusters.Resize(ctx, compute.ResizeCluster{
		ClusterId:       id,
		NumWorkers:      config.NumWorkers,
		Autoscale:       config.Autoscale,
		ForceSendFields: utils.FilterFields[compute.ResizeCluster](config.ForceSendFields),
	})
	return err
}

func (r *ResourceCluster) DoDelete(ctx context.Context, id string) error {
	return r.client.Clusters.PermanentDeleteByClusterId(ctx, id)
}

func (r *ResourceCluster) OverrideChangeDesc(ctx context.Context, p *structpath.PathNode, change *ChangeDesc, remoteState *ClusterRemote) error {
	// We're only interested in downgrading some updates to skips. Changes that already skipped or cause recreation should remain unchanged.
	if change.Action != deployplan.Update {
		return nil
	}

	path := p.Prefix(1).String()
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
