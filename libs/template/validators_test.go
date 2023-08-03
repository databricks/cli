package template

import (
	"testing"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestValidatorString(t *testing.T) {
	err := validateString("abc")
	assert.NoError(t, err)

	err = validateString(1)
	assert.ErrorContains(t, err, "expected type string, but value is 1")

	err = validateString(true)
	assert.ErrorContains(t, err, "expected type string, but value is true")

	err = validateString("false")
	assert.NoError(t, err)
}

func TestValidatorBoolean(t *testing.T) {
	err := validateBoolean(true)
	assert.NoError(t, err)

	err = validateBoolean(1)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")

	err = validateBoolean("abc")
	assert.ErrorContains(t, err, "expected type boolean, but value is \"abc\"")

	err = validateBoolean("false")
	assert.ErrorContains(t, err, "expected type boolean, but value is \"false\"")
}

func TestValidatorNumber(t *testing.T) {
	err := validateNumber(true)
	assert.ErrorContains(t, err, "expected type float, but value is true")

	err = validateNumber(int32(1))
	assert.ErrorContains(t, err, "expected type float, but value is 1")

	err = validateNumber(int64(2))
	assert.ErrorContains(t, err, "expected type float, but value is 2")

	err = validateNumber(float32(1))
	assert.NoError(t, err)

	err = validateNumber(float64(1))
	assert.NoError(t, err)

	err = validateNumber("abc")
	assert.ErrorContains(t, err, "expected type float, but value is \"abc\"")
}

func TestValidatorInt(t *testing.T) {
	err := validateInteger(true)
	assert.ErrorContains(t, err, "expected type integer, but value is true")

	err = validateInteger(int32(1))
	assert.NoError(t, err)

	err = validateInteger(int64(1))
	assert.NoError(t, err)

	err = validateInteger(float32(1))
	assert.ErrorContains(t, err, "expected type integer, but value is 1")

	err = validateInteger(float64(1))
	assert.ErrorContains(t, err, "expected type integer, but value is 1")

	err = validateInteger("abc")
	assert.ErrorContains(t, err, "expected type integer, but value is \"abc\"")
}

func TestTemplateValidateType(t *testing.T) {
	// assert validation passing
	err := validateType(int(0), jsonschema.IntegerType)
	assert.NoError(t, err)
	err = validateType(int32(1), jsonschema.IntegerType)
	assert.NoError(t, err)
	err = validateType(int64(1), jsonschema.IntegerType)
	assert.NoError(t, err)

	err = validateType(float32(1.1), jsonschema.NumberType)
	assert.NoError(t, err)
	err = validateType(float64(1.2), jsonschema.NumberType)
	assert.NoError(t, err)


	err = validateType(false, jsonschema.BooleanType)
	assert.NoError(t, err)

	err = validateType("abc", jsonschema.StringType)
	assert.NoError(t, err)

	// assert validation failing for integers
	err = validateType(float64(1.2), jsonschema.IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is 1.2")
	err = validateType(true, jsonschema.IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is true")
	err = validateType("abc", jsonschema.IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is \"abc\"")

	// assert validation failing for floats
	err = validateType(true, jsonschema.NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is true")
	err = validateType("abc", jsonschema.NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is \"abc\"")
	err = validateType(int(1), jsonschema.NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is 1")

	// assert validation failing for boolean
	err = validateType(int(1), jsonschema.BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType(float64(1), jsonschema.BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType("abc", jsonschema.BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is \"abc\"")

	// assert validation failing for string
	err = validateType(int(1), jsonschema.StringType)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(float64(1), jsonschema.StringType)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(false, jsonschema.StringType)
	assert.ErrorContains(t, err, "expected type string, but value is false")
}
