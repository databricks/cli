package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestSetWithEmptyPath(t *testing.T) {
	// An empty path means to return the value itself.
	vin := dyn.V(42)
	vout, err := dyn.SetByPath(dyn.InvalidValue, dyn.EmptyPath, vin)
	assert.NoError(t, err)
	assert.Equal(t, vin, vout)
}

func TestSetOnNilValue(t *testing.T) {
	var err error
	_, err = dyn.SetByPath(dyn.NilValue, dyn.NewPath(dyn.Key("foo")), dyn.V(42))
	assert.ErrorContains(t, err, `expected a map to index "foo", found nil`)
	_, err = dyn.SetByPath(dyn.NilValue, dyn.NewPath(dyn.Index(42)), dyn.V(42))
	assert.ErrorContains(t, err, `expected a sequence to index "[42]", found nil`)
}

func TestSetOnMap(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(42),
		"bar": dyn.V(43),
	})

	var err error

	_, err = dyn.SetByPath(vin, dyn.NewPath(dyn.Index(42)), dyn.V(42))
	assert.ErrorContains(t, err, `expected a sequence to index "[42]", found map`)

	// Note: in the test cases below we implicitly test that the original
	// value is not modified as we repeatedly set values on it.

	vfoo, err := dyn.SetByPath(vin, dyn.NewPath(dyn.Key("foo")), dyn.V(44))
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 44,
		"bar": 43,
	}, vfoo.AsAny())

	vbar, err := dyn.SetByPath(vin, dyn.NewPath(dyn.Key("bar")), dyn.V(45))
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 42,
		"bar": 45,
	}, vbar.AsAny())

	vbaz, err := dyn.SetByPath(vin, dyn.NewPath(dyn.Key("baz")), dyn.V(46))
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{
		"foo": 42,
		"bar": 43,
		"baz": 46,
	}, vbaz.AsAny())
}

func TestSetOnSequence(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V(42),
		dyn.V(43),
	})

	var err error

	_, err = dyn.SetByPath(vin, dyn.NewPath(dyn.Key("foo")), dyn.V(42))
	assert.ErrorContains(t, err, `expected a map to index "foo", found sequence`)

	// It is not allowed to set a value at an index that is out of bounds.
	_, err = dyn.SetByPath(vin, dyn.NewPath(dyn.Index(-1)), dyn.V(42))
	assert.True(t, dyn.IsIndexOutOfBoundsError(err))
	_, err = dyn.SetByPath(vin, dyn.NewPath(dyn.Index(2)), dyn.V(42))
	assert.True(t, dyn.IsIndexOutOfBoundsError(err))

	// Note: in the test cases below we implicitly test that the original
	// value is not modified as we repeatedly set values on it.

	v0, err := dyn.SetByPath(vin, dyn.NewPath(dyn.Index(0)), dyn.V(44))
	assert.NoError(t, err)
	assert.Equal(t, []any{44, 43}, v0.AsAny())

	v1, err := dyn.SetByPath(vin, dyn.NewPath(dyn.Index(1)), dyn.V(45))
	assert.NoError(t, err)
	assert.Equal(t, []any{42, 45}, v1.AsAny())
}
