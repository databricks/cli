package tfdyn

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertExternalLocation(t *testing.T) {
	tests := []struct {
		name string
		key  string
		src  resources.ExternalLocation
		want map[string]any
	}{
		{
			name: "minimal",
			key:  "sales_loc",
			src: resources.ExternalLocation{
				Name:           "sales_loc",
				Url:            "s3://acme-sales/prod",
				CredentialName: "sales_cred",
			},
			want: map[string]any{
				"name":            "sales_loc",
				"url":             "s3://acme-sales/prod",
				"credential_name": "sales_cred",
			},
		},
		{
			name: "all fields",
			key:  "ro_loc",
			src: resources.ExternalLocation{
				Name:           "ro_loc",
				Url:            "abfss://data@acme.dfs.core.windows.net/ro",
				CredentialName: "shared_cred",
				Comment:        "read-only location",
				ReadOnly:       true,
				SkipValidation: true,
				Fallback:       true,
			},
			want: map[string]any{
				"name":            "ro_loc",
				"url":             "abfss://data@acme.dfs.core.windows.net/ro",
				"credential_name": "shared_cred",
				"comment":         "read-only location",
				"read_only":       true,
				"skip_validation": true,
				"fallback":        true,
			},
		},
		{
			name: "defaults name from key",
			key:  "inferred",
			src: resources.ExternalLocation{
				Url:            "gs://acme/prod",
				CredentialName: "cred",
			},
			want: map[string]any{
				"name":            "inferred",
				"url":             "gs://acme/prod",
				"credential_name": "cred",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)
			out := NewResources()
			err = externalLocationConverter{}.Convert(t.Context(), tc.key, vin, out)
			require.NoError(t, err)
			got, ok := out.ExternalLocation[tc.key]
			require.True(t, ok)
			assert.Equal(t, tc.want, got.AsAny())
		})
	}
}

func TestConvertExternalLocation_ErrorsOnMissingUrl(t *testing.T) {
	vin, err := convert.FromTyped(resources.ExternalLocation{Name: "bad", CredentialName: "c"}, dyn.NilValue)
	require.NoError(t, err)
	out := NewResources()
	err = externalLocationConverter{}.Convert(t.Context(), "bad", vin, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

func TestConvertExternalLocation_ErrorsOnMissingCredentialName(t *testing.T) {
	vin, err := convert.FromTyped(resources.ExternalLocation{Name: "bad", Url: "s3://x/y"}, dyn.NilValue)
	require.NoError(t, err)
	out := NewResources()
	err = externalLocationConverter{}.Convert(t.Context(), "bad", vin, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credential_name is required")
}
