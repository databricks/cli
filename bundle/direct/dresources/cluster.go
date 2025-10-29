package dresources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ResourceCluster struct {
	client *databricks.WorkspaceClient
}

func (r *ResourceCluster) New(client *databricks.WorkspaceClient) any {
	return &ResourceCluster{
		client: client,
	}
}

func (r *ResourceCluster) PrepareState(input *resources.Cluster) *compute.ClusterSpec {
	return &input.ClusterSpec
}

func (r *ResourceCluster) RemapState(input *compute.ClusterDetails) *compute.ClusterSpec {
	spec := &compute.ClusterSpec{
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
		ForceSendFields:            filterFields[compute.ClusterSpec](input.ForceSendFields),
	}
	if input.Spec != nil {
		spec.ApplyPolicyDefaultValues = input.Spec.ApplyPolicyDefaultValues
	}
	return spec
}

func (r *ResourceCluster) DoRefresh(ctx context.Context, id string) (*compute.ClusterDetails, error) {
	return r.client.Clusters.GetByClusterId(ctx, id)
}

func (r *ResourceCluster) DoCreate(ctx context.Context, config *compute.ClusterSpec) (string, error) {
	wait, err := r.client.Clusters.Create(ctx, makeCreateCluster(config))
	if err != nil {
		return "", err
	}
	return wait.ClusterId, nil
}

func (r *ResourceCluster) DoUpdate(ctx context.Context, id string, config *compute.ClusterSpec) error {
	// Same retry as in TF provider logic
	// https://github.com/databricks/terraform-provider-databricks/blob/3eecd0f90cf99d7777e79a3d03c41f9b2aafb004/clusters/resource_cluster.go#L624
	timeout := 15 * time.Minute
	_, err := retries.Poll(ctx, timeout, func() (*compute.WaitGetClusterRunning[struct{}], *retries.Err) {
		wait, err := r.client.Clusters.Edit(ctx, makeEditCluster(id, config))
		if err == nil {
			return wait, nil
		}

		var apiErr *apierr.APIError
		// Only Running and Terminated clusters can be modified. In particular, autoscaling clusters cannot be modified
		// while the resizing is ongoing. We retry in this case. Scaling can take several minutes.
		if errors.As(err, &apiErr) && apiErr.ErrorCode == "INVALID_STATE" {
			return nil, retries.Continues(fmt.Sprintf("cluster %s cannot be modified in its current state", id))
		}
		return nil, retries.Halt(err)
	})
	return err
}

func (r *ResourceCluster) DoResize(ctx context.Context, id string, config *compute.ClusterSpec) error {
	_, err := r.client.Clusters.Resize(ctx, compute.ResizeCluster{
		ClusterId:       id,
		NumWorkers:      config.NumWorkers,
		Autoscale:       config.Autoscale,
		ForceSendFields: filterFields[compute.ResizeCluster](config.ForceSendFields),
	})
	return err
}

func (r *ResourceCluster) DoDelete(ctx context.Context, id string) error {
	return r.client.Clusters.PermanentDeleteByClusterId(ctx, id)
}

func (r *ResourceCluster) ClassifyChange(change structdiff.Change, remoteState *compute.ClusterDetails, _ bool) (deployplan.ActionType, error) {
	changedPath := change.Path.String()
	if changedPath == "data_security_mode" {
		// We do change skip here in the same way TF provider does suppress diff if the alias is used.
		// https://github.com/databricks/terraform-provider-databricks/blob/main/clusters/resource_cluster.go#L109-L117
		if change.Old == compute.DataSecurityModeDataSecurityModeStandard && change.New == compute.DataSecurityModeUserIsolation {
			return deployplan.ActionTypeSkip, nil
		}
		if change.Old == compute.DataSecurityModeDataSecurityModeDedicated && change.New == compute.DataSecurityModeSingleUser {
			return deployplan.ActionTypeSkip, nil
		}
		if change.Old == compute.DataSecurityModeDataSecurityModeAuto && (change.New == compute.DataSecurityModeSingleUser || change.New == compute.DataSecurityModeUserIsolation) {
			return deployplan.ActionTypeSkip, nil
		}
	}

	// Always update if the cluster is not running.
	if remoteState.State != compute.StateRunning {
		return deployplan.ActionTypeUpdate, nil
	}

	if changedPath == "num_workers" || strings.HasPrefix(changedPath, "autoscale") {
		return deployplan.ActionTypeResize, nil
	}

	return deployplan.ActionTypeUpdate, nil
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
		ForceSendFields:            filterFields[compute.CreateCluster](config.ForceSendFields),
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
		ForceSendFields:            filterFields[compute.EditCluster](config.ForceSendFields),
	}

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	// If NumWorkers is not nil, we don't need to set it to ForceSendFields as it will be sent anyway.
	if config.Autoscale == nil && config.NumWorkers == 0 {
		edit.ForceSendFields = append(edit.ForceSendFields, "NumWorkers")
	}

	return edit
}
