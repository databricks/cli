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
			"key": dyn.NewValue("value", dyn.Location{File: "file", Line: 1, Column: 2}),
		},
	)

	vv1, ok := v.AsMapping()
	assert.True(t, ok)

	_, ok = dyn.NilValue.AsMapping()
	assert.False(t, ok)

	vv2 := v.MustMapping()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind map, got nil", func() {
		dyn.NilValue.MustMapping()
	})
}

func TestValueUnderlyingSequence(t *testing.T) {
	v := dyn.V(
		[]dyn.Value{
			dyn.NewValue("value", dyn.Location{File: "file", Line: 1, Column: 2}),
		},
	)

	vv1, ok := v.AsSequence()
	assert.True(t, ok)

	_, ok = dyn.NilValue.AsSequence()
	assert.False(t, ok)

	vv2 := v.MustSequence()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind sequence, got nil", func() {
		dyn.NilValue.MustSequence()
	})
}

func TestValueUnderlyingString(t *testing.T) {
	v := dyn.V("value")

	vv1, ok := v.AsString()
	assert.True(t, ok)

	_, ok = dyn.NilValue.AsString()
	assert.False(t, ok)

	vv2 := v.MustString()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind string, got nil", func() {
		dyn.NilValue.MustString()
	})
}

func TestValueUnderlyingBool(t *testing.T) {
	v := dyn.V(true)

	vv1, ok := v.AsBool()
	assert.True(t, ok)

	_, ok = dyn.NilValue.AsBool()
	assert.False(t, ok)

	vv2 := v.MustBool()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind bool, got nil", func() {
		dyn.NilValue.MustBool()
	})
}

func TestValueUnderlyingInt(t *testing.T) {
	v := dyn.V(int(1))

	vv1, ok := v.AsInt()
	assert.True(t, ok)

	_, ok = dyn.NilValue.AsInt()
	assert.False(t, ok)

	vv2 := v.MustInt()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind int, got nil", func() {
		dyn.NilValue.MustInt()
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

	_, ok = dyn.NilValue.AsFloat()
	assert.False(t, ok)

	vv2 := v.MustFloat()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind float, got nil", func() {
		dyn.NilValue.MustFloat()
	})

	// Test float64 type specifically.
	v = dyn.V(float64(1.0))
	vv1, ok = v.AsFloat()
	assert.True(t, ok)
	assert.Equal(t, float64(1.0), vv1)
}

func TestValueUnderlyingTime(t *testing.T) {
	v := dyn.V(time.Now())

	vv1, ok := v.AsTime()
	assert.True(t, ok)

	_, ok = dyn.NilValue.AsTime()
	assert.False(t, ok)

	vv2 := v.MustTime()
	assert.Equal(t, vv1, vv2)

	// Test panic.
	assert.PanicsWithValue(t, "expected kind time, got nil", func() {
		dyn.NilValue.MustTime()
	})
}
