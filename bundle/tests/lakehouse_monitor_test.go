package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
)

func assertExpectedLakehouseMonitor(t *testing.T, p *resources.LakehouseMonitor) {
	assert.Equal(t, "/Shared/provider-test/databricks_lakehouse_monitoring/main.test.thing1", p.AssetsDir)
	assert.Equal(t, "model_id", p.InferenceLog.ModelIdCol)
	assert.Equal(t, "prediction", p.InferenceLog.PredictionCol)
	assert.Equal(t, catalog.MonitorInferenceLogProblemType("PROBLEM_TYPE_REGRESSION"), p.InferenceLog.ProblemType)
	assert.Equal(t, "timestamp", p.InferenceLog.TimestampCol)
}

func TestLakehouseMonitorTableNames(t *testing.T) {
	b := loadTarget(t, "./lakehouse_monitor", "development")
	assert.Len(t, b.Config.Resources.LakehouseMonitors, 1)
	assert.Equal(t, b.Config.Bundle.Mode, config.Development)

	p := b.Config.Resources.LakehouseMonitors["my_lakehouse_monitor"]
	assert.Equal(t, "main.test.dev", p.TableName)
	assertExpectedLakehouseMonitor(t, p)
}

func TestLakehouseMonitorStaging(t *testing.T) {
	b := loadTarget(t, "./lakehouse_monitor", "staging")
	assert.Len(t, b.Config.Resources.LakehouseMonitors, 1)

	p := b.Config.Resources.LakehouseMonitors["my_lakehouse_monitor"]
	assert.Equal(t, "main.test.staging", p.TableName)
	assert.Equal(t, "staging", p.OutputSchemaName)
	assertExpectedLakehouseMonitor(t, p)
}

func TestLakehouseMonitorProduction(t *testing.T) {
	b := loadTarget(t, "./lakehouse_monitor", "production")
	assert.Len(t, b.Config.Resources.LakehouseMonitors, 1)

	p := b.Config.Resources.LakehouseMonitors["my_lakehouse_monitor"]
	assert.Equal(t, "main.test.prod", p.TableName)
	assert.Equal(t, "prod", p.OutputSchemaName)

	assert.Equal(t, "/Shared/provider-test/databricks_lakehouse_monitoring/main.test.thing1", p.AssetsDir)
	assert.Equal(t, "model_id", p.InferenceLog.ModelIdCol)
	assert.Equal(t, "prediction_prod", p.InferenceLog.PredictionCol)
	assert.Equal(t, catalog.MonitorInferenceLogProblemType("PROBLEM_TYPE_REGRESSION"), p.InferenceLog.ProblemType)
	assert.Equal(t, "timestamp_prod", p.InferenceLog.TimestampCol)
}
