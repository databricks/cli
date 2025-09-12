package tnresources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceSchema_DoUpdate_WithUnsupportedForceSendFields(t *testing.T) {
	_, client := setupTestServerClient(t)

	adapter := (*ResourceSchema)(nil).New(client)
	ctx := context.Background()

	config := &catalog.CreateSchema{
		CatalogName:     "main",
		Name:            "test_schema",
		Comment:         "original comment",
		Properties:      map[string]string{"key": "value"},
		StorageRoot:     "",
		ForceSendFields: nil,
	}

	id, _, err := adapter.DoCreate(ctx, config)
	require.NoError(t, err)

	config.Comment = "updated comment"
	config.Properties = map[string]string{"key": "updated_value"}
	config.ForceSendFields = []string{
		"Comment",
		"Properties",
		"EnablePredictiveOptimization", // Unsupported - should be filtered out
		"NewName",                      // Unsupported - should be filtered out
		"Owner",                        // Unsupported - should be filtered out
	}

	_, err = adapter.DoUpdate(ctx, id, config)
	require.NoError(t, err)

	result, err := adapter.DoRefresh(ctx, id)
	require.NoError(t, err)

	resultJSON, err := json.Marshal(result)
	require.NoError(t, err)
	expected := `{
		"catalog_name": "main",
		"comment": "updated comment",
		"properties": {"key": "updated_value"},
		"full_name": "main.test_schema",
		"name": "test_schema"
	}`
	assert.JSONEq(t, expected, string(resultJSON))
}
