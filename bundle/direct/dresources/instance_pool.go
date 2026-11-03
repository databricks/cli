package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ResourceInstancePool struct {
	client *databricks.WorkspaceClient
}

func (*ResourceInstancePool) New(client *databricks.WorkspaceClient) *ResourceInstancePool {
	return &ResourceInstancePool{client: client}
}

func (*ResourceInstancePool) PrepareState(input *resources.InstancePool) *compute.CreateInstancePool {
	return &input.CreateInstancePool
}

// RemapState copies the config fields shared by GetInstancePool and CreateInstancePool;
// output-only fields (state, stats, default_tags, instance_pool_id) are not in the state.
func (*ResourceInstancePool) RemapState(remote *compute.GetInstancePool) *compute.CreateInstancePool {
	return &compute.CreateInstancePool{
		AwsAttributes:                      remote.AwsAttributes,
		AzureAttributes:                    remote.AzureAttributes,
		CustomTags:                         remote.CustomTags,
		DiskSpec:                           remote.DiskSpec,
		EnableElasticDisk:                  remote.EnableElasticDisk,
		GcpAttributes:                      remote.GcpAttributes,
		IdleInstanceAutoterminationMinutes: remote.IdleInstanceAutoterminationMinutes,
		InstancePoolName:                   remote.InstancePoolName,
		MaxCapacity:                        remote.MaxCapacity,
		MinIdleInstances:                   remote.MinIdleInstances,
		NodeTypeFlexibility:                remote.NodeTypeFlexibility,
		NodeTypeId:                         remote.NodeTypeId,
		PreloadedDockerImages:              remote.PreloadedDockerImages,
		PreloadedSparkVersions:             remote.PreloadedSparkVersions,
		RemoteDiskThroughput:               remote.RemoteDiskThroughput,
		TotalInitialRemoteDiskSize:         remote.TotalInitialRemoteDiskSize,
		ForceSendFields:                    utils.FilterFields[compute.CreateInstancePool](remote.ForceSendFields),
	}
}

func (r *ResourceInstancePool) DoRead(ctx context.Context, id string) (*compute.GetInstancePool, error) {
	return r.client.InstancePools.GetByInstancePoolId(ctx, id)
}

func (r *ResourceInstancePool) DoCreate(ctx context.Context, config *compute.CreateInstancePool) (string, *compute.GetInstancePool, error) {
	resp, err := r.client.InstancePools.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return resp.InstancePoolId, nil, nil
}

func (r *ResourceInstancePool) DoUpdate(ctx context.Context, id string, config *compute.CreateInstancePool, _ *PlanEntry) (*compute.GetInstancePool, error) {
	return nil, r.client.InstancePools.Edit(ctx, compute.EditInstancePool{
		InstancePoolId:                     id,
		InstancePoolName:                   config.InstancePoolName,
		NodeTypeId:                         config.NodeTypeId,
		MinIdleInstances:                   config.MinIdleInstances,
		MaxCapacity:                        config.MaxCapacity,
		IdleInstanceAutoterminationMinutes: config.IdleInstanceAutoterminationMinutes,
		CustomTags:                         config.CustomTags,
		RemoteDiskThroughput:               config.RemoteDiskThroughput,
		TotalInitialRemoteDiskSize:         config.TotalInitialRemoteDiskSize,
		ForceSendFields:                    utils.FilterFields[compute.EditInstancePool](config.ForceSendFields),
	})
}

func (r *ResourceInstancePool) DoDelete(ctx context.Context, id string, _ *compute.CreateInstancePool) error {
	return r.client.InstancePools.DeleteByInstancePoolId(ctx, id)
}
