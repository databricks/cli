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

func TestResolve(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${a}"),
		"c": dyn.V("${a}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "b").MustString())
	assert.Equal(t, "a", getByPath(t, out, "c").MustString())
}

func TestResolveNotFound(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"b": dyn.V("${a}"),
	})

	_, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.ErrorContains(t, err, `reference does not exist: ${a}`)
}

func TestResolveWithNesting(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("${f.a}"),
		"f": dyn.V(map[string]dyn.Value{
			"a": dyn.V("a"),
			"b": dyn.V("${f.a}"),
		}),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "f.a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "f.b").MustString())
}

func TestResolveWithRecursion(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${a}"),
		"c": dyn.V("${b}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "b").MustString())
	assert.Equal(t, "a", getByPath(t, out, "c").MustString())
}

func TestResolveWithRecursionLoop(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${c}"),
		"c": dyn.V("${d}"),
		"d": dyn.V("${b}"),
	})

	_, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	assert.ErrorContains(t, err, "cycle detected in field resolution: b -> c -> d -> b")
}

func TestResolveWithRecursionLoopSelf(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("${a}"),
	})

	_, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	assert.ErrorContains(t, err, "cycle detected in field resolution: a -> a")
}

func TestResolveWithStringConcatenation(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("b"),
		"c": dyn.V("${a}${b}${a}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "b", getByPath(t, out, "b").MustString())
	assert.Equal(t, "aba", getByPath(t, out, "c").MustString())
}

func TestResolveWithTypeRetentionFailure(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V(1),
		"b": dyn.V(2),
		"c": dyn.V("${a} ${b}"),
	})

	_, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.ErrorContains(t, err, "cannot interpolate non-string value: ${a}")
}

func TestResolveWithTypeRetention(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"int":            dyn.V(1),
		"int_var":        dyn.V("${int}"),
		"bool_true":      dyn.V(true),
		"bool_true_var":  dyn.V("${bool_true}"),
		"bool_false":     dyn.V(false),
		"bool_false_var": dyn.V("${bool_false}"),
		"float":          dyn.V(1.0),
		"float_var":      dyn.V("${float}"),
		"string":         dyn.V("a"),
		"string_var":     dyn.V("${string}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.EqualValues(t, 1, getByPath(t, out, "int").MustInt())
	assert.EqualValues(t, 1, getByPath(t, out, "int_var").MustInt())

	assert.EqualValues(t, true, getByPath(t, out, "bool_true").MustBool())
	assert.EqualValues(t, true, getByPath(t, out, "bool_true_var").MustBool())

	assert.EqualValues(t, false, getByPath(t, out, "bool_false").MustBool())
	assert.EqualValues(t, false, getByPath(t, out, "bool_false_var").MustBool())

	assert.EqualValues(t, 1.0, getByPath(t, out, "float").MustFloat())
	assert.EqualValues(t, 1.0, getByPath(t, out, "float_var").MustFloat())

	assert.EqualValues(t, "a", getByPath(t, out, "string").MustString())
	assert.EqualValues(t, "a", getByPath(t, out, "string_var").MustString())
}

func TestResolveWithSkip(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("b"),
		"c": dyn.V("${a}"),
		"d": dyn.V("${b}"),
		"e": dyn.V("${a} ${b}"),
		"f": dyn.V("${b} ${a} ${a} ${b}"),
	})

	fallback := dynvar.DefaultLookup(in)
	ignore := func(path dyn.Path) (dyn.Value, error) {
		// If the variable reference to look up starts with "b", skip it.
		if path.HasPrefix(dyn.NewPath(dyn.Key("b"))) {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}
		return fallback(path)
	}

	out, err := dynvar.Resolve(in, ignore)
	require.NoError(t, err)

	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "b", getByPath(t, out, "b").MustString())
	assert.Equal(t, "a", getByPath(t, out, "c").MustString())

	// Check that the skipped variable references are not interpolated.
	assert.Equal(t, "${b}", getByPath(t, out, "d").MustString())
	assert.Equal(t, "a ${b}", getByPath(t, out, "e").MustString())
	assert.Equal(t, "${b} a a ${b}", getByPath(t, out, "f").MustString())
}
