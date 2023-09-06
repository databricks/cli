package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInstanceAdditionalPropertiesPermitted(t *testing.T) {
	instance := map[string]any{
		"int_val":                1,
		"float_val":              1.0,
		"bool_val":               false,
		"an_additional_property": "abc",
	}

	schema, err := Load("./testdata/instance-validate/test-schema.json")
	require.NoError(t, err)

	err = schema.validateAdditionalProperties(instance)
	assert.NoError(t, err)

	err = schema.ValidateInstance(instance)
	assert.NoError(t, err)
}

func TestValidateInstanceAdditionalPropertiesForbidden(t *testing.T) {
	instance := map[string]any{
		"int_val":                1,
		"float_val":              1.0,
		"bool_val":               false,
		"an_additional_property": "abc",
	}

	schema, err := Load("./testdata/instance-validate/test-schema-no-additional-properties.json")
	require.NoError(t, err)

	err = schema.validateAdditionalProperties(instance)
	assert.EqualError(t, err, "property an_additional_property is not defined in the schema")

	err = schema.ValidateInstance(instance)
	assert.EqualError(t, err, "property an_additional_property is not defined in the schema")

	instanceWOAdditionalProperties := map[string]any{
		"int_val":   1,
		"float_val": 1.0,
		"bool_val":  false,
	}

	err = schema.validateAdditionalProperties(instanceWOAdditionalProperties)
	assert.NoError(t, err)

	err = schema.ValidateInstance(instanceWOAdditionalProperties)
	assert.NoError(t, err)
}

func TestValidateInstanceTypes(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema.json")
	require.NoError(t, err)

	validInstance := map[string]any{
		"int_val":   1,
		"float_val": 1.0,
		"bool_val":  false,
	}

	err = schema.validateTypes(validInstance)
	assert.NoError(t, err)

	err = schema.ValidateInstance(validInstance)
	assert.NoError(t, err)

	invalidInstance := map[string]any{
		"int_val":   "abc",
		"float_val": 1.0,
		"bool_val":  false,
	}

	err = schema.validateTypes(invalidInstance)
	assert.EqualError(t, err, "incorrect type for property int_val: expected type integer, but value is \"abc\"")

	err = schema.ValidateInstance(invalidInstance)
	assert.EqualError(t, err, "incorrect type for property int_val: expected type integer, but value is \"abc\"")
}

func TestValidateInstanceRequired(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema-some-fields-required.json")
	require.NoError(t, err)

	validInstance := map[string]any{
		"int_val":   1,
		"float_val": 1.0,
		"bool_val":  false,
	}
	err = schema.validateRequired(validInstance)
	assert.NoError(t, err)
	err = schema.ValidateInstance(validInstance)
	assert.NoError(t, err)

	invalidInstance := map[string]any{
		"string_val": "abc",
		"float_val":  1.0,
		"bool_val":   false,
	}
	err = schema.validateRequired(invalidInstance)
	assert.EqualError(t, err, "no value provided for required property int_val")
	err = schema.ValidateInstance(invalidInstance)
	assert.EqualError(t, err, "no value provided for required property int_val")
}
