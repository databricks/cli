package dyn_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestMapWithEmptyPath(t *testing.T) {
	// An empty path means to return the value itself.
	vin := dyn.V(42)
	vout, err := dyn.MapByPath(dyn.InvalidValue, dyn.EmptyPath, func(v dyn.Value) (dyn.Value, error) {
		return vin, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, vin, vout)
}

func TestMapOnNilValue(t *testing.T) {
	var err error
	_, err = dyn.MapByPath(dyn.NilValue, dyn.NewPath(dyn.Key("foo")), nil)
	assert.ErrorContains(t, err, `expected a map to index "foo", found nil`)
	_, err = dyn.MapByPath(dyn.NilValue, dyn.NewPath(dyn.Index(42)), nil)
	assert.ErrorContains(t, err, `expected a sequence to index "[42]", found nil`)
}

func TestMapFuncOnMap(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(42),
		"bar": dyn.V(43),
	})

	var err error

	_, err = dyn.MapByPath(vin, dyn.NewPath(dyn.Index(42)), nil)
	assert.ErrorContains(t, err, `expected a sequence to index "[42]", found map`)

	// A key that does not exist is not an error.
	vout, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("baz")), nil)
	assert.NoError(t, err)
	assert.Equal(t, vin, vout)

	// Note: in the test cases below we implicitly test that the original
	// value is not modified as we repeatedly set values on it.
	vfoo, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("foo")), func(v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.V(42), v)
		return dyn.V(44), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 44,
		"bar": 43,
	}, vfoo.AsAny())

	vbar, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("bar")), func(v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.V(43), v)
		return dyn.V(45), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 42,
		"bar": 45,
	}, vbar.AsAny())

	// Return error from map function.
	var ref = fmt.Errorf("error")
	verr, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("foo")), func(v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, ref
	})
	assert.Equal(t, dyn.InvalidValue, verr)
	assert.ErrorIs(t, err, ref)
}

func TestMapFuncOnSequence(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V(42),
		dyn.V(43),
	})

	var err error

	_, err = dyn.MapByPath(vin, dyn.NewPath(dyn.Key("foo")), nil)
	assert.ErrorContains(t, err, `expected a map to index "foo", found sequence`)

	// An index that does not exist is not an error.
	vout, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(2)), nil)
	assert.NoError(t, err)
	assert.Equal(t, vin, vout)

	// Note: in the test cases below we implicitly test that the original
	// value is not modified as we repeatedly set values on it.
	v0, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(0)), func(v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.V(42), v)
		return dyn.V(44), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{44, 43}, v0.AsAny())

	v1, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(1)), func(v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.V(43), v)
		return dyn.V(45), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{42, 45}, v1.AsAny())

	// Return error from map function.
	var ref = fmt.Errorf("error")
	verr, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(0)), func(v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, ref
	})
	assert.Equal(t, dyn.InvalidValue, verr)
	assert.ErrorIs(t, err, ref)
}
