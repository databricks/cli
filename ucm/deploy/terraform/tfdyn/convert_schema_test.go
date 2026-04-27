package tfdyn

import (
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertSchema(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		src     resources.Schema
		managed map[string]struct{}
		want    map[string]any
	}{
		{
			name: "minimal external catalog",
			key:  "raw",
			src:  resources.Schema{CreateSchema: catalog.CreateSchema{Name: "raw", CatalogName: "external_catalog"}},
			want: map[string]any{
				"name":          "raw",
				"catalog_name":  "external_catalog",
				"force_destroy": true,
			},
		},
		{
			name:    "managed catalog emits depends_on",
			key:     "raw",
			src:     resources.Schema{CreateSchema: catalog.CreateSchema{Name: "raw_data", CatalogName: "sales"}},
			managed: map[string]struct{}{"sales": {}},
			want: map[string]any{
				"name":          "raw_data",
				"catalog_name":  "sales",
				"force_destroy": true,
				"depends_on":    []any{"databricks_catalog.sales"},
			},
		},
		{
			name: "with comment and tags",
			key:  "analytics",
			src: resources.Schema{CreateSchema: catalog.CreateSchema{Name: "analytics", CatalogName: "ext", Comment: "analytics schema"}, Tags: map[string]string{"owner": "data"}},
			want: map[string]any{
				"name":          "analytics",
				"catalog_name":  "ext",
				"comment":       "analytics schema",
				"properties":    map[string]any{"owner": "data"},
				"force_destroy": true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)

			out := NewResources()
			for k := range tc.managed {
				out.Catalog[k] = dyn.V("placeholder")
			}

			err = schemaConverter{}.Convert(t.Context(), tc.key, vin, out)
			require.NoError(t, err)

			got, ok := out.Schema[tc.key]
			require.True(t, ok)
			assert.Equal(t, tc.want, got.AsAny())
		})
	}
}
