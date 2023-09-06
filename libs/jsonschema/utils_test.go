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
