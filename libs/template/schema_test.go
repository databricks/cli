package template

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSchema(t *testing.T) *schema.Schema {
	schemaJson := `{
		"properties": {
			"int_val": {
				"type": "integer"
			},
			"float_val": {
				"type": "number"
			},
			"bool_val": {
				"type": "boolean"
			},
			"string_val": {
				"type": "string"
			}
		}
	}`
	var jsonSchema schema.Schema
	err := json.Unmarshal([]byte(schemaJson), &jsonSchema)
	require.NoError(t, err)
	return &jsonSchema
}

func TestTemplateSchemaIsInteger(t *testing.T) {
	assert.False(t, isIntegerValue(1.1))
	assert.False(t, isIntegerValue(0.1))
	assert.False(t, isIntegerValue(-0.1))

	assert.True(t, isIntegerValue(-1.0))
	assert.True(t, isIntegerValue(0.0))
	assert.True(t, isIntegerValue(2.0))
}

func TestTemplateSchemaCastFloatToInt(t *testing.T) {
	// define schema for config
	jsonSchema := testSchema(t)

	// define the config
	configJson := `{
		"int_val":    1,
		"float_val":  2,
		"bool_val":   true,
		"string_val": "main hoon na"
	}`
	var config map[string]any
	err := json.Unmarshal([]byte(configJson), &config)
	require.NoError(t, err)

	// assert types before casting, checking that the integer was indeed loaded
	// as a floating point
	assert.IsType(t, float64(0), config["int_val"])
	assert.IsType(t, float64(0), config["float_val"])
	assert.IsType(t, true, config["bool_val"])
	assert.IsType(t, "abc", config["string_val"])

	err = castFloatConfigValuesToInt(config, jsonSchema)
	require.NoError(t, err)

	// assert type after casting, that the float value was converted to an integer
	// for int_val.
	assert.IsType(t, int(0), config["int_val"])
	assert.IsType(t, float64(0), config["float_val"])
	assert.IsType(t, true, config["bool_val"])
	assert.IsType(t, "abc", config["string_val"])
}

func TestTemplateSchemaCastFloatToIntFailsForUnknownTypes(t *testing.T) {
	// define schema for config
	schemaJson := `{
		"properties": {
			"foo": {
				"type": "integer"
			}
		}
	}`
	var jsonSchema schema.Schema
	err := json.Unmarshal([]byte(schemaJson), &jsonSchema)
	require.NoError(t, err)

	// define the config
	configJson := `{
		"bar": true
	}`
	var config map[string]any
	err = json.Unmarshal([]byte(configJson), &config)
	require.NoError(t, err)

	err = castFloatConfigValuesToInt(config, &jsonSchema)
	assert.ErrorContains(t, err, "bar is not defined as an input parameter for the template")
}

func TestTemplateSchemaCastFloatToIntFailsWhenWithNonIntValues(t *testing.T) {
	// define schema for config
	schemaJson := `{
		"properties": {
			"foo": {
				"type": "integer"
			}
		}
	}`
	var jsonSchema schema.Schema
	err := json.Unmarshal([]byte(schemaJson), &jsonSchema)
	require.NoError(t, err)

	// define the config
	configJson := `{
		"foo": 1.1
	}`
	var config map[string]any
	err = json.Unmarshal([]byte(configJson), &config)
	require.NoError(t, err)

	err = castFloatConfigValuesToInt(config, &jsonSchema)
	assert.ErrorContains(t, err, "expected foo to have integer value but it is 1.1")
}

func TestTemplateSchemaValidateType(t *testing.T) {
	// assert validation passing
	err := validateType(int(0), schema.IntegerType)
	assert.NoError(t, err)
	err = validateType(int32(1), schema.IntegerType)
	assert.NoError(t, err)
	err = validateType(int64(1), schema.IntegerType)
	assert.NoError(t, err)

	err = validateType(float32(1.1), schema.NumberType)
	assert.NoError(t, err)
	err = validateType(float64(1.2), schema.NumberType)
	assert.NoError(t, err)
	err = validateType(int(1), schema.NumberType)
	assert.NoError(t, err)

	err = validateType(false, schema.BooleanType)
	assert.NoError(t, err)

	err = validateType("abc", schema.StringType)
	assert.NoError(t, err)

	// assert validation failing for integers
	err = validateType(float64(1.2), schema.IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is 1.2")
	err = validateType(true, schema.IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is true")
	err = validateType("abc", schema.IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is \"abc\"")

	// assert validation failing for floats
	err = validateType(true, schema.NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is true")
	err = validateType("abc", schema.NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is \"abc\"")

	// assert validation failing for boolean
	err = validateType(int(1), schema.BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType(float64(1), schema.BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType("abc", schema.BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is \"abc\"")

	// assert validation failing for string
	err = validateType(int(1), schema.StringType)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(float64(1), schema.StringType)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(false, schema.StringType)
	assert.ErrorContains(t, err, "expected type string, but value is false")
}

func TestTemplateSchemaValidateConfig(t *testing.T) {
	// define schema for config
	jsonSchema := testSchema(t)

	// define the config
	config := map[string]any{
		"int_val":    1,
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abc",
	}

	err := validateConfigValueTypes(config, jsonSchema)
	assert.NoError(t, err)
}

func TestTemplateSchemaValidateConfigFailsForUnknownField(t *testing.T) {
	// define schema for config
	jsonSchema := testSchema(t)

	// define the config
	config := map[string]any{
		"foo":        1,
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abc",
	}

	err := validateConfigValueTypes(config, jsonSchema)
	assert.ErrorContains(t, err, "foo is not defined as an input parameter for the template")
}

func TestTemplateSchemaValidateConfigFailsForWhenIncorrectTypes(t *testing.T) {
	// define schema for config
	jsonSchema := testSchema(t)

	// define the config
	config := map[string]any{
		"int_val":    1,
		"float_val":  1.1,
		"bool_val":   "true",
		"string_val": "abc",
	}

	err := validateConfigValueTypes(config, jsonSchema)
	assert.ErrorContains(t, err, "incorrect type for bool_val. expected type boolean, but value is \"true\"")
}

func TestTemplateSchemaValidateConfigFailsForWhenMissingInputParams(t *testing.T) {
	// define schema for config
	schemaJson := `{
		"properties": {
			"int_val": {
				"type": "integer"
			},
			"string_val": {
				"type": "string"
			}
		}
		}`
	var jsonSchema schema.Schema
	err := json.Unmarshal([]byte(schemaJson), &jsonSchema)
	require.NoError(t, err)

	// define the config
	config := map[string]any{
		"int_val": 1,
	}

	err = assignDefaultConfigValues(config, &jsonSchema)
	assert.ErrorContains(t, err, "input parameter string_val is not defined in config")
}

func TestTemplateDefaultAssignment(t *testing.T) {
	// define schema for config
	schemaJson := `{
		"properties": {
			"foo": {
				"type": "integer",
				"default": 1
			}
		}
		}`
	var jsonSchema schema.Schema
	err := json.Unmarshal([]byte(schemaJson), &jsonSchema)
	require.NoError(t, err)

	// define the config
	config := map[string]any{}

	err = assignDefaultConfigValues(config, &jsonSchema)
	assert.NoError(t, err)
	assert.Equal(t, 1.0, config["foo"])
}
