package tfdyn

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertVolume(t *testing.T) {
	tests := []struct {
		name string
		key  string
		src  resources.Volume
		want map[string]any
	}{
		{
			name: "managed",
			key:  "landing",
			src: resources.Volume{
				Name:        "landing",
				CatalogName: "sales",
				SchemaName:  "raw",
				VolumeType:  "MANAGED",
				Comment:     "landing zone",
			},
			want: map[string]any{
				"name":         "landing",
				"catalog_name": "sales",
				"schema_name":  "raw",
				"volume_type":  "MANAGED",
				"comment":      "landing zone",
			},
		},
		{
			name: "external",
			key:  "archive",
			src: resources.Volume{
				Name:            "archive",
				CatalogName:     "sales",
				SchemaName:      "raw",
				VolumeType:      "EXTERNAL",
				StorageLocation: "s3://acme-archive/sales",
			},
			want: map[string]any{
				"name":             "archive",
				"catalog_name":     "sales",
				"schema_name":      "raw",
				"volume_type":      "EXTERNAL",
				"storage_location": "s3://acme-archive/sales",
			},
		},
		{
			name: "lowercase volume_type normalised",
			key:  "lower",
			src: resources.Volume{
				Name:        "lower",
				CatalogName: "sales",
				SchemaName:  "raw",
				VolumeType:  "managed",
			},
			want: map[string]any{
				"name":         "lower",
				"catalog_name": "sales",
				"schema_name":  "raw",
				"volume_type":  "MANAGED",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)
			out := NewResources()
			err = volumeConverter{}.Convert(t.Context(), tc.key, vin, out)
			require.NoError(t, err)
			got, ok := out.Volume[tc.key]
			require.True(t, ok)
			assert.Equal(t, tc.want, got.AsAny())
		})
	}
}

func TestConvertVolume_Errors(t *testing.T) {
	tests := []struct {
		name    string
		src     resources.Volume
		wantMsg string
	}{
		{
			name:    "missing catalog_name",
			src:     resources.Volume{Name: "v", SchemaName: "s", VolumeType: "MANAGED"},
			wantMsg: "catalog_name is required",
		},
		{
			name:    "missing schema_name",
			src:     resources.Volume{Name: "v", CatalogName: "c", VolumeType: "MANAGED"},
			wantMsg: "schema_name is required",
		},
		{
			name:    "invalid volume_type",
			src:     resources.Volume{Name: "v", CatalogName: "c", SchemaName: "s", VolumeType: "WEIRD"},
			wantMsg: "volume_type must be MANAGED or EXTERNAL",
		},
		{
			name:    "external without storage_location",
			src:     resources.Volume{Name: "v", CatalogName: "c", SchemaName: "s", VolumeType: "EXTERNAL"},
			wantMsg: "storage_location is required for EXTERNAL",
		},
		{
			name:    "managed with storage_location",
			src:     resources.Volume{Name: "v", CatalogName: "c", SchemaName: "s", VolumeType: "MANAGED", StorageLocation: "s3://x/y"},
			wantMsg: "storage_location must not be set for MANAGED",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)
			out := NewResources()
			err = volumeConverter{}.Convert(t.Context(), "k", vin, out)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}
