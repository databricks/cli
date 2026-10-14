package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	instancepools "github.com/databricks/cli/libs/sdktypes/instancepools/v2"
	"github.com/databricks/cli/libs/utils"
	sdkinstancepools "github.com/databricks/sdk-go/instancepools/v2"
)

type ResourceInstancePool struct {
	client *sdkclients.Clients
}

func (*ResourceInstancePool) New(client *sdkclients.Clients) *ResourceInstancePool {
	return &ResourceInstancePool{client: client}
}

func (*ResourceInstancePool) PrepareState(input *resources.InstancePool) *instancepools.CreateInstancePoolRequest {
	return &input.CreateInstancePoolRequest
}

// RemapState copies the config fields shared by GetInstancePoolResponse and CreateInstancePoolRequest;
// output-only fields (state, stats, default_tags, instance_pool_id) are not in the state.
func (*ResourceInstancePool) RemapState(remote *instancepools.GetInstancePoolResponse) *instancepools.CreateInstancePoolRequest {
	return &instancepools.CreateInstancePoolRequest{
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
	}
}

func (r *ResourceInstancePool) DoRead(ctx context.Context, id string) (*instancepools.GetInstancePoolResponse, error) {
	pool, err := r.client.InstancePools.GetInstancePool(ctx, sdkinstancepools.GetInstancePoolRequest{InstancePoolId: new(id)})
	if err != nil {
		return nil, err
	}
	return instancepools.GetInstancePoolResponseFromSDK(pool)
}

func (r *ResourceInstancePool) DoCreate(ctx context.Context, config *instancepools.CreateInstancePoolRequest) (string, *instancepools.GetInstancePoolResponse, error) {
	request, err := instancepools.CreateInstancePoolRequestToSDK(config)
	if err != nil {
		return "", nil, err
	}
	resp, err := r.client.InstancePools.CreateInstancePool(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return utils.Deref(resp.InstancePoolId), nil, nil
}

// DoUpdate sends only the fields the edit API allows changing.
func (r *ResourceInstancePool) DoUpdate(ctx context.Context, id string, config *instancepools.CreateInstancePoolRequest, _ *PlanEntry) (*instancepools.GetInstancePoolResponse, error) {
	request, err := instancepools.EditInstancePoolRequestToSDK(&instancepools.EditInstancePoolRequest{
		InstancePoolId:                     new(id),
		InstancePoolName:                   config.InstancePoolName,
		NodeTypeId:                         config.NodeTypeId,
		MinIdleInstances:                   config.MinIdleInstances,
		MaxCapacity:                        config.MaxCapacity,
		IdleInstanceAutoterminationMinutes: config.IdleInstanceAutoterminationMinutes,
		CustomTags:                         config.CustomTags,
		RemoteDiskThroughput:               config.RemoteDiskThroughput,
		TotalInitialRemoteDiskSize:         config.TotalInitialRemoteDiskSize,
	})
	if err != nil {
		return nil, err
	}
	_, err = r.client.InstancePools.EditInstancePool(ctx, *request)
	return nil, err
}

func (r *ResourceInstancePool) DoDelete(ctx context.Context, id string, _ *instancepools.CreateInstancePoolRequest) error {
	_, err := r.client.InstancePools.DeleteInstancePool(ctx, sdkinstancepools.DeleteInstancePoolRequest{InstancePoolId: new(id)})
	return err
}
