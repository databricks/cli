package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresDataApiSettings(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "parent only",
			yaml: `
resources:
  postgres_data_apis:
    foo:
      parent: projects/p/branches/b/databases/d
`,
			wantErr: true,
		},
		{
			name: "parent and lifecycle only",
			yaml: `
resources:
  postgres_data_apis:
    foo:
      parent: projects/p/branches/b/databases/d
      lifecycle:
        prevent_destroy: true
`,
			wantErr: true,
		},
		{
			name: "parent and a setting",
			yaml: `
resources:
  postgres_data_apis:
    foo:
      parent: projects/p/branches/b/databases/d
      db_schemas:
        - public
`,
		},
		{
			name: "setting given as a reference",
			yaml: `
resources:
  postgres_data_apis:
    foo:
      parent: projects/p/branches/b/databases/d
      db_max_rows: ${var.max_rows}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tc.yaml))
			require.Empty(t, diags)
			b := &bundle.Bundle{Config: *root}

			diags = PostgresDataApiSettings().Apply(t.Context(), b)
			if !tc.wantErr {
				assert.Empty(t, diags)
				return
			}
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Equal(t, "postgres_data_apis.foo: at least one Data API setting besides parent is required, for example db_schemas", diags[0].Summary)
			assert.Equal(t, "resources.postgres_data_apis.foo", diags[0].Paths[0].String())
			require.Len(t, diags[0].Locations, 1)
			assert.Equal(t, "databricks.yml", diags[0].Locations[0].File)
		})
	}
}
