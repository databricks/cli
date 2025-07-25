package dyn_test

import (
	"errors"
	"testing"

	. "github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

// Return values for specific paths.
type walkReturn struct {
	path Path

	// Return values.
	fn  func(Value) Value
	err error
}

// Track the calls to the callback.
type walkCall struct {
	path  Path
	value Value
}

// Track the calls to the callback.
type walkCallTracker struct {
	returns []walkReturn
	calls   []walkCall
}

func (w *walkCallTracker) on(path string, fn func(Value) Value, err error) {
	w.returns = append(w.returns, walkReturn{MustPathFromString(path), fn, err})
}

func (w *walkCallTracker) returnSkip(path string) {
	w.on(path, func(v Value) Value { return v }, ErrSkip)
}

func (w *walkCallTracker) returnDrop(path string) {
	w.on(path, func(v Value) Value { return InvalidValue }, ErrDrop)
}

func (w *walkCallTracker) track(p Path, v Value) (Value, error) {
	w.calls = append(w.calls, walkCall{p, v})

	// Look for matching return.
	for _, r := range w.returns {
		if p.Equal(r.path) {
			return r.fn(v), r.err
		}
	}

	return v, nil
}

func TestWalkEmpty(t *testing.T) {
	var tracker walkCallTracker

	value := V(nil)
	out, err := Walk(value, tracker.track)
	require.NoError(t, err)
	assert.Equal(t, value, out)

	// The callback should have been called once.
	assert.Len(t, tracker.calls, 1)

	// The call should have been made with the empty path.
	assert.Equal(t, EmptyPath, tracker.calls[0].path)

	// The value should be the same as the input.
	assert.Equal(t, value, tracker.calls[0].value)
}

func TestWalkMapSkip(t *testing.T) {
	var tracker walkCallTracker

	// Skip traversal of the root value.
	tracker.returnSkip(".")

	value := V(map[string]Value{
		"key": V("value"),
	})
	out, err := Walk(value, tracker.track)
	require.NoError(t, err)
	assert.Equal(
		t,
		V(map[string]Value{
			"key": V("value"),
		}),
		out,
	)

	// The callback should have been called once.
	assert.Len(t, tracker.calls, 1)

	// The call should have been made with the empty path.
	assert.Equal(t, EmptyPath, tracker.calls[0].path)

	// The value should be the same as the input.
	assert.Equal(t, value, tracker.calls[0].value)
}

func TestWalkMapDrop(t *testing.T) {
	var tracker walkCallTracker

	// Drop the value at key "foo".
	tracker.returnDrop(".foo")

	value := V(map[string]Value{
		"foo": V("bar"),
		"bar": V("baz"),
	})
	out, err := Walk(value, tracker.track)
	require.NoError(t, err)
	assert.Equal(
		t,
		V(map[string]Value{
			"bar": V("baz"),
		}),
		out,
	)

	// The callback should have been called for the root and every key in the map.
	assert.Len(t, tracker.calls, 3)

	// Calls 2 and 3 have been made for the keys in the map.
	assert.ElementsMatch(t,
		[]Path{
			tracker.calls[1].path,
			tracker.calls[2].path,
		}, []Path{
			MustPathFromString(".foo"),
			MustPathFromString(".bar"),
		})
}

func TestWalkMapError(t *testing.T) {
	var tracker walkCallTracker

	// Return an error from the callback for key "foo".
	cerr := errors.New("error!")
	tracker.on(".foo", func(v Value) Value { return v }, cerr)

	value := V(map[string]Value{
		"foo": V("bar"),
	})
	out, err := Walk(value, tracker.track)
	assert.Equal(t, cerr, err)
	assert.Equal(t, InvalidValue, out)

	// The callback should have been called twice.
	assert.Len(t, tracker.calls, 2)

	// The second call was for the value at key "foo".
	assert.Equal(t, MustPathFromString(".foo"), tracker.calls[1].path)
}

func TestWalkSequenceSkip(t *testing.T) {
	var tracker walkCallTracker

	// Skip traversal of the root value.
	tracker.returnSkip(".")

	value := V([]Value{
		V("foo"),
		V("bar"),
	})
	out, err := Walk(value, tracker.track)
	require.NoError(t, err)
	assert.Equal(
		t,
		V([]Value{
			V("foo"),
			V("bar"),
		}),
		out,
	)

	// The callback should have been called once.
	assert.Len(t, tracker.calls, 1)

	// The call should have been made with the empty path.
	assert.Equal(t, EmptyPath, tracker.calls[0].path)

	// The value should be the same as the input.
	assert.Equal(t, value, tracker.calls[0].value)
}

func TestWalkSequenceDrop(t *testing.T) {
	var tracker walkCallTracker

	// Drop the value at index 1.
	tracker.returnDrop(".[1]")

	value := V([]Value{
		V("foo"),
		V("bar"),
		V("baz"),
	})
	out, err := Walk(value, tracker.track)
	require.NoError(t, err)
	assert.Equal(
		t,
		V([]Value{
			V("foo"),
			V("baz"),
		}),
		out,
	)

	// The callback should have been called for the root and every value in the sequence.
	assert.Len(t, tracker.calls, 4)

	// The second call was for the value at index 0.
	assert.Equal(t, MustPathFromString(".[0]"), tracker.calls[1].path)
	assert.Equal(t, V("foo"), tracker.calls[1].value)

	// The third call was for the value at index 1.
	assert.Equal(t, MustPathFromString(".[1]"), tracker.calls[2].path)
	assert.Equal(t, V("bar"), tracker.calls[2].value)

	// The fourth call was for the value at index 2.
	assert.Equal(t, MustPathFromString(".[2]"), tracker.calls[3].path)
	assert.Equal(t, V("baz"), tracker.calls[3].value)
}

func TestWalkSequenceError(t *testing.T) {
	var tracker walkCallTracker

	// Return an error from the callback for index 1.
	cerr := errors.New("error!")
	tracker.on(".[1]", func(v Value) Value { return v }, cerr)

	value := V([]Value{
		V("foo"),
		V("bar"),
	})
	out, err := Walk(value, tracker.track)
	assert.Equal(t, cerr, err)
	assert.Equal(t, InvalidValue, out)

	// The callback should have been called three times.
	assert.Len(t, tracker.calls, 3)

	// The second call was for the value at index 0.
	assert.Equal(t, MustPathFromString(".[0]"), tracker.calls[1].path)
	assert.Equal(t, V("foo"), tracker.calls[1].value)

	// The third call was for the value at index 1.
	assert.Equal(t, MustPathFromString(".[1]"), tracker.calls[2].path)
	assert.Equal(t, V("bar"), tracker.calls[2].value)
}

func TestCollectLeafPaths(t *testing.T) {
	v := V(map[string]Value{
		"a": V(1),
		"b": V(map[string]Value{
			"c": V(2),
			"d": V(map[string]Value{
				"e": V(3),
			}),
		}),
		"f": V([]Value{V(4), V(5)}),
	})
	paths := CollectLeafPaths(v)
	assert.ElementsMatch(t, []string{"a", "b.c", "b.d.e", "f[0]", "f[1]"}, paths)
}
