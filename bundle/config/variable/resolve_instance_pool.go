package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveInstancePool struct {
	name string
}

func (l resolveInstancePool) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.InstancePools.GetByInstancePoolName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.InstancePoolId, nil
}

func (l resolveInstancePool) String() string {
	return "instance-pool: " + l.name
}
