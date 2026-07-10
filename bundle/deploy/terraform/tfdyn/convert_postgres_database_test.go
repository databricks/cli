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

func TestConvertPostgresDatabase(t *testing.T) {
	src := resources.PostgresDatabase{
		PostgresDatabaseConfig: resources.PostgresDatabaseConfig{
			DatabaseId: "my-database",
			Parent:     "projects/my-project/branches/main",
			DatabaseDatabaseSpec: postgres.DatabaseDatabaseSpec{
				PostgresDatabase: "my_postgres_db",
				Role:             "projects/my-project/branches/main/roles/owner",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresDatabaseConverter{}.Convert(ctx, "my_postgres_database", vin, out)
	require.NoError(t, err)

	postgresDatabase := out.PostgresDatabase["my_postgres_database"]
	assert.Equal(t, map[string]any{
		"database_id": "my-database",
		"parent":      "projects/my-project/branches/main",
		"spec": map[string]any{
			"postgres_database": "my_postgres_db",
			"role":              "projects/my-project/branches/main/roles/owner",
		},
	}, postgresDatabase)
}

func TestConvertPostgresDatabaseMinimal(t *testing.T) {
	src := resources.PostgresDatabase{
		PostgresDatabaseConfig: resources.PostgresDatabaseConfig{
			DatabaseId: "minimal-database",
			Parent:     "projects/my-project/branches/main",
			DatabaseDatabaseSpec: postgres.DatabaseDatabaseSpec{
				PostgresDatabase: "minimal_db",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresDatabaseConverter{}.Convert(ctx, "minimal_postgres_database", vin, out)
	require.NoError(t, err)

	postgresDatabase := out.PostgresDatabase["minimal_postgres_database"]
	assert.Equal(t, map[string]any{
		"database_id": "minimal-database",
		"parent":      "projects/my-project/branches/main",
		"spec": map[string]any{
			"postgres_database": "minimal_db",
		},
	}, postgresDatabase)
}
