package template

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

func TestValidatorFloat(t *testing.T) {
	err := validateFloat(true)
	assert.ErrorContains(t, err, "expected type float, but value is true")

	err = validateFloat(int32(1))
	assert.ErrorContains(t, err, "expected type float, but value is 1")

	err = validateFloat(int64(1))
	assert.ErrorContains(t, err, "expected type float, but value is 1")

	err = validateFloat(float32(1))
	assert.NoError(t, err)

	err = validateFloat(float64(1))
	assert.NoError(t, err)

	err = validateFloat("abc")
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
