package mutator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
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

	bundletest.SetLocation(b, "resources.alerts.my_alert", []diag.Location{{
		File: filepath.Join(dir, "databricks.yml"),
	}})

	// Note: This test only verifies that the mutator doesn't error when a file_path is set.
	// The full functionality is tested in acceptance tests where the bundle is properly initialized.
	diags := bundle.Apply(t.Context(), b, mutator.LoadDBAlertFiles())
	require.NoError(t, diags.Error())

	assert.Equal(t, "Test Alert", b.Config.Resources.Alerts["my_alert"].DisplayName)
	assert.Equal(t, "abc123", b.Config.Resources.Alerts["my_alert"].WarehouseId)
	assert.Equal(t, "SELECT * FROM table\nWHERE value > 100\n", b.Config.Resources.Alerts["my_alert"].QueryText)
}

func TestLoadDBAlertFilesRelativeToBundleRoot(t *testing.T) {
	dir := t.TempDir()

	alertJSON := `{
  "query_lines": [
    "SELECT 1"
  ]
}`

	err := os.WriteFile(filepath.Join(dir, "alert.dbalert.json"), []byte(alertJSON), 0o644)
	require.NoError(t, err)

	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Alerts: map[string]*resources.Alert{
					"my_alert": {
						FilePath: "alert.dbalert.json",
						AlertV2: sql.AlertV2{
							DisplayName: "Test Alert",
							WarehouseId: "abc123",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.alerts.my_alert", []diag.Location{{
		File: filepath.Join(dir, "databricks.yml"),
	}})

	diags := bundle.Apply(t.Context(), b, mutator.LoadDBAlertFiles())
	require.NoError(t, diags.Error())

	assert.Equal(t, "SELECT 1\n", b.Config.Resources.Alerts["my_alert"].QueryText)
}

func newSQLFileAlertBundle(t *testing.T, dir, sqlFile string, alert sql.AlertV2) *bundle.Bundle {
	t.Helper()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Resources: config.Resources{
				Alerts: map[string]*resources.Alert{
					"my_alert": {
						FilePath: sqlFile,
						AlertV2:  alert,
					},
				},
			},
		},
	}
	bundletest.SetLocation(b, "resources.alerts.my_alert", []diag.Location{{
		File: filepath.Join(dir, "databricks.yml"),
	}})
	return b
}

func TestLoadDBAlertFilesSQLQueryFile(t *testing.T) {
	dir := t.TempDir()
	query := "SELECT count(*) AS failed\nFROM system.lakeflow.pipeline_update_timeline\nWHERE result_state = 'FAILED'\n"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sql"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sql", "failed_runs.sql"), []byte(query), 0o644))

	evaluation := sql.AlertV2Evaluation{
		ComparisonOperator: sql.ComparisonOperatorGreaterThan,
		Source:             sql.AlertV2OperandColumn{Name: "failed"},
	}
	b := newSQLFileAlertBundle(t, dir, "sql/failed_runs.sql", sql.AlertV2{
		DisplayName:   "Failed runs",
		WarehouseId:   "abc123",
		CustomSummary: "Pipeline failures",
		Evaluation:    evaluation,
	})

	diags := bundle.Apply(t.Context(), b, mutator.LoadDBAlertFiles())
	require.NoError(t, diags.Error())

	// The query comes from the file and every other field stays as configured in YAML.
	alert := b.Config.Resources.Alerts["my_alert"]
	assert.Equal(t, query, alert.QueryText)
	assert.Equal(t, "Failed runs", alert.DisplayName)
	assert.Equal(t, "abc123", alert.WarehouseId)
	assert.Equal(t, "Pipeline failures", alert.CustomSummary)
	assert.Equal(t, evaluation, alert.Evaluation)
	assert.Equal(t, "sql/failed_runs.sql", alert.FilePath)
}

func TestLoadDBAlertFilesSQLQueryFileUppercaseExtension(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "query.SQL"), []byte("SELECT 1"), 0o644))

	b := newSQLFileAlertBundle(t, dir, "query.SQL", sql.AlertV2{DisplayName: "a", WarehouseId: "w"})

	diags := bundle.Apply(t.Context(), b, mutator.LoadDBAlertFiles())
	require.NoError(t, diags.Error())
	assert.Equal(t, "SELECT 1", b.Config.Resources.Alerts["my_alert"].QueryText)
}

func TestLoadDBAlertFilesSQLQueryFileErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content *string
		alert   sql.AlertV2
		summary string
	}{
		{
			name:    "query_text also set",
			content: new("SELECT 1"),
			alert:   sql.AlertV2{DisplayName: "a", WarehouseId: "w", QueryText: "SELECT 2"},
			summary: "query_text cannot be set when file_path points to a .sql file",
		},
		{
			name:    "missing file",
			alert:   sql.AlertV2{DisplayName: "a", WarehouseId: "w"},
			summary: "failed to read alert query file query.sql",
		},
		{
			name:    "empty file",
			content: new("  \n"),
			alert:   sql.AlertV2{DisplayName: "a", WarehouseId: "w"},
			summary: "alert query file query.sql is empty",
		},
		{
			name:    "variable reference",
			content: new("SELECT * FROM ${var.catalog}.sales"),
			alert:   sql.AlertV2{DisplayName: "a", WarehouseId: "w"},
			summary: "alert query file query.sql must not contain variable interpolations.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.content != nil {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "query.sql"), []byte(*tc.content), 0o644))
			}

			b := newSQLFileAlertBundle(t, dir, "query.sql", tc.alert)

			diags := bundle.Apply(t.Context(), b, mutator.LoadDBAlertFiles())
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Contains(t, diags[0].Summary, tc.summary)
		})
	}
}
