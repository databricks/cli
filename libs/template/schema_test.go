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
		"int_val": {
			"type": "integer"
		},
		"float_val": {
			"type": "float"
		},
		"bool_val": {
			"type": "boolean"
		},
		"string_val": {
			"type": "string"
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

	err = schema.CastFloatToInt(config)
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
		"foo": {
			"type": "integer"
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

	err = schema.CastFloatToInt(config)
	assert.ErrorContains(t, err, "bar is not defined as an input parameter for the template")
}
