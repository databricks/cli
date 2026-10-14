package variable

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/instancepools/v2"
)

type resolveInstancePool struct {
	name string
}

func (l resolveInstancePool) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	resp, err := w.InstancePools.ListInstancePools(ctx, instancepools.ListInstancePoolsRequest{})
	if err != nil {
		return "", err
	}
	entity, err := findByName(sliceSeq(resp.InstancePools), l.name, "InstancePoolAndStats", func(p *instancepools.InstancePoolAndStats) string {
		return utils.Deref(p.InstancePoolName)
	})
	if err != nil {
		return "", err
	}
	return utils.Deref(entity.InstancePoolId), nil
}

func (l resolveInstancePool) String() string {
	return "instance-pool: " + l.name
}
