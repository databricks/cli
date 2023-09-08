package jsonschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaValidateTypeNames(t *testing.T) {
	var err error
	toSchema := func(s string) *Schema {
		return &Schema{
			Properties: map[string]*Schema{
				"foo": {
					Type: Type(s),
				},
			},
		}
	}

	err = toSchema("string").validate()
	assert.NoError(t, err)

	err = toSchema("boolean").validate()
	assert.NoError(t, err)

	err = toSchema("number").validate()
	assert.NoError(t, err)

	err = toSchema("integer").validate()
	assert.NoError(t, err)

	err = toSchema("int").validate()
	assert.EqualError(t, err, "type int is not a recognized json schema type. Please use \"integer\" instead")

	err = toSchema("float").validate()
	assert.EqualError(t, err, "type float is not a recognized json schema type. Please use \"number\" instead")

	err = toSchema("bool").validate()
	assert.EqualError(t, err, "type bool is not a recognized json schema type. Please use \"boolean\" instead")

	err = toSchema("foobar").validate()
	assert.EqualError(t, err, "type foobar is not a recognized json schema type")
}

func TestSchemaLoadIntegers(t *testing.T) {
	schema, err := Load("./testdata/schema-load-int/schema-valid.json")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), schema.Properties["abc"].Default)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, schema.Properties["abc"].Enum)
}

func TestSchemaLoadIntegersWithInvalidDefault(t *testing.T) {
	_, err := Load("./testdata/schema-load-int/schema-invalid-default.json")
	assert.EqualError(t, err, "failed to parse default value for property abc: expected integer value, got: 1.1")
}

func TestSchemaLoadIntegersWithInvalidEnums(t *testing.T) {
	_, err := Load("./testdata/schema-load-int/schema-invalid-enum.json")
	assert.EqualError(t, err, "failed to parse enum value 2.4 at index 1 for property abc: expected integer value, got: 2.4")
}

func TestSchemaValidateDefaultType(t *testing.T) {
	invalidSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "number",
				Default: "abc",
			},
		},
	}

	err := invalidSchema.validate()
	assert.EqualError(t, err, "type validation for default value of property foo failed: expected type float, but value is \"abc\"")

	validSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "boolean",
				Default: true,
			},
		},
	}

	err = validSchema.validate()
	assert.NoError(t, err)
}

func TestSchemaValidateEnumType(t *testing.T) {
	invalidSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type: "boolean",
				Enum: []any{true, "false"},
			},
		},
	}

	err := invalidSchema.validate()
	assert.EqualError(t, err, "type validation for enum at index 1 failed for property foo: expected type boolean, but value is \"false\"")

	validSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type: "boolean",
				Enum: []any{true, false},
			},
		},
	}

	err = validSchema.validate()
	assert.NoError(t, err)
}

func TestSchemaValidateErrorWhenDefaultValueIsNotInEnums(t *testing.T) {
	invalidSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Default: "abc",
				Enum:    []any{"def", "ghi"},
			},
		},
	}

	err := invalidSchema.validate()
	assert.EqualError(t, err, "list of enum values for property foo does not contain default value abc: [def ghi]")

	validSchema := &Schema{
		Properties: map[string]*Schema{
			"foo": {
				Type:    "string",
				Default: "abc",
				Enum:    []any{"def", "ghi", "abc"},
			},
		},
	}

	err = validSchema.validate()
	assert.NoError(t, err)
}
