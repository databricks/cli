package dynvar_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getByPath(t *testing.T, v dyn.Value, path string) dyn.Value {
	v, err := dyn.Get(v, path)
	require.NoError(t, err)
	return v
}

func TestInterpolation(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${a}"),
		"c": dyn.V("${a}"),
	})

	out, err := dynvar.Interpolate(in)
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "b").MustString())
	assert.Equal(t, "a", getByPath(t, out, "c").MustString())
}

func TestInterpolationWithNesting(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("${f.a}"),
		"f": dyn.V(map[string]dyn.Value{
			"a": dyn.V("a"),
			"b": dyn.V("${f.a}"),
		}),
	})

	out, err := dynvar.Interpolate(in)
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "f.a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "f.b").MustString())
}

func TestInterpolationWithRecursion(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${a}"),
		"c": dyn.V("${b}"),
	})

	out, err := dynvar.Interpolate(in)
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "b").MustString())
	assert.Equal(t, "a", getByPath(t, out, "c").MustString())
}

func TestInterpolationWithRecursionLoop(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${c}"),
		"c": dyn.V("${d}"),
		"d": dyn.V("${b}"),
	})

	_, err := dynvar.Interpolate(in)
	assert.ErrorContains(t, err, "cycle detected in field resolution: b -> c -> d -> b")
}

func TestInterpolationWithRecursionLoopSelf(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("${a}"),
	})

	_, err := dynvar.Interpolate(in)
	assert.ErrorContains(t, err, "cycle detected in field resolution: a -> a")
}

func TestInterpolationWithTypeRetention(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"int":         dyn.V(1),
		"int_":        dyn.V("${int}"),
		"bool_true":   dyn.V(true),
		"bool_true_":  dyn.V("${bool_true}"),
		"bool_false":  dyn.V(false),
		"bool_false_": dyn.V("${bool_false}"),
		"float":       dyn.V(1.0),
		"float_":      dyn.V("${float}"),
		"string":      dyn.V("a"),
		"string_":     dyn.V("${string}"),
	})

	out, err := dynvar.Interpolate(in)
	require.NoError(t, err)

	assert.EqualValues(t, 1, getByPath(t, out, "int").MustInt())
	assert.EqualValues(t, 1, getByPath(t, out, "int_").MustInt())

	assert.EqualValues(t, true, getByPath(t, out, "bool_true").MustBool())
	assert.EqualValues(t, true, getByPath(t, out, "bool_true_").MustBool())

	assert.EqualValues(t, false, getByPath(t, out, "bool_false").MustBool())
	assert.EqualValues(t, false, getByPath(t, out, "bool_false_").MustBool())

	assert.EqualValues(t, 1.0, getByPath(t, out, "float").MustFloat())
	assert.EqualValues(t, 1.0, getByPath(t, out, "float_").MustFloat())

	assert.EqualValues(t, "a", getByPath(t, out, "string").MustString())
	assert.EqualValues(t, "a", getByPath(t, out, "string_").MustString())
}
