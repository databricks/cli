package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceSchema_DoUpdate(t *testing.T) {
	_, client := setupTestServerClient(t)

	adapter := (*ResourceSchema)(nil).New(client)
	ctx := t.Context()

	config := &resources.CreateSchema{
		CatalogName: new("main"),
		Name:        new("test_schema"),
		Comment:     new("original comment"),
		Properties:  map[string]string{"key": "value"},
	}

	id, _, err := adapter.DoCreate(ctx, config)
	require.NoError(t, err)

	config.Comment = new("updated comment")
	config.Properties = map[string]string{"key": "updated_value"}

	_, err = adapter.DoUpdate(ctx, id, config, &PlanEntry{})
	require.NoError(t, err)

	result, err := adapter.DoRead(ctx, id)
	require.NoError(t, err)

	result.CreatedAt = new(int64(0))
	result.UpdatedAt = new(int64(0))
	result.SchemaId = new("")

	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	// browse_only is force-sent from create onwards, matching UC, which returns it on
	// every read including after an update.
	expected := `{
		"browse_only": false,
		"catalog_name": "main",
		"catalog_type": "MANAGED_CATALOG",
		"created_at": 0,
		"created_by": "tester@databricks.com",
		"comment": "updated comment",
		"custom_max_retention_hours": 0,
		"effective_predictive_optimization_flag": {
			"inherited_from_name": "deco-uc-prod-isolated-aws-us-east-1",
			"inherited_from_type": "METASTORE",
			"value": "ENABLE"
		},
		"enable_predictive_optimization": "INHERIT",
		"properties": {"key": "updated_value"},
		"full_name": "main.test_schema",
		"metastore_id": "120efa64-9b68-46ba-be38-f319458430d2",
		"name": "test_schema",
		"owner": "tester@databricks.com",
		"updated_at": 0,
		"updated_by": "tester@databricks.com",
		"schema_id": ""
	}`
	assert.JSONEq(t, expected, string(resultJSON))
}
