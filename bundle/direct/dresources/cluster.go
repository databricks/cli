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
		ForceSendFields:            utils.FilterFields[compute.ClusterSpec](input.ForceSendFields),
	}
	if input.Spec != nil {
		spec.ApplyPolicyDefaultValues = input.Spec.ApplyPolicyDefaultValues
	}
	return spec
}

func (r *ResourceCluster) DoRead(ctx context.Context, id string) (*compute.ClusterDetails, error) {
	return r.client.Clusters.GetByClusterId(ctx, id)
}

func (r *ResourceCluster) DoCreate(ctx context.Context, config *compute.ClusterSpec) (string, *compute.ClusterDetails, error) {
	wait, err := r.client.Clusters.Create(ctx, makeCreateCluster(config))
	if err != nil {
		return "", nil, err
	}
	return wait.ClusterId, nil, nil
}

func (r *ResourceCluster) DoUpdate(ctx context.Context, id string, config *compute.ClusterSpec, _ Changes) (*compute.ClusterDetails, error) {
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
			return nil, retries.Continues(fmt.Sprintf("cluster %s cannot be modified in its current state: %s", id, apiErr.Message))
		}
		return nil, retries.Halt(err)
	})
	return nil, err
}

func (r *ResourceCluster) DoResize(ctx context.Context, id string, config *compute.ClusterSpec) error {
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

func (r *ResourceCluster) OverrideChangeDesc(ctx context.Context, p *structpath.PathNode, change *ChangeDesc, remoteState *compute.ClusterDetails) error {
	path := p.Prefix(1).String()
	switch path {
	case "data_security_mode":
		// We do change skip here in the same way TF provider does suppress diff if the alias is used.
		// https://github.com/databricks/terraform-provider-databricks/blob/main/clusters/resource_cluster.go#L109-L117
		if change.New == compute.DataSecurityModeDataSecurityModeStandard && change.Remote == compute.DataSecurityModeUserIsolation && change.New == change.Old {
			change.Action = deployplan.ActionTypeSkip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeDedicated && change.Remote == compute.DataSecurityModeSingleUser && change.New == change.Old {
			change.Action = deployplan.ActionTypeSkip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeAuto && (change.Remote == compute.DataSecurityModeSingleUser || change.Remote == compute.DataSecurityModeUserIsolation) && change.New == change.Old {
			change.Action = deployplan.ActionTypeSkip
			change.Reason = deployplan.ReasonAlias
		}

	case "num_workers", "autoscale":
		if remoteState.State == compute.StateRunning {
			change.Action = deployplan.ActionTypeResize
		}
	}
	return nil
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
		ForceSendFields:            utils.FilterFields[compute.EditCluster](config.ForceSendFields),
	}

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	// If NumWorkers is not nil, we don't need to set it to ForceSendFields as it will be sent anyway.
	if config.Autoscale == nil && config.NumWorkers == 0 {
		edit.ForceSendFields = append(edit.ForceSendFields, "NumWorkers")
	}

	return edit
}
