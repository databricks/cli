package tfdyn

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertCatalog(t *testing.T) {
	tests := []struct {
		name string
		key  string
		src  resources.Catalog
		want map[string]any
	}{
		{
			name: "minimal",
			key:  "sales",
			src:  resources.Catalog{Name: "sales_prod"},
			want: map[string]any{
				"name":          "sales_prod",
				"force_destroy": true,
			},
		},
		{
			name: "with comment and storage root",
			key:  "sales",
			src: resources.Catalog{
				Name:        "sales_prod",
				Comment:     "Sales team catalog",
				StorageRoot: "s3://bucket/root",
			},
			want: map[string]any{
				"name":          "sales_prod",
				"comment":       "Sales team catalog",
				"storage_root":  "s3://bucket/root",
				"force_destroy": true,
			},
		},
		{
			name: "with tags -> properties",
			key:  "sales",
			src: resources.Catalog{
				Name: "sales_prod",
				Tags: map[string]string{"team": "sales", "env": "prod"},
			},
			want: map[string]any{
				"name": "sales_prod",
				"properties": map[string]any{
					"team": "sales",
					"env":  "prod",
				},
				"force_destroy": true,
			},
		},
		{
			name: "defaults name from key when missing",
			key:  "analytics",
			src:  resources.Catalog{},
			want: map[string]any{
				"name":          "analytics",
				"force_destroy": true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)

			out := NewResources()
			err = catalogConverter{}.Convert(t.Context(), tc.key, vin, out)
			require.NoError(t, err)

			got, ok := out.Catalog[tc.key]
			require.True(t, ok)
			assert.Equal(t, tc.want, got.AsAny())
		})
	}
}
