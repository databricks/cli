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
	s, err := toString(true, BooleanType)
	assert.NoError(t, err)
	assert.Equal(t, "true", s)

	s, err = toString("abc", StringType)
	assert.NoError(t, err)
	assert.Equal(t, "abc", s)

	s, err = toString(1.1, NumberType)
	assert.NoError(t, err)
	assert.Equal(t, "1.1", s)

	s, err = toString(2, IntegerType)
	assert.NoError(t, err)
	assert.Equal(t, "2", s)

	_, err = toString([]string{}, ArrayType)
	assert.EqualError(t, err, "cannot format object of type array as a string. Value of object: []string{}")

	_, err = toString("true", BooleanType)
	assert.EqualError(t, err, "expected bool, got: \"true\"")

	_, err = toString(123, StringType)
	assert.EqualError(t, err, "expected string, got: 123")

	_, err = toString(false, NumberType)
	assert.EqualError(t, err, "expected float, got: false")

	_, err = toString("abc", IntegerType)
	assert.EqualError(t, err, "cannot convert \"abc\" to an integer")

	_, err = toString("abc", "foobar")
	assert.EqualError(t, err, "unknown json schema type: \"foobar\"")
}

func TestTemplateFromString(t *testing.T) {
	v, err := fromString("true", BooleanType)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = fromString("abc", StringType)
	assert.NoError(t, err)
	assert.Equal(t, "abc", v)

	v, err = fromString("1.1", NumberType)
	assert.NoError(t, err)
	// Floating point conversions are not perfect
	assert.Less(t, (v.(float64) - 1.1), 0.000001)

	v, err = fromString("12345", IntegerType)
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), v)

	v, err = fromString("123", NumberType)
	assert.NoError(t, err)
	assert.InDelta(t, float64(123), v.(float64), 0.0001)

	_, err = fromString("qrt", ArrayType)
	assert.EqualError(t, err, "cannot parse string as object of type array. Value of string: \"qrt\"")

	_, err = fromString("abc", IntegerType)
	assert.EqualError(t, err, "\"abc\" is not a integer")

	_, err = fromString("1.0", IntegerType)
	assert.EqualError(t, err, "\"1.0\" is not a integer")

	_, err = fromString("1.0", "foobar")
	assert.EqualError(t, err, "unknown json schema type: \"foobar\"")
}

func TestTemplateToStringSlice(t *testing.T) {
	s, err := toStringSlice([]any{"a", "b", "c"}, StringType)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, s)

	s, err = toStringSlice([]any{1.1, 2.2, 3.3}, NumberType)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1.1", "2.2", "3.3"}, s)
}

func TestValidatePropertyPatternMatch(t *testing.T) {
	var err error

	// Expect no error if no pattern is specified.
	err = validatePatternMatch("foo", 1, &Schema{Type: "integer"})
	assert.NoError(t, err)

	// Expect error because value is not a string.
	err = validatePatternMatch("bar", 1, &Schema{Type: "integer", Pattern: "abc"})
	assert.EqualError(t, err, "invalid value for bar: 1. Expected a value of type string")

	// Expect error because the pattern is invalid.
	err = validatePatternMatch("bar", "xyz", &Schema{Type: "string", Pattern: "(abc"})
	assert.EqualError(t, err, "error parsing regexp: missing closing ): `(abc`")

	// Expect no error because the pattern matches.
	err = validatePatternMatch("bar", "axyzd", &Schema{Type: "string", Pattern: "(a*.d)"})
	assert.NoError(t, err)

	// Expect custom error message on match fail
	err = validatePatternMatch("bar", "axyze", &Schema{
		Type:    "string",
		Pattern: "(a*.d)",
		Extension: Extension{
			PatternMatchFailureMessage: "my custom msg",
		},
	})
	assert.EqualError(t, err, "invalid value for bar: \"axyze\". my custom msg")

	// Expect generic message on match fail
	err = validatePatternMatch("bar", "axyze", &Schema{
		Type:    "string",
		Pattern: "(a*.d)",
	})
	assert.EqualError(t, err, "invalid value for bar: \"axyze\". Expected to match regex pattern: (a*.d)")
}
