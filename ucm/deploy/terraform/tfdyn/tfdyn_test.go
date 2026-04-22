package tfdyn

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert_HappyPath(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
resources:
  catalogs:
    sales:
      name: sales_prod
      comment: sales data
  schemas:
    raw:
      name: raw
      catalog: sales
  grants:
    sales_admins:
      securable:
        type: catalog
        name: sales
      principal: sales-admins
      privileges: [USE_CATALOG]
`))
	require.False(t, diags.HasError(), "load diagnostics: %v", diags)

	u := &ucm.Ucm{Config: *cfg}

	got, err := Convert(t.Context(), u)
	require.NoError(t, err)

	want := map[string]any{
		"terraform": map[string]any{
			"required_providers": map[string]any{
				"databricks": map[string]any{
					"source":  "databricks/databricks",
					"version": "1.112.0",
				},
			},
		},
		"provider": map[string]any{
			"databricks": map[string]any{},
		},
		"resource": map[string]any{
			"databricks_catalog": map[string]any{
				"sales": map[string]any{
					"name":          "sales_prod",
					"comment":       "sales data",
					"force_destroy": true,
				},
			},
			"databricks_schema": map[string]any{
				"raw": map[string]any{
					"name":          "raw",
					"catalog_name":  "sales",
					"force_destroy": true,
					"depends_on":    []any{"databricks_catalog.sales"},
				},
			},
			"databricks_grants": map[string]any{
				"sales_admins": map[string]any{
					"catalog": "${databricks_catalog.sales.name}",
					"grant": []any{
						map[string]any{
							"principal":  "sales-admins",
							"privileges": []any{"USE_CATALOG"},
						},
					},
					"depends_on": []any{"databricks_catalog.sales"},
				},
			},
		},
	}
	assert.Equal(t, want, got.AsAny())
}

func TestConvert_EmptyResources(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: empty
`))
	require.False(t, diags.HasError(), "load diagnostics: %v", diags)

	u := &ucm.Ucm{Config: *cfg}

	got, err := Convert(t.Context(), u)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"terraform": map[string]any{
			"required_providers": map[string]any{
				"databricks": map[string]any{
					"source":  "databricks/databricks",
					"version": "1.112.0",
				},
			},
		},
		"provider": map[string]any{
			"databricks": map[string]any{},
		},
		"resource": map[string]any{},
	}, got.AsAny())
}

func TestConvert_PreservesLocations(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
resources:
  catalogs:
    sales:
      name: sales_prod
      comment: sales data
`))
	require.False(t, diags.HasError(), "load diagnostics: %v", diags)

	u := &ucm.Ucm{Config: *cfg}

	got, err := Convert(t.Context(), u)
	require.NoError(t, err)

	catalog := got.Get("resource").Get("databricks_catalog").Get("sales")
	require.True(t, catalog.IsValid())
	// The catalog block inherits its source location from the input node
	// so diagnostics can point back to the offending span in ucm.yml.
	loc := catalog.Location()
	assert.Equal(t, "/test/ucm.yml", loc.File)
	assert.NotZero(t, loc.Line)

	nameField := catalog.Get("name")
	require.True(t, nameField.IsValid())
	assert.Equal(t, "/test/ucm.yml", nameField.Location().File)
}
