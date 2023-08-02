package template

import (
	"testing"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestTemplateIsInteger(t *testing.T) {
	assert.False(t, isIntegerValue(1.1))
	assert.False(t, isIntegerValue(0.1))
	assert.False(t, isIntegerValue(-0.1))

	assert.True(t, isIntegerValue(-1.0))
	assert.True(t, isIntegerValue(0.0))
	assert.True(t, isIntegerValue(2.0))
}

func TestTemplateToInteger(t *testing.T) {
	v, err := toInteger(float32(2))
	assert.NoError(t, err)
	assert.Equal(t, int(2), v)

	v, err = toInteger(float64(4))
	assert.NoError(t, err)
	assert.Equal(t, int(4), v)

	v, err = toInteger(2)
	assert.NoError(t, err)
	assert.Equal(t, int(2), v)

	_, err = toInteger(float32(2.2))
	assert.EqualError(t, err, "expected integer value, got: 2.2")

	_, err = toInteger("abcd")
	assert.EqualError(t, err, "cannot convert \"abcd\" to an integer")
}

func TestTemplateToString(t *testing.T) {
	s, err := toString(true, jsonschema.BooleanType)
	assert.NoError(t, err)
	assert.Equal(t, "true", s)

	s, err = toString("abc", jsonschema.StringType)
	assert.NoError(t, err)
	assert.Equal(t, "abc", s)

	s, err = toString(1.1, jsonschema.NumberType)
	assert.NoError(t, err)
	assert.Equal(t, "1.1", s)

	s, err = toString(2, jsonschema.IntegerType)
	assert.NoError(t, err)
	assert.Equal(t, "2", s)

	_, err = toString([]string{}, jsonschema.ArrayType)
	assert.EqualError(t, err, "cannot format object of type array as a string. Value of object: []string{}")

	_, err = toString("true", jsonschema.BooleanType)
	assert.EqualError(t, err, "expected bool, got: \"true\"")

	_, err = toString(123, jsonschema.StringType)
	assert.EqualError(t, err, "expected string, got: 123")

	_, err = toString(false, jsonschema.NumberType)
	assert.EqualError(t, err, "expected float, got: false")

	_, err = toString("abc", jsonschema.IntegerType)
	assert.EqualError(t, err, "cannot convert \"abc\" to an integer")
}

func TestTemplateFromString(t *testing.T) {
	v, err := fromString("true", jsonschema.BooleanType)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = fromString("abc", jsonschema.StringType)
	assert.NoError(t, err)
	assert.Equal(t, "abc", v)

	v, err = fromString("1.1", jsonschema.NumberType)
	assert.NoError(t, err)
	// Floating point conversions are not perfect
	assert.True(t, (v.(float64)-1.1) < 0.000001)

	v, err = fromString("12345", jsonschema.IntegerType)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), v)

	_, err = fromString("qrt", jsonschema.ArrayType)
	assert.EqualError(t, err, "cannot parse string as object of type array. Value of string: \"qrt\"")

	_, err = fromString("abc", jsonschema.IntegerType)
	assert.EqualError(t, err, "could not parse \"abc\" as a integer: strconv.ParseInt: parsing \"abc\": invalid syntax")
}
