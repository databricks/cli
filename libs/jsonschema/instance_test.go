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

func TestValidateInstanceForMultiplePatterns(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/multiple-patterns-schema.json")
	require.NoError(t, err)

	// Valid values for both foo and bar
	validInstance := map[string]any{
		"foo": "abcc",
		"bar": "deff",
	}
	assert.NoError(t, schema.validatePattern(validInstance))
	assert.NoError(t, schema.ValidateInstance(validInstance))

	// Valid value for bar, invalid value for foo
	invalidInstanceValue := map[string]any{
		"foo": "xyz",
		"bar": "deff",
	}
	assert.EqualError(t, schema.validatePattern(invalidInstanceValue), "invalid value for foo: \"xyz\". Expected to match regex pattern: ^[a-c]+$")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "invalid value for foo: \"xyz\". Expected to match regex pattern: ^[a-c]+$")

	// Valid value for foo, invalid value for bar
	invalidInstanceValue = map[string]any{
		"foo": "abcc",
		"bar": "xyz",
	}
	assert.EqualError(t, schema.validatePattern(invalidInstanceValue), "invalid value for bar: \"xyz\". Expected to match regex pattern: ^[d-f]+$")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "invalid value for bar: \"xyz\". Expected to match regex pattern: ^[d-f]+$")
}

func TestValidateInstanceForConst(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema-const.json")
	require.NoError(t, err)

	// Valid values for both foo and bar
	validInstance := map[string]any{
		"foo": "abc",
		"bar": "def",
	}
	assert.NoError(t, schema.validateConst(validInstance))
	assert.NoError(t, schema.ValidateInstance(validInstance))

	// Empty instance
	emptyInstanceValue := map[string]any{}
	assert.NoError(t, schema.validateConst(emptyInstanceValue))
	assert.NoError(t, schema.ValidateInstance(emptyInstanceValue))

	// Missing value for bar
	missingInstanceValue := map[string]any{
		"foo": "abc",
	}
	assert.NoError(t, schema.validateConst(missingInstanceValue))
	assert.NoError(t, schema.ValidateInstance(missingInstanceValue))

	// Valid value for bar, invalid value for foo
	invalidInstanceValue := map[string]any{
		"foo": "xyz",
		"bar": "def",
	}
	assert.EqualError(t, schema.validateConst(invalidInstanceValue), "expected value of property foo to be abc. Found: xyz")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "expected value of property foo to be abc. Found: xyz")

	// Valid value for foo, invalid value for bar
	invalidInstanceValue = map[string]any{
		"foo": "abc",
		"bar": "xyz",
	}
	assert.EqualError(t, schema.validateConst(invalidInstanceValue), "expected value of property bar to be def. Found: xyz")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "expected value of property bar to be def. Found: xyz")
}

func TestValidateInstanceForEmptySchema(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-empty-anyof.json")
	require.NoError(t, err)

	// Valid values for both foo and bar
	validInstance := map[string]any{
		"foo": "abc",
		"bar": "abc",
	}
	assert.ErrorContains(t, schema.validateAnyOf(validInstance), "anyOf must contain at least one schema")
	assert.ErrorContains(t, schema.ValidateInstance(validInstance), "anyOf must contain at least one schema")
}

func TestValidateInstanceForAnyOf(t *testing.T) {
	schema, err := Load("./testdata/instance-validate/test-schema-anyof.json")
	require.NoError(t, err)

	// Valid values for both foo and bar
	validInstance := map[string]any{
		"foo": "abc",
		"bar": "abc",
	}
	assert.NoError(t, schema.validateAnyOf(validInstance))
	assert.NoError(t, schema.ValidateInstance(validInstance))

	// Valid values for bar
	validInstance = map[string]any{
		"foo": "abc",
		"bar": "def",
	}
	assert.NoError(t, schema.validateAnyOf(validInstance))
	assert.NoError(t, schema.ValidateInstance(validInstance))

	// Empty instance. Invalid because "foo" is required.
	emptyInstanceValue := map[string]any{}
	assert.ErrorContains(t, schema.validateAnyOf(emptyInstanceValue), "instance does not match any of the schemas in anyOf")
	assert.ErrorContains(t, schema.ValidateInstance(emptyInstanceValue), "instance does not match any of the schemas in anyOf")

	// Missing values for bar, invalid value for foo. Passes because only "foo"
	// is required in second condition.
	missingInstanceValue := map[string]any{
		"foo": "xyz",
	}
	assert.NoError(t, schema.validateAnyOf(missingInstanceValue))
	assert.NoError(t, schema.ValidateInstance(missingInstanceValue))

	// Valid value for bar, invalid value for foo
	invalidInstanceValue := map[string]any{
		"foo": "xyz",
		"bar": "abc",
	}
	assert.EqualError(t, schema.validateAnyOf(invalidInstanceValue), "instance does not match any of the schemas in anyOf")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "instance does not match any of the schemas in anyOf")

	// Invalid value for both
	invalidInstanceValue = map[string]any{
		"bar": "xyz",
	}
	assert.EqualError(t, schema.validateAnyOf(invalidInstanceValue), "instance does not match any of the schemas in anyOf")
	assert.EqualError(t, schema.ValidateInstance(invalidInstanceValue), "instance does not match any of the schemas in anyOf")
}
