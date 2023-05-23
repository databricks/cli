package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateSchematIsInterger(t *testing.T) {
	assert.False(t, isIntegerValue(1.1))
	assert.False(t, isIntegerValue(0.1))
	assert.False(t, isIntegerValue(-0.1))

	assert.True(t, isIntegerValue(-1.0))
	assert.True(t, isIntegerValue(0.0))
	assert.True(t, isIntegerValue(2.0))
}

func TestTemplateSchemaCastFloatToInt(t *testing.T) {
	// define schema for config
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
	var schema Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	require.NoError(t, err)

	// define the config
	configJson := `{
		"int_val":    1,
		"float_val":  2,
		"bool_val":   true,
		"string_val": "main hoon na"
	}`
	var config map[string]any
	err = json.Unmarshal([]byte(configJson), &config)
	require.NoError(t, err)

	// assert types before casting, checking that the integer was indeed loaded
	// as a floating point
	assert.IsType(t, float64(0), config["int_val"])
	assert.IsType(t, float64(0), config["float_val"])
	assert.IsType(t, true, config["bool_val"])
	assert.IsType(t, "abc", config["string_val"])

	err = castFloatToInt(config, &schema)
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
	var schema Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	require.NoError(t, err)

	// define the config
	configJson := `{
		"bar": true
	}`
	var config map[string]any
	err = json.Unmarshal([]byte(configJson), &config)
	require.NoError(t, err)

	err = castFloatToInt(config, &schema)
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
	var schema Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	require.NoError(t, err)

	// define the config
	configJson := `{
		"foo": 1.1
	}`
	var config map[string]any
	err = json.Unmarshal([]byte(configJson), &config)
	require.NoError(t, err)

	err = castFloatToInt(config, &schema)
	assert.ErrorContains(t, err, "expected foo to have integer value but it is 1.1")
}

func TestTemplateSchemaValidateType(t *testing.T) {
	// assert validation passing
	err := validateType(int(0), PropertyTypeInt)
	assert.NoError(t, err)
	err = validateType(int32(1), PropertyTypeInt)
	assert.NoError(t, err)
	err = validateType(int64(1), PropertyTypeInt)
	assert.NoError(t, err)

	err = validateType(float32(1.1), PropertyTypeNumber)
	assert.NoError(t, err)
	err = validateType(float64(1.2), PropertyTypeNumber)
	assert.NoError(t, err)
	err = validateType(int(1), PropertyTypeNumber)
	assert.NoError(t, err)

	err = validateType(false, PropertyTypeBoolean)
	assert.NoError(t, err)

	err = validateType("abc", PropertyTypeString)
	assert.NoError(t, err)

	// assert validation failing for integers
	err = validateType(float64(1.2), PropertyTypeInt)
	assert.ErrorContains(t, err, "expected type integer, but value is 1.2")
	err = validateType(true, PropertyTypeInt)
	assert.ErrorContains(t, err, "expected type integer, but value is true")
	err = validateType("abc", PropertyTypeInt)
	assert.ErrorContains(t, err, "expected type integer, but value is \"abc\"")

	// assert validation failing for floats
	err = validateType(true, PropertyTypeNumber)
	assert.ErrorContains(t, err, "expected type float, but value is true")
	err = validateType("abc", PropertyTypeNumber)
	assert.ErrorContains(t, err, "expected type float, but value is \"abc\"")

	// assert validation failing for boolean
	err = validateType(int(1), PropertyTypeBoolean)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType(float64(1), PropertyTypeBoolean)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType("abc", PropertyTypeBoolean)
	assert.ErrorContains(t, err, "expected type boolean, but value is \"abc\"")

	// assert validation failing for string
	err = validateType(int(1), PropertyTypeString)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(float64(1), PropertyTypeString)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(false, PropertyTypeString)
	assert.ErrorContains(t, err, "expected type string, but value is false")
}

func TestTemplateSchemaValidateConfig(t *testing.T) {
	// define schema for config
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
	var schema Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	require.NoError(t, err)

	// define the config
	config := map[string]any{
		"int_val":    1,
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abc",
	}

	err = schema.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestTemplateSchemaValidateConfigFailsForUnknownField(t *testing.T) {
	// define schema for config
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
	var schema Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	require.NoError(t, err)

	// define the config
	config := map[string]any{
		"foo":        1,
		"float_val":  1.1,
		"bool_val":   true,
		"string_val": "abc",
	}

	err = schema.ValidateConfig(config)
	assert.ErrorContains(t, err, "foo is not defined as an input parameter for the template")
}

func TestTemplateSchemaValidateConfigFailsForWhenIncorrectTypes(t *testing.T) {
	// define schema for config
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
	var schema Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	require.NoError(t, err)

	// define the config
	config := map[string]any{
		"int_val":    1,
		"float_val":  1.1,
		"bool_val":   "true",
		"string_val": "abc",
	}

	err = schema.ValidateConfig(config)
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
	var schema Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	require.NoError(t, err)

	// define the config
	config := map[string]any{
		"int_val": 1,
	}

	err = schema.ValidateConfig(config)
	assert.ErrorContains(t, err, "input parameter string_val is not defined in config")
}
