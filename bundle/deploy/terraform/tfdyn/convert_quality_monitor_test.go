package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertQualityMonitor(t *testing.T) {
	src := resources.QualityMonitor{
		TableName: "test_table_name",
		CreateMonitor: catalog.CreateMonitor{
			AssetsDir:        "assets_dir",
			OutputSchemaName: "output_schema_name",
			InferenceLog: &catalog.MonitorInferenceLog{
				ModelIdCol:    "model_id",
				PredictionCol: "test_prediction_col",
				ProblemType:   "PROBLEM_TYPE_CLASSIFICATION",
			},
		},
	}
	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	ctx := context.Background()
	out := schema.NewResources()
	err = qualityMonitorConverter{}.Convert(ctx, "my_monitor", vin, out)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"assets_dir":         "assets_dir",
		"output_schema_name": "output_schema_name",
		"table_name":         "test_table_name",
		"inference_log": map[string]any{
			"model_id_col":   "model_id",
			"prediction_col": "test_prediction_col",
			"problem_type":   "PROBLEM_TYPE_CLASSIFICATION",
		},
	}, out.QualityMonitor["my_monitor"])
}
