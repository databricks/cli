package jsonschema

import (
	"math"
	"testing"

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
	assert.Equal(t, int64(2), v)

	v, err = toInteger(float64(4))
	assert.NoError(t, err)
	assert.Equal(t, int64(4), v)

	v, err = toInteger(float64(4))
	assert.NoError(t, err)
	assert.Equal(t, int64(4), v)

	v, err = toInteger(float64(math.MaxInt32 + 10))
	assert.NoError(t, err)
	assert.Equal(t, int64(2147483657), v)

	v, err = toInteger(2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), v)

	_, err = toInteger(float32(2.2))
	assert.EqualError(t, err, "expected integer value, got: 2.2")

	_, err = toInteger(float64(math.MaxInt32 + 100.1))
	assert.ErrorContains(t, err, "expected integer value, got: 2.1474837471e+09")

	_, err = toInteger("abcd")
	assert.EqualError(t, err, "cannot convert \"abcd\" to an integer")
}

func TestTemplateToString(t *testing.T) {
	s, err := ToString(true, BooleanType)
	assert.NoError(t, err)
	assert.Equal(t, "true", s)

	s, err = ToString("abc", StringType)
	assert.NoError(t, err)
	assert.Equal(t, "abc", s)

	s, err = ToString(1.1, NumberType)
	assert.NoError(t, err)
	assert.Equal(t, "1.1", s)

	s, err = ToString(2, IntegerType)
	assert.NoError(t, err)
	assert.Equal(t, "2", s)

	_, err = ToString([]string{}, ArrayType)
	assert.EqualError(t, err, "cannot format object of type array as a string. Value of object: []string{}")

	_, err = ToString("true", BooleanType)
	assert.EqualError(t, err, "expected bool, got: \"true\"")

	_, err = ToString(123, StringType)
	assert.EqualError(t, err, "expected string, got: 123")

	_, err = ToString(false, NumberType)
	assert.EqualError(t, err, "expected float, got: false")

	_, err = ToString("abc", IntegerType)
	assert.EqualError(t, err, "cannot convert \"abc\" to an integer")

	_, err = ToString("abc", "foobar")
	assert.EqualError(t, err, "unknown json schema type: \"foobar\"")
}

func TestTemplateFromString(t *testing.T) {
	v, err := FromString("true", BooleanType)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = FromString("abc", StringType)
	assert.NoError(t, err)
	assert.Equal(t, "abc", v)

	v, err = FromString("1.1", NumberType)
	assert.NoError(t, err)
	// Floating point conversions are not perfect
	assert.True(t, (v.(float64)-1.1) < 0.000001)

	v, err = FromString("12345", IntegerType)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), v)

	v, err = FromString("123", NumberType)
	assert.NoError(t, err)
	assert.Equal(t, float64(123), v)

	_, err = FromString("qrt", ArrayType)
	assert.EqualError(t, err, "cannot parse string as object of type array. Value of string: \"qrt\"")

	_, err = FromString("abc", IntegerType)
	assert.EqualError(t, err, "could not parse \"abc\" as a integer: strconv.ParseInt: parsing \"abc\": invalid syntax")

	_, err = FromString("1.0", IntegerType)
	assert.EqualError(t, err, "could not parse \"1.0\" as a integer: strconv.ParseInt: parsing \"1.0\": invalid syntax")

	_, err = FromString("1.0", "foobar")
	assert.EqualError(t, err, "unknown json schema type: \"foobar\"")
}
