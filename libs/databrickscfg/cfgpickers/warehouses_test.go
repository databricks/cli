package cfgpickers

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/httpfixtures"
	"github.com/databricks/cli/libs/sdkclients"
	sql "github.com/databricks/sdk-go/warehouses/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstCompatibleWarehouse(t *testing.T) {
	cfg, _ := httpfixtures.Fixtures{
		{
			Method:   "GET",
			Resource: "/api/2.0/sql/warehouses",
			Response: `{"warehouses":[{"id":"efg-id","name":"First PRO Warehouse","warehouse_type":"PRO"},{"id":"ghe-id","name":"Second UNKNOWN Warehouse","warehouse_type":"TYPE_UNSPECIFIED"}]}`,
		},
	}.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	clusterID, err := AskForWarehouse(ctx, w, WithWarehouseTypes(sql.WarehouseType_Pro))
	require.NoError(t, err)
	assert.Equal(t, "efg-id", clusterID)
}

func TestNoCompatibleWarehouses(t *testing.T) {
	cfg, _ := httpfixtures.Fixtures{
		{
			Method:   "GET",
			Resource: "/api/2.0/sql/warehouses",
			Response: `{"warehouses":[{"id":"efg-id","name":"...","warehouse_type":"CLASSIC"}]}`,
		},
	}.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)

	ctx := cmdio.MockDiscard(t.Context())
	_, err = AskForWarehouse(ctx, w, WithWarehouseTypes(sql.WarehouseType_Pro))
	assert.Equal(t, ErrNoCompatibleWarehouses, err)
}
