package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
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

var (
	toCreateInstancePool = newCopier[compute.GetInstancePool, compute.CreateInstancePool]()
	toEditInstancePool   = newCopier[compute.CreateInstancePool, compute.EditInstancePool]()
)

// RemapState copies the config fields shared by GetInstancePool and CreateInstancePool;
// output-only fields (state, stats, default_tags, instance_pool_id) are not in the state.
func (*ResourceInstancePool) RemapState(remote *compute.GetInstancePool) *compute.CreateInstancePool {
	return toCreateInstancePool.Copy(remote)
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
	edit := toEditInstancePool.Copy(config)
	edit.InstancePoolId = id
	return nil, r.client.InstancePools.Edit(ctx, *edit)
}

func (r *ResourceInstancePool) DoDelete(ctx context.Context, id string, _ *compute.CreateInstancePool) error {
	return r.client.InstancePools.DeleteByInstancePoolId(ctx, id)
}
