package variable

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

type lookupWarehouse struct {
	name string
}

func (l lookupWarehouse) Resolve(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	entity, err := w.Warehouses.GetByName(ctx, l.name)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(entity.Id), nil
}

func (l lookupWarehouse) String() string {
	return fmt.Sprintf("warehouse: %s", l.name)
}
