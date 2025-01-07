package variable

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

type resolveWarehouse struct {
	name string
}

func (l resolveWarehouse) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Warehouses.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return entity.Id, nil
}

func (l resolveWarehouse) String() string {
	return "warehouse: " + l.name
}
