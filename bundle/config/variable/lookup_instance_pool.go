package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupInstancePool struct {
	name string
}

func (l lookupInstancePool) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.InstancePools.GetByInstancePoolName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.InstancePoolId), nil
}

func (l lookupInstancePool) String() string {
	return fmt.Sprintf("instance-pool: %s", l.name)
}
