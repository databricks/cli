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

func TestLoadInstance(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema.json")
	require.NoError(t, err)

	// Expect the instance to be loaded successfully.
	instance, err := schema.LoadInstance("./testdata/instance-load/valid-instance.json")
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"bool_val":   false,
		"int_val":    int64(1),
		"string_val": "abc",
		"float_val":  2.0,
	}, instance)

	// Expect instance validation against the schema to fail.
	_, err = schema.LoadInstance("./testdata/instance-load/invalid-type-instance.json")
	assert.EqualError(t, err, "incorrect type for property string_val: expected type string, but value is 123")
}

func TestValidateInstanceEnum(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema-enum.json")
	require.NoError(t, err)

	validInstance := map[string]any{
		"foo": "b",
		"bar": int64(6),
	}
	assert.NoError(t, schema.validateEnum(validInstance))
	assert.NoError(t, schema.ValidateInstance(validInstance))

	invalidStringInstance := map[string]any{
		"foo": "d",
		"bar": int64(2),
	}
	assert.EqualError(t, schema.validateEnum(invalidStringInstance), "expected value of property foo to be one of [a b c]. Found: d")
	assert.EqualError(t, schema.ValidateInstance(invalidStringInstance), "expected value of property foo to be one of [a b c]. Found: d")

	invalidIntInstance := map[string]any{
		"foo": "a",
		"bar": int64(1),
	}
	assert.EqualError(t, schema.validateEnum(invalidIntInstance), "expected value of property bar to be one of [2 4 6]. Found: 1")
	assert.EqualError(t, schema.ValidateInstance(invalidIntInstance), "expected value of property bar to be one of [2 4 6]. Found: 1")
}

func TestValidateInstancePattern(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema-pattern.json")
	require.NoError(t, err)

	validInstance := map[string]any{
		"foo": "axyzc",
	}
	assert.NoError(t, schema.validatePattern(validInstance))
	assert.NoError(t, schema.ValidateInstance(validInstance))

	invalidInstanceValue := map[string]any{
		"foo": "xyz",
	}
	assert.EqualError(t, schema.validatePattern(invalidInstanceValue), "invalid value for foo: \"xyz\". Expected to match regex pattern: a.*c")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "invalid value for foo: \"xyz\". Expected to match regex pattern: a.*c")

	invalidInstanceType := map[string]any{
		"foo": 1,
	}
	assert.EqualError(t, schema.validatePattern(invalidInstanceType), "invalid value for foo: 1. Expected a value of type string")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceType), "incorrect type for property foo: expected type string, but value is 1")
}

func TestValidateInstancePatternWithCustomMessage(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema-pattern-with-custom-message.json")
	require.NoError(t, err)

	validInstance := map[string]any{
		"foo": "axyzc",
	}
	assert.NoError(t, schema.validatePattern(validInstance))
	assert.NoError(t, schema.ValidateInstance(validInstance))

	invalidInstanceValue := map[string]any{
		"foo": "xyz",
	}
	assert.EqualError(t, schema.validatePattern(invalidInstanceValue), "invalid value for foo: \"xyz\". Please enter a string starting with 'a' and ending with 'c'")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "invalid value for foo: \"xyz\". Please enter a string starting with 'a' and ending with 'c'")
}
