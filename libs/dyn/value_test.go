package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestInvalidValue(t *testing.T) {
	// Assert that the zero value of [dyn.Value] is the invalid value.
	var zero dyn.Value
	assert.Equal(t, zero, dyn.InvalidValue)
}

func TestValueIsAnchor(t *testing.T) {
	var zero dyn.Value
	assert.False(t, zero.IsAnchor())
	mark := zero.MarkAnchor()
	assert.True(t, mark.IsAnchor())
}

func TestValueAsMap(t *testing.T) {
	var zeroValue dyn.Value
	_, ok := zeroValue.AsMap()
	assert.False(t, ok)

	intValue := dyn.V(1)
	_, ok = intValue.AsMap()
	assert.False(t, ok)

	mapValue := dyn.NewValue(
		map[string]dyn.Value{
			"key": dyn.NewValue(
				"value",
				[]dyn.Location{{File: "file", Line: 1, Column: 2}}),
		},
		[]dyn.Location{{File: "file", Line: 1, Column: 2}},
	)

	m, ok := mapValue.AsMap()
	assert.True(t, ok)
	assert.Equal(t, 1, m.Len())
}

func TestValueIsValid(t *testing.T) {
	var zeroValue dyn.Value
	assert.False(t, zeroValue.IsValid())
	intValue := dyn.V(1)
	assert.True(t, intValue.IsValid())
}

func TestIsZero(t *testing.T) {
	assert.True(t, dyn.V(int(0)).IsZero(), "int")
	assert.False(t, dyn.V(int(1)).IsZero(), "int")
	// assert.True(t, dyn.V(uint(0)).IsZero(), "uint") // panics

	// assert.True(t, dyn.V(int8(0)).IsZero(), "int8") // panics
	// assert.True(t, dyn.V(uint8(0)).IsZero(), "uint8") // panics

	// assert.True(t, dyn.V(int16(0)).IsZero(), "int16") // panics
	// assert.True(t, dyn.V(uint16(0)).IsZero(), "uint16") // panics

	assert.True(t, dyn.V(int32(0)).IsZero(), "int32")
	assert.False(t, dyn.V(int32(1)).IsZero(), "int32")
	// assert.True(t, dyn.V(uint32(0)).IsZero(), "uint32") // panics

	assert.True(t, dyn.V(int64(0)).IsZero(), "int64")
	assert.False(t, dyn.V(int64(-1)).IsZero(), "int64")

	// assert.True(t, dyn.V(uint64(0)).IsZero(), "uint64") // panics
	// assert.False(t, dyn.V(uint64(2)).IsZero(), "uint64") // panics

	assert.True(t, dyn.V("").IsZero(), "string")
	assert.False(t, dyn.V("x").IsZero(), "string")

	assert.True(t, dyn.V(false).IsZero(), "bool")
	assert.False(t, dyn.V(true).IsZero(), "bool")

	assert.True(t, dyn.V(float32(0.0)).IsZero(), "float32")
	assert.False(t, dyn.V(float32(0.01)).IsZero(), "float32")

	assert.True(t, dyn.V(float64(0.0)).IsZero(), "float64")
	assert.False(t, dyn.V(float64(0.01)).IsZero(), "float64")
}
