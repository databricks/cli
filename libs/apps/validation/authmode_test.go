package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const authModeManifest = `{
  "plugins": {
    "analytics": {
      "resources": {
        "required": [
          {"type": "sql_warehouse", "resourceKey": "sql-warehouse", "scope": "sql", "fields": {"id": {"env": "WAREHOUSE_ID"}}},
          {"type": "secret", "resourceKey": "secret", "appOnly": true, "fields": {"scope": {"env": "SECRET_SCOPE"}, "key": {"env": "SECRET_KEY"}}}
        ]
      }
    }
  }
}`

func writeAuthModeProject(t *testing.T, appYAML, bundleYAML string) string {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "appkit.plugins.json"), []byte(authModeManifest), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(appYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(bundleYAML), 0o644))
	return dir
}

func TestValidateAuthModes(t *testing.T) {
	const boundSecret = `
resources:
  apps:
    app:
      resources:
        - name: secret
`
	tests := []struct {
		name    string
		appYAML string
		bundle  string
		wantErr string
	}{
		{
			name:    "sp",
			appYAML: "env:\n  - name: WAREHOUSE_ID\n    valueFrom: sql-warehouse\n",
			bundle:  "resources:\n  apps:\n    app:\n      resources:\n        - name: sql-warehouse\n",
		},
		{
			name:    "obo with scope",
			appYAML: "env:\n  - name: WAREHOUSE_ID\n    value: wh1\n",
			bundle:  "resources:\n  apps:\n    app:\n      user_api_scopes:\n        - sql\n",
		},
		{
			name:    "bound with literal value is not obo",
			appYAML: "env:\n  - name: WAREHOUSE_ID\n    value: wh1\n",
			bundle:  "resources:\n  apps:\n    app:\n      resources:\n        - name: sql-warehouse\n",
		},
		{
			name:    "obo missing scope",
			appYAML: "env:\n  - name: WAREHOUSE_ID\n    value: wh1\n",
			bundle:  boundSecret,
			wantErr: `resource "sql-warehouse" is accessed on behalf of the user but its scope "sql" is missing from user_api_scopes in databricks.yml`,
		},
		{
			name:    "app only not bound",
			appYAML: "env:\n  - name: SECRET_SCOPE\n    value: s\n  - name: SECRET_KEY\n    value: k\n",
			bundle:  "resources:\n  apps:\n    app:\n      user_api_scopes:\n        - sql\n",
			wantErr: `resource "secret" must be bound to the app's service principal in databricks.yml, but app.yaml sets SECRET_SCOPE to a literal value`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthModes(writeAuthModeProject(t, tt.appYAML, tt.bundle))
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestValidateAuthModesWithoutManifest(t *testing.T) {
	assert.NoError(t, ValidateAuthModes(t.TempDir()))
}
