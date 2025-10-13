package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go"
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
	_, err := r.client.Clusters.Edit(ctx, makeEditCluster(id, config))
	if err != nil {
		return err
	}
	return nil
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

func (r *ResourceCluster) ClassifyChange(change structdiff.Change, remoteState *compute.ClusterDetails) (deployplan.ActionType, error) {
	// Always update if the cluster is not running.
	if remoteState.State != compute.StateRunning {
		return deployplan.ActionTypeUpdate, nil
	}

	changedPath := change.Path.String()
	if changedPath == "num_workers" || strings.HasPrefix(changedPath, "autoscale") {
		return deployplan.ActionTypeResize, nil
	}

	return deployplan.ActionTypeUpdate, nil
}

func makeCreateCluster(config *compute.ClusterSpec) compute.CreateCluster {
	return compute.CreateCluster{
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
}

func makeEditCluster(id string, config *compute.ClusterSpec) compute.EditCluster {
	return compute.EditCluster{
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
}
