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

func TestMarkSensitive(t *testing.T) {
	v := dyn.V("secret-value")
	assert.False(t, v.IsSensitive())

	sv := v.MarkSensitive()
	assert.True(t, sv.IsSensitive())

	// AsAny returns the redaction placeholder for sensitive strings.
	assert.Equal(t, "********", sv.AsAny())

	// MustString returns the real underlying value.
	assert.Equal(t, "secret-value", sv.MustString())
}

func TestSensitivePreservedByWithLocations(t *testing.T) {
	locs := []dyn.Location{{File: "file", Line: 1, Column: 1}}
	v := dyn.V("secret").MarkSensitive()
	v2 := v.WithLocations(locs)
	assert.True(t, v2.IsSensitive())
	assert.Equal(t, locs, v2.Locations())
}

func TestSensitivePreservedByWithSensitive(t *testing.T) {
	v := dyn.V("secret").MarkSensitive()
	assert.True(t, v.IsSensitive())

	v2 := v.WithSensitive(false)
	assert.False(t, v2.IsSensitive())

	v3 := v2.WithSensitive(true)
	assert.True(t, v3.IsSensitive())
}

func TestIsZero(t *testing.T) {
	assert.True(t, dyn.V(0).IsZero(), "int")
	assert.True(t, dyn.V(int(0)).IsZero(), "int")
	assert.False(t, dyn.V(int(1)).IsZero(), "int")
	assert.True(t, dyn.V(uint(0)).IsZero(), "uint")

	assert.True(t, dyn.V(int8(0)).IsZero(), "int8")
	assert.True(t, dyn.V(uint8(0)).IsZero(), "uint8")

	assert.True(t, dyn.V(int16(0)).IsZero(), "int16")
	assert.True(t, dyn.V(uint16(0)).IsZero(), "uint16")

	assert.True(t, dyn.V(int32(0)).IsZero(), "int32")
	assert.False(t, dyn.V(int32(1)).IsZero(), "int32")
	assert.True(t, dyn.V(uint32(0)).IsZero(), "uint32")

	assert.True(t, dyn.V(int64(0)).IsZero(), "int64")
	assert.False(t, dyn.V(int64(-1)).IsZero(), "int64")

	assert.True(t, dyn.V(uint64(0)).IsZero(), "uint64")
	assert.False(t, dyn.V(uint64(2)).IsZero(), "uint64")

	assert.True(t, dyn.V("").IsZero(), "string")
	assert.False(t, dyn.V("x").IsZero(), "string")

	assert.True(t, dyn.V(false).IsZero(), "bool")
	assert.False(t, dyn.V(true).IsZero(), "bool")

	assert.True(t, dyn.V(float32(0.0)).IsZero(), "float32")
	assert.False(t, dyn.V(float32(0.01)).IsZero(), "float32")

	assert.True(t, dyn.V(float64(0.0)).IsZero(), "float64")
	assert.False(t, dyn.V(float64(0.01)).IsZero(), "float64")

	assert.True(t, dyn.V(dyn.Time{}).IsZero(), "time")
	assert.True(t, dyn.V(dyn.Mapping{}).IsZero(), "Mapping")
	assert.True(t, dyn.V([]dyn.Value{}).IsZero(), "Sequence")
	assert.False(t, dyn.V([]dyn.Value{dyn.V(0)}).IsZero(), "Sequence")
}
