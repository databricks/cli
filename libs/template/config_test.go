package template

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSchema(t *testing.T) *jsonschema.Schema {
	schemaJson := `{
		"properties": {
			"int_val": {
				"type": "integer",
				"default": 123
			},
			"float_val": {
				"type": "number"
			},
			"bool_val": {
				"type": "boolean"
			},
			"string_val": {
				"type": "string",
				"default": "abc"
			}
		}
	}`
	var jsonSchema jsonschema.Schema
	err := json.Unmarshal([]byte(schemaJson), &jsonSchema)
	require.NoError(t, err)
	return &jsonSchema
}

func TestTemplateConfigAssignValuesFromFile(t *testing.T) {
	c := config{
		schema: testSchema(t),
		values: make(map[string]any),
	}

	err := c.assignValuesFromFile("./testdata/config-assign-from-file/config.json")
	assert.NoError(t, err)

	assert.Equal(t, int(1), c.values["int_val"])
	assert.Equal(t, float64(2), c.values["float_val"])
	assert.Equal(t, true, c.values["bool_val"])
	assert.Equal(t, "hello", c.values["string_val"])
}

func TestTemplateConfigAssignValuesFromFileForUnknownField(t *testing.T) {
	c := config{
		schema: testSchema(t),
		values: make(map[string]any),
	}

	err := c.assignValuesFromFile("./testdata/config-assign-from-file-unknown-property/config.json")
	assert.EqualError(t, err, "unknown_prop is not defined as an input parameter for the template")
}

func TestTemplateConfigAssignValuesFromFileForInvalidIntegerValue(t *testing.T) {
	c := config{
		schema: testSchema(t),
		values: make(map[string]any),
	}

	err := c.assignValuesFromFile("./testdata/config-assign-from-file-invalid-int/config.json")
	assert.EqualError(t, err, "failed to cast value abc of property int_val from file ./testdata/config-assign-from-file-invalid-int/config.json to an integer: cannot convert \"abc\" to an integer")
}

func TestTemplateConfigAssignDefaultValues(t *testing.T) {
	c := config{
		schema: testSchema(t),
		values: make(map[string]any),
	}

	err := c.assignDefaultValues()
	assert.NoError(t, err)

	assert.Len(t, c.values, 2)
	assert.Equal(t, "abc", c.values["string_val"])
	assert.Equal(t, int(123), c.values["int_val"])
}

func TestTemplateConfigValidateValuesDefined(t *testing.T) {
	c := config{
		schema: testSchema(t),
		values: map[string]any{
			"int_val":   1,
			"float_val": 1.0,
			"bool_val":  false,
		},
	}

	err := c.validateValuesDefined()
	assert.EqualError(t, err, "no value has been assigned to input parameter string_val")
}

func TestTemplateConfigValidateTypeForValidConfig(t *testing.T) {
	c := &config{
		schema: testSchema(t),
		values: map[string]any{
			"int_val":    1,
			"float_val":  1.1,
			"bool_val":   true,
			"string_val": "abcd",
		},
	}

	err := c.validateValuesType()
	assert.NoError(t, err)

	err = c.validate()
	assert.NoError(t, err)
}

func TestTemplateConfigValidateTypeForUnknownField(t *testing.T) {
	c := &config{
		schema: testSchema(t),
		values: map[string]any{
			"unknown_prop": 1,
		},
	}

	err := c.validateValuesType()
	assert.EqualError(t, err, "unknown_prop is not defined as an input parameter for the template")
}

func TestTemplateConfigValidateTypeForInvalidType(t *testing.T) {
	c := &config{
		schema: testSchema(t),
		values: map[string]any{
			"int_val":    "this-should-be-an-int",
			"float_val":  1.1,
			"bool_val":   true,
			"string_val": "abcd",
		},
	}

	err := c.validateValuesType()
	assert.EqualError(t, err, `incorrect type for int_val. expected type integer, but value is "this-should-be-an-int"`)
}
