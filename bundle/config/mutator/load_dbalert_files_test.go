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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDBAlertFiles(t *testing.T) {
	dir := t.TempDir()

	alertJSON := `{
  "display_name": "Test Alert",
  "query_text": "SELECT * FROM table WHERE value > 100",
  "warehouse_id": "abc123",
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
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.alerts.my_alert", []dyn.Location{{
		File: filepath.Join(dir, "databricks.yml"),
	}})

	diags := bundle.Apply(context.Background(), b, mutator.LoadDBAlertFiles())
	require.NoError(t, diags.Error())

	alert := b.Config.Resources.Alerts["my_alert"]
	assert.Equal(t, "Test Alert", alert.DisplayName)
	assert.Equal(t, "SELECT * FROM table WHERE value > 100", alert.QueryText)
	assert.Equal(t, "abc123", alert.WarehouseId)
}

func TestLoadDBAlertFilesWithVariableInterpolation(t *testing.T) {
	dir := t.TempDir()

	alertJSON := `{
  "display_name": "Test Alert ${var.environment}",
  "query_text": "SELECT * FROM table WHERE value > 100",
  "warehouse_id": "abc123",
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
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.alerts.my_alert", []dyn.Location{{
		File: filepath.Join(dir, "databricks.yml"),
	}})

	diags := bundle.Apply(context.Background(), b, mutator.LoadDBAlertFiles())
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "contains bundle variable interpolations")
	assert.Contains(t, diags.Error().Error(), "${var.environment}")
}
