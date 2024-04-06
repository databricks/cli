package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestGetWithEmptyPath(t *testing.T) {
	// An empty path means to return the value itself.
	vin := dyn.V(42)
	vout, err := dyn.GetByPath(vin, dyn.NewPath())
	assert.NoError(t, err)
	assert.Equal(t, vin, vout)
}

func TestGetOnNilValue(t *testing.T) {
	var err error
	_, err = dyn.GetByPath(dyn.NilValue, dyn.NewPath(dyn.Key("foo")))
	assert.ErrorContains(t, err, `expected a map to index "foo", found nil`)
	_, err = dyn.GetByPath(dyn.NilValue, dyn.NewPath(dyn.Index(42)))
	assert.ErrorContains(t, err, `expected a sequence to index "[42]", found nil`)
}

func TestGetOnMap(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(42),
		"bar": dyn.V(43),
	})

	var err error

	_, err = dyn.GetByPath(vin, dyn.NewPath(dyn.Index(42)))
	assert.ErrorContains(t, err, `expected a sequence to index "[42]", found map`)

	_, err = dyn.GetByPath(vin, dyn.NewPath(dyn.Key("baz")))
	assert.True(t, dyn.IsNoSuchKeyError(err))
	assert.ErrorContains(t, err, `key not found at "baz"`)

	vfoo, err := dyn.GetByPath(vin, dyn.NewPath(dyn.Key("foo")))
	assert.NoError(t, err)
	assert.Equal(t, dyn.V(42), vfoo)

	vbar, err := dyn.GetByPath(vin, dyn.NewPath(dyn.Key("bar")))
	assert.NoError(t, err)
	assert.Equal(t, dyn.V(43), vbar)
}

func TestGetOnSequence(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V(42),
		dyn.V(43),
	})

	var err error

	_, err = dyn.GetByPath(vin, dyn.NewPath(dyn.Key("foo")))
	assert.ErrorContains(t, err, `expected a map to index "foo", found sequence`)

	_, err = dyn.GetByPath(vin, dyn.NewPath(dyn.Index(-1)))
	assert.True(t, dyn.IsIndexOutOfBoundsError(err))
	assert.ErrorContains(t, err, `index out of bounds at "[-1]"`)

	_, err = dyn.GetByPath(vin, dyn.NewPath(dyn.Index(2)))
	assert.True(t, dyn.IsIndexOutOfBoundsError(err))
	assert.ErrorContains(t, err, `index out of bounds at "[2]"`)

	v0, err := dyn.GetByPath(vin, dyn.NewPath(dyn.Index(0)))
	assert.NoError(t, err)
	assert.Equal(t, dyn.V(42), v0)

	v1, err := dyn.GetByPath(vin, dyn.NewPath(dyn.Index(1)))
	assert.NoError(t, err)
	assert.Equal(t, dyn.V(43), v1)
}
