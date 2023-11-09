package cfgpickers

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/qa"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstCompatibleWarehouse(t *testing.T) {
	cfg, server := qa.HTTPFixtures{
		{
			Method:   "GET",
			Resource: "/api/2.0/sql/warehouses?",
			Response: sql.ListWarehousesResponse{
				Warehouses: []sql.EndpointInfo{
					{
						Id:            "efg-id",
						Name:          "First PRO Warehouse",
						WarehouseType: sql.EndpointInfoWarehouseTypePro,
					},
					{
						Id:            "ghe-id",
						Name:          "Second UNKNOWN Warehouse",
						WarehouseType: sql.EndpointInfoWarehouseTypeTypeUnspecified,
					},
				},
			},
		},
	}.Config(t)
	defer server.Close()
	w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))

	ctx := context.Background()
	clusterID, err := AskForWarehouse(ctx, w, WithWarehouseTypes(sql.EndpointInfoWarehouseTypePro))
	require.NoError(t, err)
	assert.Equal(t, "efg-id", clusterID)
}

func TestNoCompatibleWarehouses(t *testing.T) {
	cfg, server := qa.HTTPFixtures{
		{
			Method:   "GET",
			Resource: "/api/2.0/sql/warehouses?",
			Response: sql.ListWarehousesResponse{
				Warehouses: []sql.EndpointInfo{
					{
						Id:            "efg-id",
						Name:          "...",
						WarehouseType: sql.EndpointInfoWarehouseTypeClassic,
					},
				},
			},
		},
	}.Config(t)
	defer server.Close()
	w := databricks.Must(databricks.NewWorkspaceClient((*databricks.Config)(cfg)))

	ctx := context.Background()
	_, err := AskForWarehouse(ctx, w, WithWarehouseTypes(sql.EndpointInfoWarehouseTypePro))
	assert.Equal(t, ErrNoCompatibleWarehouses, err)
}
