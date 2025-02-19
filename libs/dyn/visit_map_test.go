package dyn_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
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
	var nv dyn.Value
	var err error
	nv, err = dyn.MapByPath(dyn.NilValue, dyn.NewPath(dyn.Key("foo")), nil)
	assert.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
	nv, err = dyn.MapByPath(dyn.NilValue, dyn.NewPath(dyn.Index(42)), nil)
	assert.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
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
	vfoo, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("foo")), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.NewPath(dyn.Key("foo")), p)
		assert.Equal(t, dyn.V(42), v)
		return dyn.V(44), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 44,
		"bar": 43,
	}, vfoo.AsAny())

	vbar, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Key("bar")), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.NewPath(dyn.Key("bar")), p)
		assert.Equal(t, dyn.V(43), v)
		return dyn.V(45), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 42,
		"bar": 45,
	}, vbar.AsAny())

	// Return error from map function.
	ref := errors.New("error")
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

	for i := range variants {
		vin := dyn.V(map[string]dyn.Value{
			"key": variants[i],
		})

		for j := range variants {
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
	v0, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(0)), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.NewPath(dyn.Index(0)), p)
		assert.Equal(t, dyn.V(42), v)
		return dyn.V(44), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{44, 43}, v0.AsAny())

	v1, err := dyn.MapByPath(vin, dyn.NewPath(dyn.Index(1)), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.NewPath(dyn.Index(1)), p)
		assert.Equal(t, dyn.V(43), v)
		return dyn.V(45), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{42, 45}, v1.AsAny())

	// Return error from map function.
	ref := errors.New("error")
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

	for i := range variants {
		vin := dyn.V([]dyn.Value{
			variants[i],
		})

		for j := range variants {
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
	ref := errors.New("error")
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
	ref := errors.New("error")
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

func TestMapForeachOnNil(t *testing.T) {
	vin := dyn.NilValue

	// Check that if foreach is applied to nil, it returns nil.
	vout, err := dyn.Map(vin, ".", dyn.Foreach(func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.InvalidValue, nil
	}))
	assert.NoError(t, err)
	assert.Equal(t, dyn.NilValue, vout)
}

func TestMapByPatternOnNilValue(t *testing.T) {
	var err error
	_, err = dyn.MapByPattern(dyn.NilValue, dyn.NewPattern(dyn.AnyKey()), nil)
	assert.ErrorContains(t, err, `expected a map at "", found nil`)
	_, err = dyn.MapByPattern(dyn.NilValue, dyn.NewPattern(dyn.AnyIndex()), nil)
	assert.ErrorContains(t, err, `expected a sequence at "", found nil`)
}

func TestMapByPatternOnMap(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"a": dyn.V(map[string]dyn.Value{
			"b": dyn.V(42),
		}),
		"b": dyn.V(map[string]dyn.Value{
			"c": dyn.V(43),
		}),
	})

	var err error

	// Expect an error if the pattern structure doesn't match the value structure.
	_, err = dyn.MapByPattern(vin, dyn.NewPattern(dyn.AnyKey(), dyn.Index(0)), nil)
	assert.ErrorContains(t, err, `expected a sequence to index`)

	// Apply function to pattern "*.b".
	vout, err := dyn.MapByPattern(vin, dyn.NewPattern(dyn.AnyKey(), dyn.Key("b")), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.NewPath(dyn.Key("a"), dyn.Key("b")), p)
		assert.Equal(t, dyn.V(42), v)
		return dyn.V(44), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"a": map[string]any{
			"b": 44,
		},
		"b": map[string]any{
			"c": 43,
		},
	}, vout.AsAny())
}

func TestMapByPatternOnMapWithoutMatch(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"a": dyn.V(map[string]dyn.Value{
			"b": dyn.V(42),
		}),
		"b": dyn.V(map[string]dyn.Value{
			"c": dyn.V(43),
		}),
	})

	// Apply function to pattern "*.zzz".
	vout, err := dyn.MapByPattern(vin, dyn.NewPattern(dyn.AnyKey(), dyn.Key("zzz")), nil)
	assert.NoError(t, err)
	assert.Equal(t, vin, vout)
}

func TestMapByPatternOnSequence(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V([]dyn.Value{
			dyn.V(42),
		}),
		dyn.V([]dyn.Value{
			dyn.V(43),
			dyn.V(44),
		}),
	})

	var err error

	// Expect an error if the pattern structure doesn't match the value structure.
	_, err = dyn.MapByPattern(vin, dyn.NewPattern(dyn.AnyIndex(), dyn.Key("a")), nil)
	assert.ErrorContains(t, err, `expected a map to index`)

	// Apply function to pattern "*.c".
	vout, err := dyn.MapByPattern(vin, dyn.NewPattern(dyn.AnyIndex(), dyn.Index(1)), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		assert.Equal(t, dyn.NewPath(dyn.Index(1), dyn.Index(1)), p)
		assert.Equal(t, dyn.V(44), v)
		return dyn.V(45), nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{
		[]any{42},
		[]any{43, 45},
	}, vout.AsAny())
}

func TestMapByPatternOnSequenceWithoutMatch(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V([]dyn.Value{
			dyn.V(42),
		}),
		dyn.V([]dyn.Value{
			dyn.V(43),
			dyn.V(44),
		}),
	})

	// Apply function to pattern "*.zzz".
	vout, err := dyn.MapByPattern(vin, dyn.NewPattern(dyn.AnyIndex(), dyn.Index(42)), nil)
	assert.NoError(t, err)
	assert.Equal(t, vin, vout)
}
