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

func TestConvertConnection(t *testing.T) {
	tests := []struct {
		name string
		key  string
		src  resources.Connection
		want map[string]any
	}{
		{
			name: "mysql minimal",
			key:  "sales_mysql",
			src: resources.Connection{CreateConnection: catalog.CreateConnection{Name: "sales_mysql", ConnectionType: catalog.ConnectionType("MYSQL"), Options: map[string]string{
					"host": "mysql.acme.internal",
					"port": "3306",
					"user": "reader",
				}}},
			want: map[string]any{
				"name":            "sales_mysql",
				"connection_type": "MYSQL",
				"options": map[string]any{
					"host": "mysql.acme.internal",
					"port": "3306",
					"user": "reader",
				},
			},
		},
		{
			name: "all fields",
			key:  "pg",
			src: resources.Connection{CreateConnection: catalog.CreateConnection{Name: "pg", ConnectionType: catalog.ConnectionType("POSTGRESQL"), Options: map[string]string{"host": "pg.acme.internal"}, Comment: "prod pg", Properties: map[string]string{"purpose": "analytics"}, ReadOnly: true}},
			want: map[string]any{
				"name":            "pg",
				"connection_type": "POSTGRESQL",
				"options":         map[string]any{"host": "pg.acme.internal"},
				"comment":         "prod pg",
				"properties":      map[string]any{"purpose": "analytics"},
				"read_only":       true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)
			out := NewResources()
			err = connectionConverter{}.Convert(t.Context(), tc.key, vin, out)
			require.NoError(t, err)
			got, ok := out.Connection[tc.key]
			require.True(t, ok)
			assert.Equal(t, tc.want, got.AsAny())
		})
	}
}

func TestConvertConnection_Errors(t *testing.T) {
	tests := []struct {
		name    string
		src     resources.Connection
		wantMsg string
	}{
		{
			name:    "missing connection_type",
			src:     resources.Connection{CreateConnection: catalog.CreateConnection{Name: "c", Options: map[string]string{"host": "x"}}},
			wantMsg: "connection_type is required",
		},
		{
			name:    "missing options",
			src:     resources.Connection{CreateConnection: catalog.CreateConnection{Name: "c", ConnectionType: catalog.ConnectionType("MYSQL")}},
			wantMsg: "options is required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vin, err := convert.FromTyped(tc.src, dyn.NilValue)
			require.NoError(t, err)
			out := NewResources()
			err = connectionConverter{}.Convert(t.Context(), "k", vin, out)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}
