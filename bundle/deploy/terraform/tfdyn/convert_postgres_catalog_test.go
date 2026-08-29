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

func TestConvertPostgresCatalog(t *testing.T) {
	src := resources.PostgresCatalog{
		PostgresCatalogConfig: resources.PostgresCatalogConfig{
			CatalogId: "shop_lakebase",
			CatalogCatalogSpec: postgres.CatalogCatalogSpec{
				Branch:                  "projects/my-shop/branches/production",
				PostgresDatabase:        "appdb",
				CreateDatabaseIfMissing: true,
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresCatalogConverter{}.Convert(ctx, "my_postgres_catalog", vin, out)
	require.NoError(t, err)

	postgresCatalog := out.PostgresCatalog["my_postgres_catalog"]
	assert.Equal(t, map[string]any{
		"catalog_id": "shop_lakebase",
		"spec": map[string]any{
			"branch":                     "projects/my-shop/branches/production",
			"postgres_database":          "appdb",
			"create_database_if_missing": true,
		},
	}, postgresCatalog)
}

func TestConvertPostgresCatalogMinimal(t *testing.T) {
	src := resources.PostgresCatalog{
		PostgresCatalogConfig: resources.PostgresCatalogConfig{
			CatalogId: "minimal_catalog",
			CatalogCatalogSpec: postgres.CatalogCatalogSpec{
				Branch:           "projects/my-shop/branches/production",
				PostgresDatabase: "appdb",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := t.Context()
	out := schema.NewResources()
	err = postgresCatalogConverter{}.Convert(ctx, "minimal_postgres_catalog", vin, out)
	require.NoError(t, err)

	postgresCatalog := out.PostgresCatalog["minimal_postgres_catalog"]
	assert.Equal(t, map[string]any{
		"catalog_id": "minimal_catalog",
		"spec": map[string]any{
			"branch":            "projects/my-shop/branches/production",
			"postgres_database": "appdb",
		},
	}, postgresCatalog)
}
