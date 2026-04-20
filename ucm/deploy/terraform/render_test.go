package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// golden is the expected main.tf.json body for a 1-catalog + 1-schema +
// 1-grant input. Indentation and key order must match jsonsaver's output
// exactly — converters emit keys in insertion order and jsonsaver preserves
// that order.
const golden = `{
  "resource": {
    "databricks_catalog": {
      "sales": {
        "name": "sales_prod",
        "comment": "sales data",
        "force_destroy": true
      }
    },
    "databricks_schema": {
      "raw": {
        "name": "raw",
        "catalog_name": "sales",
        "force_destroy": true,
        "depends_on": [
          "databricks_catalog.sales"
        ]
      }
    },
    "databricks_grants": {
      "sales_admins": {
        "catalog": "${databricks_catalog.sales.name}",
        "grant": [
          {
            "principal": "sales-admins",
            "privileges": [
              "USE_CATALOG"
            ]
          }
        ],
        "depends_on": [
          "databricks_catalog.sales"
        ]
      }
    }
  }
}
`

func newRenderUcm(t *testing.T) (*ucm.Ucm, string) {
	t.Helper()
	root := t.TempDir()

	cfg, diags := config.LoadFromBytes(filepath.Join(root, "ucm.yml"), []byte(`
ucm:
  name: render-test
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
	cfg.Ucm.Target = "dev"

	return &ucm.Ucm{RootPath: root, Config: *cfg}, root
}

func TestRenderWritesExpectedMainTfJson(t *testing.T) {
	u, _ := newRenderUcm(t)
	workingDir, err := WorkingDir(u)
	require.NoError(t, err)

	tf := &Terraform{
		WorkingDir:    workingDir,
		runnerFactory: defaultRunnerFactory,
	}

	require.NoError(t, tf.Render(t.Context(), u))

	body, err := os.ReadFile(filepath.Join(workingDir, MainConfigFileName))
	require.NoError(t, err)
	assert.Equal(t, golden, string(body))
}

func TestRenderIsIdempotent(t *testing.T) {
	u, _ := newRenderUcm(t)
	workingDir, err := WorkingDir(u)
	require.NoError(t, err)

	tf := &Terraform{WorkingDir: workingDir}

	require.NoError(t, tf.Render(t.Context(), u))
	first, err := os.ReadFile(filepath.Join(workingDir, MainConfigFileName))
	require.NoError(t, err)

	require.NoError(t, tf.Render(t.Context(), u))
	second, err := os.ReadFile(filepath.Join(workingDir, MainConfigFileName))
	require.NoError(t, err)

	assert.Equal(t, string(first), string(second))
}
