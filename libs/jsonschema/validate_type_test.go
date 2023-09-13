package jsonschema

import (
	"testing"

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
	err := validateType(int(0), IntegerType)
	assert.NoError(t, err)
	err = validateType(int32(1), IntegerType)
	assert.NoError(t, err)
	err = validateType(int64(1), IntegerType)
	assert.NoError(t, err)

	err = validateType(float32(1.1), NumberType)
	assert.NoError(t, err)
	err = validateType(float64(1.2), NumberType)
	assert.NoError(t, err)

	err = validateType(false, BooleanType)
	assert.NoError(t, err)

	err = validateType("abc", StringType)
	assert.NoError(t, err)

	// assert validation failing for integers
	err = validateType(float64(1.2), IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is 1.2")
	err = validateType(true, IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is true")
	err = validateType("abc", IntegerType)
	assert.ErrorContains(t, err, "expected type integer, but value is \"abc\"")

	// assert validation failing for floats
	err = validateType(true, NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is true")
	err = validateType("abc", NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is \"abc\"")
	err = validateType(int(1), NumberType)
	assert.ErrorContains(t, err, "expected type float, but value is 1")

	// assert validation failing for boolean
	err = validateType(int(1), BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType(float64(1), BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is 1")
	err = validateType("abc", BooleanType)
	assert.ErrorContains(t, err, "expected type boolean, but value is \"abc\"")

	// assert validation failing for string
	err = validateType(int(1), StringType)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(float64(1), StringType)
	assert.ErrorContains(t, err, "expected type string, but value is 1")
	err = validateType(false, StringType)
	assert.ErrorContains(t, err, "expected type string, but value is false")
}
