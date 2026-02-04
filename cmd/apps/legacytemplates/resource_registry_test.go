package legacytemplates_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/apps/legacytemplates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceRegistryAllResourcesRegistered(t *testing.T) {
	registry := legacytemplates.GetGlobalRegistry()

	// Verify all expected resource types are registered
	expectedTypes := []legacytemplates.ResourceType{
		legacytemplates.ResourceTypeSQLWarehouse,
		legacytemplates.ResourceTypeServingEndpoint,
		legacytemplates.ResourceTypeExperiment,
		legacytemplates.ResourceTypeDatabase,
		legacytemplates.ResourceTypeUCVolume,
	}

	for _, resType := range expectedTypes {
		handler, ok := registry.Get(resType)
		assert.True(t, ok, "Resource type %s should be registered", resType)
		assert.NotNil(t, handler, "Handler for %s should not be nil", resType)
	}
}

func TestResourceRegistryMetadata(t *testing.T) {
	registry := legacytemplates.GetGlobalRegistry()

	testCases := []struct {
		resourceType legacytemplates.ResourceType
		yamlName     string
		description  string
	}{
		{
			legacytemplates.ResourceTypeSQLWarehouse,
			"sql-warehouse",
			"SQL Warehouse for analytics",
		},
		{
			legacytemplates.ResourceTypeServingEndpoint,
			"serving-endpoint",
			"Model serving endpoint",
		},
		{
			legacytemplates.ResourceTypeExperiment,
			"experiment",
			"MLflow experiment",
		},
		{
			legacytemplates.ResourceTypeDatabase,
			"database",
			"Lakebase database",
		},
		{
			legacytemplates.ResourceTypeUCVolume,
			"uc-volume",
			"Unity Catalog volume",
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.resourceType), func(t *testing.T) {
			handler, ok := registry.Get(tc.resourceType)
			require.True(t, ok)

			metadata := handler.Metadata()
			assert.Equal(t, tc.yamlName, metadata.YamlName)
			assert.Equal(t, tc.description, metadata.Description)
			assert.NotEmpty(t, metadata.FlagNames)
			assert.NotEmpty(t, metadata.VariableNames)
		})
	}
}

func TestResourceRegistryBindingLines(t *testing.T) {
	registry := legacytemplates.GetGlobalRegistry()

	// Test SQL warehouse binding lines
	warehouseHandler, _ := registry.Get(legacytemplates.ResourceTypeSQLWarehouse)
	warehouseMeta := warehouseHandler.Metadata()
	lines := warehouseMeta.BindingLines([]string{"warehouse123"})
	assert.NotEmpty(t, lines)
	assert.Contains(t, strings.Join(lines, "\n"), "sql_warehouse:")
	assert.Contains(t, strings.Join(lines, "\n"), "id: ${var.warehouse_id}")

	// Test UC Volume binding lines
	volumeHandler, _ := registry.Get(legacytemplates.ResourceTypeUCVolume)
	volumeMeta := volumeHandler.Metadata()
	volumeLines := volumeMeta.BindingLines([]string{"my-volume"})
	assert.NotEmpty(t, volumeLines)
	assert.Contains(t, strings.Join(volumeLines, "\n"), "volume:")
	assert.Contains(t, strings.Join(volumeLines, "\n"), "name: ${var.uc_volume}")
}
