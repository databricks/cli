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

func TestConvertPostgresProject(t *testing.T) {
	src := resources.PostgresProject{
		ProjectId: "my-project",
		ProjectSpec: postgres.ProjectSpec{
			DisplayName:              "My Postgres Project",
			PgVersion:                17,
			HistoryRetentionDuration: duration.New(86400 * time.Second),
			DefaultEndpointSettings: &postgres.ProjectDefaultEndpointSettings{
				AutoscalingLimitMinCu:  0.5,
				AutoscalingLimitMaxCu:  4,
				SuspendTimeoutDuration: duration.New(300 * time.Second),
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresProjectConverter{}.Convert(ctx, "my_postgres_project", vin, out)
	require.NoError(t, err)

	postgresProject := out.PostgresProject["my_postgres_project"]
	assert.Equal(t, map[string]any{
		"project_id": "my-project",
		"spec": map[string]any{
			"display_name":               "My Postgres Project",
			"pg_version":                 int64(17),
			"history_retention_duration": "86400s",
			"default_endpoint_settings": map[string]any{
				"autoscaling_limit_min_cu": 0.5,
				"autoscaling_limit_max_cu": float64(4),
				"suspend_timeout_duration": "300s",
			},
		},
	}, postgresProject)
}

func TestConvertPostgresProjectMinimal(t *testing.T) {
	src := resources.PostgresProject{
		ProjectId: "minimal-project",
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresProjectConverter{}.Convert(ctx, "minimal_postgres_project", vin, out)
	require.NoError(t, err)

	postgresProject := out.PostgresProject["minimal_postgres_project"]
	assert.Equal(t, map[string]any{
		"project_id": "minimal-project",
	}, postgresProject)
}
