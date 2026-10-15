package variable

import (
	"context"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/warehouses/v1"
)

type resolveWarehouse struct {
	name string
}

func (l resolveWarehouse) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	entity, err := findByName(w.Warehouses.ListWarehousesIter(ctx, warehouses.ListWarehousesRequest{}), l.name, "EndpointInfo", func(e *warehouses.EndpointInfo) string {
		return utils.Deref(e.Name)
	})
	if err != nil {
		return "", err
	}
	return utils.Deref(entity.Id), nil
}

func (l resolveWarehouse) String() string {
	return "warehouse: " + l.name
}
