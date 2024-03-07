package dyn_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapWithEmptyPath(t *testing.T) {
	// An empty path means to return the value itself.
	vin := dyn.V(42)
	vout, err := dyn.MapByPath(dyn.InvalidValue, dyn.EmptyPath, func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
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
	vfoo, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("foo")), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.V(42), v)
		return dyn.V(44), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 44,
		"bar": 43,
	}, vfoo.AsAny())

	vbar, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("bar")), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
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
	verr, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("foo")), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, ref
	})
	assert.Equal(t, dyn.InvalidValue, verr)
	assert.ErrorIs(t, err, ref)
}

func TestMapFuncOnMapWithEmptySequence(t *testing.T) {
	variants := []dyn.Value{
		// empty sequence
		dyn.V([]dyn.Value{}),
		// non-empty sequence
		dyn.V([]dyn.Value{dyn.V(42)}),
	}

	for i := 0; i < len(variants); i++ {
		vin := dyn.V(map[string]dyn.Value{
			"key": variants[i],
		})

		for j := 0; j < len(variants); j++ {
			vout, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("key")), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
				return variants[j], nil
			})
			assert.NoError(t, err)
			assert.Equal(t, variants[j], vout.Get("key"))
		}
	}
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
	v0, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(0)), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.V(42), v)
		return dyn.V(44), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{44, 43}, v0.AsAny())

	v1, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(1)), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.V(43), v)
		return dyn.V(45), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{42, 45}, v1.AsAny())

	// Return error from map function.
	var ref = fmt.Errorf("error")
	verr, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(0)), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, ref
	})
	assert.Equal(t, dyn.InvalidValue, verr)
	assert.ErrorIs(t, err, ref)
}

func TestMapFuncOnSequenceWithEmptySequence(t *testing.T) {
	variants := []dyn.Value{
		// empty sequence
		dyn.V([]dyn.Value{}),
		// non-empty sequence
		dyn.V([]dyn.Value{dyn.V(42)}),
	}

	for i := 0; i < len(variants); i++ {
		vin := dyn.V([]dyn.Value{
			variants[i],
		})

		for j := 0; j < len(variants); j++ {
			vout, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(0)), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
				return variants[j], nil
			})
			assert.NoError(t, err)
			assert.Equal(t, variants[j], vout.Index(0))
		}
	}
}

func TestMapForeachOnMap(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(42),
		"bar": dyn.V(43),
	})

	var err error

	// Run foreach, adding 1 to each of the elements.
	vout, err := dyn.Map(vin, ".", dyn.Foreach(func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		i, ok := v.AsInt()
		require.True(t, ok, "expected an integer")
		switch p[0].Key() {
		case "foo":
			assert.EqualValues(t, 42, i)
			return dyn.V(43), nil
		case "bar":
			assert.EqualValues(t, 43, i)
			return dyn.V(44), nil
		default:
			return dyn.InvalidValue, fmt.Errorf("unexpected key %q", p[0].Key())
		}
	}))
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 43,
		"bar": 44,
	}, vout.AsAny())

	// Check that the original has not been modified.
	assert.Equal(t, map[string]any{
		"foo": 42,
		"bar": 43,
	}, vin.AsAny())
}

func TestMapForeachOnMapError(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(42),
		"bar": dyn.V(43),
	})

	// Check that an error from the map function propagates.
	var ref = fmt.Errorf("error")
	_, err := dyn.Map(vin, ".", dyn.Foreach(func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, ref
	}))
	assert.ErrorIs(t, err, ref)
}

func TestMapForeachOnSequence(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V(42),
		dyn.V(43),
	})

	var err error

	// Run foreach, adding 1 to each of the elements.
	vout, err := dyn.Map(vin, ".", dyn.Foreach(func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		i, ok := v.AsInt()
		require.True(t, ok, "expected an integer")
		switch p[0].Index() {
		case 0:
			assert.EqualValues(t, 42, i)
			return dyn.V(43), nil
		case 1:
			assert.EqualValues(t, 43, i)
			return dyn.V(44), nil
		default:
			return dyn.InvalidValue, fmt.Errorf("unexpected index %d", p[0].Index())
		}
	}))
	assert.NoError(t, err)
	assert.Equal(t, []any{43, 44}, vout.AsAny())

	// Check that the original has not been modified.
	assert.Equal(t, []any{42, 43}, vin.AsAny())
}

func TestMapForeachOnSequenceError(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V(42),
		dyn.V(43),
	})

	// Check that an error from the map function propagates.
	var ref = fmt.Errorf("error")
	_, err := dyn.Map(vin, ".", dyn.Foreach(func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, ref
	}))
	assert.ErrorIs(t, err, ref)
}

func TestMapForeachOnOtherError(t *testing.T) {
	vin := dyn.V(42)

	// Check that if foreach is applied to something other than a map or a sequence, it returns an error.
	_, err := dyn.Map(vin, ".", dyn.Foreach(func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, nil
	}))
	assert.ErrorContains(t, err, "expected a map or sequence, found int")
}
