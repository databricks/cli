package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
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
