package mutator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDBAlertFiles(t *testing.T) {
	dir := t.TempDir()

	alertJSON := `{
  "query_lines": [
    "SELECT * FROM table",
    "WHERE value > 100"
  ],
  "schedule": {
    "quartz_cron_expression": "0 0 12 * * ?",
    "timezone_id": "UTC"
  },
  "evaluation": {
    "execution_condition": "ALL_ROWS_MATCH",
    "condition": {
      "op": "GREATER_THAN",
      "operand": {
        "column": {
          "name": "value"
        }
      },
      "threshold": {
        "value": {
          "double_value": 100.0
        }
      }
    }
  }
}`

	err := os.WriteFile(filepath.Join(dir, "alert.dbalert.json"), []byte(alertJSON), 0o644)
	require.NoError(t, err)

	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Alerts: map[string]*resources.Alert{
					"my_alert": {
						FilePath: filepath.Join(dir, "alert.dbalert.json"),
						AlertV2: sql.AlertV2{
							DisplayName: "Test Alert",
							WarehouseId: "abc123",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.alerts.my_alert", []dyn.Location{{
		File: filepath.Join(dir, "databricks.yml"),
	}})

	// Note: This test only verifies that the mutator doesn't error when a file_path is set.
	// The full functionality is tested in acceptance tests where the bundle is properly initialized.
	diags := bundle.Apply(context.Background(), b, mutator.LoadDBAlertFiles())
	require.NoError(t, diags.Error())

	assert.Equal(t, "Test Alert", b.Config.Resources.Alerts["my_alert"].DisplayName)
	assert.Equal(t, "abc123", b.Config.Resources.Alerts["my_alert"].WarehouseId)
	assert.Equal(t, "SELECT * FROM table\nWHERE value > 100\n", b.Config.Resources.Alerts["my_alert"].QueryText)
}
