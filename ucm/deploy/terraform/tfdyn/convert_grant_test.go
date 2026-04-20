package tfdyn

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGrant(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		src     resources.Grant
		managed func(*Resources)
		want    map[string]any
	}{
		{
			name: "grant on external catalog",
			key:  "sales_admins",
			src: resources.Grant{
				Securable:  resources.Securable{Type: "catalog", Name: "external_cat"},
				Principal:  "sales-admins",
				Privileges: []string{"USE_CATALOG"},
			},
			want: map[string]any{
				"catalog": "external_cat",
				"grant": []any{
					map[string]any{
						"principal":  "sales-admins",
						"privileges": []any{"USE_CATALOG"},
					},
				},
			},
		},
		{
			name: "grant on managed catalog emits depends_on and interpolation",
			key:  "sales_admins",
			src: resources.Grant{
				Securable:  resources.Securable{Type: "catalog", Name: "sales"},
				Principal:  "sales-admins",
				Privileges: []string{"USE_CATALOG", "SELECT"},
			},
			managed: func(r *Resources) {
				r.Catalog["sales"] = dyn.V("placeholder")
			},
			want: map[string]any{
				"catalog": "${databricks_catalog.sales.name}",
				"grant": []any{
					map[string]any{
						"principal":  "sales-admins",
						"privileges": []any{"USE_CATALOG", "SELECT"},
					},
				},
				"depends_on": []any{"databricks_catalog.sales"},
			},
		},
		{
			name: "grant on managed schema",
			key:  "analytics_readers",
			src: resources.Grant{
				Securable:  resources.Securable{Type: "schema", Name: "raw"},
				Principal:  "analysts",
				Privileges: []string{"USE_SCHEMA"},
			},
			managed: func(r *Resources) {
				r.Schema["raw"] = dyn.V("placeholder")
			},
			want: map[string]any{
				"schema": "${databricks_schema.raw.id}",
				"grant": []any{
					map[string]any{
						"principal":  "analysts",
						"privileges": []any{"USE_SCHEMA"},
					},
				},
				"depends_on": []any{"databricks_schema.raw"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)

			out := NewResources()
			if tc.managed != nil {
				tc.managed(out)
			}
			err = grantConverter{}.Convert(t.Context(), tc.key, vin, out)
			require.NoError(t, err)

			got, ok := out.Grants[tc.key]
			require.True(t, ok)
			assert.Equal(t, tc.want, got.AsAny())
		})
	}
}

func TestConvertGrantUnsupportedSecurable(t *testing.T) {
	src := resources.Grant{
		Securable:  resources.Securable{Type: "volume", Name: "v1"},
		Principal:  "x",
		Privileges: []string{"READ_VOLUME"},
	}
	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	out := NewResources()
	err = grantConverter{}.Convert(t.Context(), "vol_grant", vin, out)
	require.ErrorContains(t, err, `unsupported securable type "volume"`)
}
