package dsc

import (
	"fmt"
	"reflect"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestResource struct {
	Name       string `json:"name"`
	Value      string `json:"value,omitempty"`
	Count      int    `json:"count"`
	IsRequired bool   `json:"is_required"`
}

func TestGenerateSchema(t *testing.T) {
	schema, err := GenerateSchema(reflect.TypeOf(TestResource{}))
	require.NoError(t, err)

	schemaMap, ok := schema.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", schemaMap["$schema"])
	assert.Equal(t, "object", fmt.Sprint(schemaMap["type"]))

	props, ok := schemaMap["properties"].(map[string]any)
	require.True(t, ok)

	existProp, ok := props["_exist"].(map[string]any)
	require.True(t, ok, "_exist property should be added")
	assert.Equal(t, "boolean", existProp["type"])
	assert.Equal(t, true, existProp["default"])

	_, ok = props["name"]
	assert.True(t, ok, "name property should exist")

	_, ok = props["count"]
	assert.True(t, ok, "count property should exist")
}

func TestGenerateSchemaWithDescriptions(t *testing.T) {
	descriptions := PropertyDescriptions{
		"name":  "The name of the resource",
		"value": "The value to store",
	}

	schema, err := GenerateSchemaWithDescriptions(reflect.TypeOf(TestResource{}), descriptions)
	require.NoError(t, err)

	schemaMap, ok := schema.(map[string]any)
	require.True(t, ok)

	props, ok := schemaMap["properties"].(map[string]any)
	require.True(t, ok)

	nameProp, ok := props["name"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "The name of the resource", nameProp["description"])

	valueProp, ok := props["value"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "The value to store", valueProp["description"])
}

func TestGenerateSchemaWithOptions(t *testing.T) {
	opts := SchemaOptions{
		Descriptions: PropertyDescriptions{
			"name": "Resource name",
		},
		SchemaDescription: "Schema for test resources",
		ResourceName:      "test resource",
	}

	schema, err := GenerateSchemaWithOptions(reflect.TypeOf(TestResource{}), opts)
	require.NoError(t, err)

	schemaMap, ok := schema.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "Schema for test resources", schemaMap["description"])

	props := schemaMap["properties"].(map[string]any)
	existProp := props["_exist"].(map[string]any)
	assert.Equal(t, "Indicates whether the test resource should exist.", existProp["description"])
}

func TestBuildMetadata(t *testing.T) {
	cfg := MetadataConfig{
		ResourceType:      "Test.DSC/Resource",
		Description:       "Test resource description",
		SchemaDescription: "Schema description",
		ResourceName:      "test",
		Tags:              []string{"test", "example"},
		Descriptions:      PropertyDescriptions{"name": "The name"},
		SchemaType:        reflect.TypeOf(TestResource{}),
	}

	metadata := BuildMetadata(cfg)

	assert.Equal(t, "Test.DSC/Resource", metadata.Type)
	assert.Equal(t, "0.1.0", metadata.Version)
	assert.Equal(t, "Test resource description", metadata.Description)
	assert.Equal(t, []string{"test", "example"}, metadata.Tags)
	assert.Equal(t, "Success", metadata.ExitCodes["0"])
	assert.Equal(t, "Error", metadata.ExitCodes["1"])
	assert.NotNil(t, metadata.Schema.Embedded)
}

func TestBuildMetadataWithCustomVersion(t *testing.T) {
	cfg := MetadataConfig{
		ResourceType: "Test.DSC/Resource",
		Version:      "1.2.3",
		Description:  "Test",
		SchemaType:   reflect.TypeOf(TestResource{}),
	}

	metadata := BuildMetadata(cfg)
	assert.Equal(t, "1.2.3", metadata.Version)
}

func TestBuildMetadataDefaultVersion(t *testing.T) {
	cfg := MetadataConfig{
		ResourceType: "Test.DSC/Resource",
		Description:  "Test",
		SchemaType:   reflect.TypeOf(TestResource{}),
	}

	metadata := BuildMetadata(cfg)
	assert.Equal(t, "0.1.0", metadata.Version)
}
