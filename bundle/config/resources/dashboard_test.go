package resources

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
)

func TestDashboardConfigIsSupersetOfSDKDashboard(t *testing.T) {
	configType := reflect.TypeOf(DashboardConfig{})
	sdkType := reflect.TypeOf(dashboards.Dashboard{})

	// Helper function to extract JSON tag name
	getJSONTagName := func(tag string) string {
		if tag == "" || tag == "-" {
			return ""
		}
		// Remove omitempty and other options from the tag
		for i, c := range tag {
			if c == ',' {
				return tag[:i]
			}
		}
		return tag
	}

	// Create a map of SDK fields by name and their JSON tags
	sdkFields := make(map[string]string)
	for i := range sdkType.NumField() {
		field := sdkType.Field(i)
		jsonTag := field.Tag.Get("json")
		jsonName := getJSONTagName(jsonTag)
		if jsonName != "" {
			sdkFields[field.Name] = jsonName
		}
	}

	// Create a map of config fields by name and their JSON tags
	configFields := make(map[string]string)
	for i := range configType.NumField() {
		field := configType.Field(i)
		jsonTag := field.Tag.Get("json")
		jsonName := getJSONTagName(jsonTag)
		if jsonName != "" {
			configFields[field.Name] = jsonName
		}
	}

	// Verify that every field in SDK type exists in Config type with the same JSON tag
	for fieldName, sdkJSONTag := range sdkFields {
		configJSONTag, exists := configFields[fieldName]
		assert.True(t, exists, "Field %s from dashboards.Dashboard is missing in DashboardConfig", fieldName)
		if exists {
			assert.Equal(t, sdkJSONTag, configJSONTag,
				"Field %s has different JSON tag: SDK has %q, DashboardConfig has %q",
				fieldName, sdkJSONTag, configJSONTag)
		}
	}
}

func TestDashboardConfigWithDatasetCatalogSchema(t *testing.T) {
	jsonConfig := `{
		"display_name": "Test Dashboard",
		"warehouse_id": "test_warehouse_id",
		"dataset_catalog": "main",
		"dataset_schema": "default",
		"embed_credentials": true,
		"serialized_dashboard": "{\"key\": \"value\"}"
	}`

	var config DashboardConfig
	err := json.Unmarshal([]byte(jsonConfig), &config)
	assert.NoError(t, err)

	assert.Equal(t, "Test Dashboard", config.DisplayName)
	assert.Equal(t, "test_warehouse_id", config.WarehouseId)
	assert.Equal(t, "main", config.DatasetCatalog)
	assert.Equal(t, "default", config.DatasetSchema)
	assert.True(t, config.EmbedCredentials)
	assert.Equal(t, `{"key": "value"}`, config.SerializedDashboard)
}
