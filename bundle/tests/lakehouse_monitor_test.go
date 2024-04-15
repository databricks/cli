package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
)

func assertExpectedLakehouseMonitor(t *testing.T, p *resources.LakehouseMonitor) {
	assert.Equal(t, "test", p.OutputSchemaName)
	assert.Equal(t, "/Shared/provider-test/databricks_lakehouse_monitoring/main.test.thing1", p.AssetsDir)
	assert.Equal(t, "model_id", p.InferenceLog.ModelIdCol)
	assert.Equal(t, "prediction", p.InferenceLog.PredictionCol)
	assert.Equal(t, catalog.MonitorInferenceLogProfileTypeProblemType("PROBLEM_TYPE_REGRESSION"), p.InferenceLog.ProblemType)
	assert.Equal(t, "timestamp", p.InferenceLog.TimestampCol)

}

func TestLakehouseMonitorDevelepment(t *testing.T) {
	b := loadTarget(t, "./lakehouse_monitor", "development")
	assert.Len(t, b.Config.Resources.LakehouseMonitors, 1)
	assert.Equal(t, b.Config.Bundle.Mode, config.Development)

	p := b.Config.Resources.LakehouseMonitors["my_lakehouse_monitor"]
	assert.Equal(t, "main.test.dev", p.FullName)
	assertExpectedLakehouseMonitor(t, p)
}

func TestLakehouseMonitorStaging(t *testing.T) {
	b := loadTarget(t, "./lakehouse_monitor", "staging")
	assert.Len(t, b.Config.Resources.LakehouseMonitors, 1)

	p := b.Config.Resources.LakehouseMonitors["my_lakehouse_monitor"]
	assert.Equal(t, "main.test.staging", p.FullName)
	assertExpectedLakehouseMonitor(t, p)
}

func TestLakehouseMonitorProduction(t *testing.T) {
	b := loadTarget(t, "./lakehouse_monitor", "production")
	assert.Len(t, b.Config.Resources.LakehouseMonitors, 1)

	p := b.Config.Resources.LakehouseMonitors["my_lakehouse_monitor"]
	assert.Equal(t, "main.test.prod", p.FullName)
	assertExpectedLakehouseMonitor(t, p)
}
