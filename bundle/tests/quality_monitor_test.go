package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
)

func assertExpectedMonitor(t *testing.T, p *resources.QualityMonitor) {
	assert.Equal(t, "/Shared/provider-test/databricks_monitoring/main.test.thing1", p.AssetsDir)
	assert.Equal(t, "model_id", p.InferenceLog.ModelIdCol)
	assert.Equal(t, "prediction", p.InferenceLog.PredictionCol)
	assert.Equal(t, catalog.MonitorInferenceLogProblemType("PROBLEM_TYPE_REGRESSION"), p.InferenceLog.ProblemType)
	assert.Equal(t, "timestamp", p.InferenceLog.TimestampCol)
}

func TestMonitorTableNames(t *testing.T) {
	b := loadTarget(t, "./quality_monitor", "development")
	assert.Len(t, b.Config.Resources.QualityMonitors, 1)
	assert.Equal(t, b.Config.Bundle.Mode, config.Development)

	p := b.Config.Resources.QualityMonitors["my_monitor"]
	assert.Equal(t, "main.test.dev", p.TableName)
	assertExpectedMonitor(t, p)
}

func TestMonitorStaging(t *testing.T) {
	b := loadTarget(t, "./quality_monitor", "staging")
	assert.Len(t, b.Config.Resources.QualityMonitors, 1)

	p := b.Config.Resources.QualityMonitors["my_monitor"]
	assert.Equal(t, "main.test.staging", p.TableName)
	assert.Equal(t, "staging", p.OutputSchemaName)
	assertExpectedMonitor(t, p)
}

func TestMonitorProduction(t *testing.T) {
	b := loadTarget(t, "./quality_monitor", "production")
	assert.Len(t, b.Config.Resources.QualityMonitors, 1)

	p := b.Config.Resources.QualityMonitors["my_monitor"]
	assert.Equal(t, "main.test.prod", p.TableName)
	assert.Equal(t, "prod", p.OutputSchemaName)

	assert.Equal(t, "/Shared/provider-test/databricks_monitoring/main.test.thing1", p.AssetsDir)
	assert.Equal(t, "model_id", p.InferenceLog.ModelIdCol)
	assert.Equal(t, "prediction_prod", p.InferenceLog.PredictionCol)
	assert.Equal(t, catalog.MonitorInferenceLogProblemType("PROBLEM_TYPE_REGRESSION"), p.InferenceLog.ProblemType)
	assert.Equal(t, "timestamp_prod", p.InferenceLog.TimestampCol)
}
