package tfdyn

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPostgresSyncedTable(t *testing.T) {
	src := resources.PostgresSyncedTable{
		PostgresSyncedTableConfig: resources.PostgresSyncedTableConfig{
			SyncedTableId: "shop_lakebase.public.orders_synced",
			SyncedTableSyncedTableSpec: postgres.SyncedTableSyncedTableSpec{
				Branch:                         "projects/my-shop/branches/production",
				PostgresDatabase:               "appdb",
				SourceTableFullName:            "main.raw.orders",
				PrimaryKeyColumns:              []string{"order_id"},
				TimeseriesKey:                  "updated_at",
				SchedulingPolicy:               postgres.SyncedTableSyncedTableSpecSyncedTableSchedulingPolicySnapshot,
				CreateDatabaseObjectsIfMissing: true,
				NewPipelineSpec: &postgres.NewPipelineSpec{
					StorageCatalog: "main",
					StorageSchema:  "pipelines",
				},
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresSyncedTableConverter{}.Convert(ctx, "my_postgres_synced_table", vin, out)
	require.NoError(t, err)

	postgresSyncedTable := out.PostgresSyncedTable["my_postgres_synced_table"]
	assert.Equal(t, map[string]any{
		"synced_table_id": "shop_lakebase.public.orders_synced",
		"spec": map[string]any{
			"branch":                             "projects/my-shop/branches/production",
			"postgres_database":                  "appdb",
			"source_table_full_name":             "main.raw.orders",
			"primary_key_columns":                []any{"order_id"},
			"timeseries_key":                     "updated_at",
			"scheduling_policy":                  "SNAPSHOT",
			"create_database_objects_if_missing": true,
			"new_pipeline_spec": map[string]any{
				"storage_catalog": "main",
				"storage_schema":  "pipelines",
			},
		},
	}, postgresSyncedTable)
}

func TestConvertPostgresSyncedTableMinimal(t *testing.T) {
	src := resources.PostgresSyncedTable{
		PostgresSyncedTableConfig: resources.PostgresSyncedTableConfig{
			SyncedTableId: "shop_lakebase.public.orders_synced",
			SyncedTableSyncedTableSpec: postgres.SyncedTableSyncedTableSpec{
				Branch:              "projects/my-shop/branches/production",
				PostgresDatabase:    "appdb",
				SourceTableFullName: "main.raw.orders",
				PrimaryKeyColumns:   []string{"order_id"},
				SchedulingPolicy:    postgres.SyncedTableSyncedTableSpecSyncedTableSchedulingPolicySnapshot,
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresSyncedTableConverter{}.Convert(ctx, "minimal_postgres_synced_table", vin, out)
	require.NoError(t, err)

	postgresSyncedTable := out.PostgresSyncedTable["minimal_postgres_synced_table"]
	assert.Equal(t, map[string]any{
		"synced_table_id": "shop_lakebase.public.orders_synced",
		"spec": map[string]any{
			"branch":                 "projects/my-shop/branches/production",
			"postgres_database":      "appdb",
			"source_table_full_name": "main.raw.orders",
			"primary_key_columns":    []any{"order_id"},
			"scheduling_policy":      "SNAPSHOT",
		},
	}, postgresSyncedTable)
}
