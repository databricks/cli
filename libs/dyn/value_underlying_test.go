package dyn_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestValueUnderlyingMap(t *testing.T) {
	v := dyn.V(
		map[string]dyn.Value{
			"key": dyn.NewValue("value", []dyn.Location{{File: "file", Line: 1, Column: 2}}),
		},
	)

	vv1, ok := v.AsMap()
	assert.True(t, ok)

	_, ok = dyn.InvalidValue.AsMap()
	assert.False(t, ok)

	vv2 := v.MustMap()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind map, got invalid", func() {
		dyn.InvalidValue.MustMap()
	})
}

func TestValueUnderlyingSequence(t *testing.T) {
	v := dyn.V(
		[]dyn.Value{
			dyn.NewValue("value", []dyn.Location{{File: "file", Line: 1, Column: 2}}),
		},
	)

	vv1, ok := v.AsSequence()
	assert.True(t, ok)

	_, ok = dyn.InvalidValue.AsSequence()
	assert.False(t, ok)

	vv2 := v.MustSequence()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind sequence, got invalid", func() {
		dyn.InvalidValue.MustSequence()
	})
}

func TestValueUnderlyingString(t *testing.T) {
	v := dyn.V("value")

	vv1, ok := v.AsString()
	assert.True(t, ok)

	_, ok = dyn.InvalidValue.AsString()
	assert.False(t, ok)

	vv2 := v.MustString()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind string, got invalid", func() {
		dyn.InvalidValue.MustString()
	})
}

func TestValueUnderlyingBool(t *testing.T) {
	v := dyn.V(true)

	vv1, ok := v.AsBool()
	assert.True(t, ok)

	_, ok = dyn.InvalidValue.AsBool()
	assert.False(t, ok)

	vv2 := v.MustBool()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind bool, got invalid", func() {
		dyn.InvalidValue.MustBool()
	})
}

func TestValueUnderlyingInt(t *testing.T) {
	v := dyn.V(int(1))

	vv1, ok := v.AsInt()
	assert.True(t, ok)

	_, ok = dyn.InvalidValue.AsInt()
	assert.False(t, ok)

	vv2 := v.MustInt()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind int, got invalid", func() {
		dyn.InvalidValue.MustInt()
	})

	// Test int32 type specifically.
	v = dyn.V(int32(1))
	vv1, ok = v.AsInt()
	assert.True(t, ok)
	assert.Equal(t, int64(1), vv1)

	// Test int64 type specifically.
	v = dyn.V(int64(1))
	vv1, ok = v.AsInt()
	assert.True(t, ok)
	assert.Equal(t, int64(1), vv1)
}

func TestValueUnderlyingFloat(t *testing.T) {
	v := dyn.V(float32(1.0))

	vv1, ok := v.AsFloat()
	assert.True(t, ok)

	_, ok = dyn.InvalidValue.AsFloat()
	assert.False(t, ok)

	vv2 := v.MustFloat()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind float, got invalid", func() {
		dyn.InvalidValue.MustFloat()
	})

	// Test float64 type specifically.
	v = dyn.V(float64(1.0))
	vv1, ok = v.AsFloat()
	assert.True(t, ok)
	assert.Equal(t, float64(1.0), vv1)
}

func TestValueUnderlyingTime(t *testing.T) {
	v := dyn.V(dyn.FromTime(time.Now()))

	vv1, ok := v.AsTime()
	assert.True(t, ok)

	_, ok = dyn.InvalidValue.AsTime()
	assert.False(t, ok)

	vv2 := v.MustTime()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind time, got invalid", func() {
		dyn.InvalidValue.MustTime()
	})
}
