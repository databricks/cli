package tfdyn

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPostgresEndpoint(t *testing.T) {
	src := resources.PostgresEndpoint{
		EndpointId: "my-endpoint",
		Parent:     "projects/my-project/branches/main",
		EndpointSpec: postgres.EndpointSpec{
			EndpointType:           postgres.EndpointTypeEndpointTypeReadWrite,
			AutoscalingLimitMinCu:  1,
			AutoscalingLimitMaxCu:  4,
			SuspendTimeoutDuration: duration.New(300 * time.Second),
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresEndpointConverter{}.Convert(ctx, "my_postgres_endpoint", vin, out)
	require.NoError(t, err)

	postgresEndpoint := out.PostgresEndpoint["my_postgres_endpoint"]
	assert.Equal(t, map[string]any{
		"endpoint_id": "my-endpoint",
		"parent":      "projects/my-project/branches/main",
		"spec": map[string]any{
			"endpoint_type":            "ENDPOINT_TYPE_READ_WRITE",
			"autoscaling_limit_min_cu": int64(1),
			"autoscaling_limit_max_cu": int64(4),
			"suspend_timeout_duration": "300s",
		},
	}, postgresEndpoint)
}

func TestConvertPostgresEndpointReadOnly(t *testing.T) {
	src := resources.PostgresEndpoint{
		EndpointId: "readonly-endpoint",
		Parent:     "projects/my-project/branches/main",
		EndpointSpec: postgres.EndpointSpec{
			EndpointType: postgres.EndpointTypeEndpointTypeReadOnly,
			NoSuspension: true,
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresEndpointConverter{}.Convert(ctx, "readonly_postgres_endpoint", vin, out)
	require.NoError(t, err)

	postgresEndpoint := out.PostgresEndpoint["readonly_postgres_endpoint"]
	assert.Equal(t, map[string]any{
		"endpoint_id": "readonly-endpoint",
		"parent":      "projects/my-project/branches/main",
		"spec": map[string]any{
			"endpoint_type": "ENDPOINT_TYPE_READ_ONLY",
			"no_suspension": true,
		},
	}, postgresEndpoint)
}

func TestConvertPostgresEndpointWithSettings(t *testing.T) {
	src := resources.PostgresEndpoint{
		EndpointId: "settings-endpoint",
		Parent:     "projects/my-project/branches/main",
		EndpointSpec: postgres.EndpointSpec{
			EndpointType: postgres.EndpointTypeEndpointTypeReadWrite,
			Settings: &postgres.EndpointSettings{
				PgSettings: map[string]string{
					"max_connections":          "100",
					"shared_buffers":           "256MB",
					"effective_cache_size":     "1GB",
					"maintenance_work_mem":     "64MB",
					"checkpoint_timeout":       "15min",
					"work_mem":                 "4MB",
					"random_page_cost":         "1.1",
					"effective_io_concurrency": "200",
				},
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresEndpointConverter{}.Convert(ctx, "settings_postgres_endpoint", vin, out)
	require.NoError(t, err)

	postgresEndpoint := out.PostgresEndpoint["settings_postgres_endpoint"]
	assert.Equal(t, map[string]any{
		"endpoint_id": "settings-endpoint",
		"parent":      "projects/my-project/branches/main",
		"spec": map[string]any{
			"endpoint_type": "ENDPOINT_TYPE_READ_WRITE",
			"settings": map[string]any{
				"pg_settings": map[string]any{
					"max_connections":          "100",
					"shared_buffers":           "256MB",
					"effective_cache_size":     "1GB",
					"maintenance_work_mem":     "64MB",
					"checkpoint_timeout":       "15min",
					"work_mem":                 "4MB",
					"random_page_cost":         "1.1",
					"effective_io_concurrency": "200",
				},
			},
		},
	}, postgresEndpoint)
}
