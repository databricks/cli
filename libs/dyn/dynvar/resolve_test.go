package dynvar_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/cli/libs/dyn/dynvar"
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

func TestResolveWithSkipEverything(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("b"),
		"c": dyn.V("${a}"),
		"d": dyn.V("${b}"),
		"e": dyn.V("${a} ${b}"),
		"f": dyn.V("${b} ${a} ${a} ${b}"),
		"g": dyn.V("${d} ${c} ${c} ${d}"),
	})

	// The call must not replace anything if the lookup function returns ErrSkipResolution.
	out, err := dynvar.Resolve(in, func(path dyn.Path) (dyn.Value, error) {
		return dyn.InvalidValue, dynvar.ErrSkipResolution
	})
	require.NoError(t, err)
	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "b", getByPath(t, out, "b").MustString())
	assert.Equal(t, "${a}", getByPath(t, out, "c").MustString())
	assert.Equal(t, "${b}", getByPath(t, out, "d").MustString())
	assert.Equal(t, "${a} ${b}", getByPath(t, out, "e").MustString())
	assert.Equal(t, "${b} ${a} ${a} ${b}", getByPath(t, out, "f").MustString())
	assert.Equal(t, "${d} ${c} ${c} ${d}", getByPath(t, out, "g").MustString())
}

func TestResolveWithInterpolateNewRef(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${a}"),
	})

	// The call replaces ${a} with ${foobar} and skips everything else.
	out, err := dynvar.Resolve(in, func(path dyn.Path) (dyn.Value, error) {
		if path.String() == "a" {
			return dyn.V("${foobar}"), nil
		}
		return dyn.InvalidValue, dynvar.ErrSkipResolution
	})

	require.NoError(t, err)
	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "${foobar}", getByPath(t, out, "b").MustString())
}

func TestResolveWithInterpolateAliasedRef(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("${a}"),
		"c": dyn.V("${x}"),
	})

	// The call replaces ${x} with ${b} and skips everything else.
	out, err := dynvar.Resolve(in, func(path dyn.Path) (dyn.Value, error) {
		if path.String() == "x" {
			return dyn.V("${b}"), nil
		}
		return dyn.GetByPath(in, path)
	})

	require.NoError(t, err)
	assert.Equal(t, "a", getByPath(t, out, "a").MustString())
	assert.Equal(t, "a", getByPath(t, out, "b").MustString())
	assert.Equal(t, "a", getByPath(t, out, "c").MustString())
}

func TestResolveIndexedRefs(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"slice": dyn.V([]dyn.Value{dyn.V("a"), dyn.V("b")}),
		"a":     dyn.V("a: ${slice[0]}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a: a", getByPath(t, out, "a").MustString())
}

func TestResolveIndexedRefsFromMap(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"map": dyn.V(
			map[string]dyn.Value{
				"slice": dyn.V([]dyn.Value{dyn.V("a")}),
			}),
		"a": dyn.V("a: ${map.slice[0]}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a: a", getByPath(t, out, "a").MustString())
}

func TestResolveMapFieldFromIndexedRefs(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"map": dyn.V(
			map[string]dyn.Value{
				"slice": dyn.V([]dyn.Value{
					dyn.V(map[string]dyn.Value{
						"value": dyn.V("a"),
					}),
				}),
			}),
		"a": dyn.V("a: ${map.slice[0].value}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a: a", getByPath(t, out, "a").MustString())
}

func TestResolveNestedIndexedRefs(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"slice": dyn.V([]dyn.Value{
			dyn.V([]dyn.Value{dyn.V("a")}),
		}),
		"a": dyn.V("a: ${slice[0][0]}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	assert.Equal(t, "a: a", getByPath(t, out, "a").MustString())
}

func TestResolveMapVariable(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"map": dyn.V(map[string]dyn.Value{
			"key1": dyn.V("value1"),
			"key2": dyn.V("value2"),
		}),
		"var": dyn.V("${map}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	// Verify that the map variable was interpolated correctly
	mapVal := getByPath(t, out, "var")
	_, ok := mapVal.AsMap()
	require.True(t, ok, "expected map value")

	// Verify the map contents
	assert.Equal(t, "value1", getByPath(t, mapVal, "key1").MustString())
	assert.Equal(t, "value2", getByPath(t, mapVal, "key2").MustString())
}

func TestResolveSequenceVariable(t *testing.T) {
	in := dyn.V(map[string]dyn.Value{
		"seq": dyn.V([]dyn.Value{
			dyn.V("value1"),
			dyn.V("value2"),
		}),
		"var": dyn.V("${seq}"),
	})

	out, err := dynvar.Resolve(in, dynvar.DefaultLookup(in))
	require.NoError(t, err)

	// Verify that the sequence variable was interpolated correctly
	seqVal := getByPath(t, out, "var")
	seq, ok := seqVal.AsSequence()
	require.True(t, ok, "expected sequence value")
	require.Len(t, seq, 2)

	// Verify the sequence contents
	assert.Equal(t, "value1", seq[0].MustString())
	assert.Equal(t, "value2", seq[1].MustString())
}
