package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSqlWarehouse(t *testing.T) {
	src := resources.SqlWarehouse{
		CreateWarehouseRequest: sql.CreateWarehouseRequest{
			AutoStopMins:            120,
			ClusterSize:             "X-Large",
			EnableServerlessCompute: true,
			MaxNumClusters:          1,
			MinNumClusters:          1,
			Name:                    "test_sql_warehouse",
			Tags: &sql.EndpointTags{
				CustomTags: []sql.EndpointTagPair{
					{
						Key:   "key",
						Value: "value",
					},
				},
			},
			Channel: &sql.Channel{
				Name: "CHANNEL_NAME",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = sqlWarehouseConverter{}.Convert(ctx, "test_sql_warehouse", vin, out)
	require.NoError(t, err)

	sqlWarehouse := out.SqlEndpoint["test_sql_warehouse"]
	assert.Equal(t, map[string]any{
		"name": "test_sql_warehouse",
		"tags": map[string]any{
			"custom_tags": []any{
				map[string]any{
					"key":   "key",
					"value": "value",
				},
			},
		},
		"channel": map[string]any{
			"name": "CHANNEL_NAME",
		},
		"auto_stop_mins":            int64(120),
		"cluster_size":              "X-Large",
		"enable_serverless_compute": true,
		"max_num_clusters":          int64(1),
		"min_num_clusters":          int64(1),
	}, sqlWarehouse)
}
