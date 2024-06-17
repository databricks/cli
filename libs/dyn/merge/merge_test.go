package merge

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestMergeMaps(t *testing.T) {
	l1 := dyn.Location{File: "file1", Line: 1, Column: 2}
	v1 := dyn.NewValue(map[string]dyn.Value{
		"foo": dyn.NewValue("bar", l1),
		"bar": dyn.NewValue("baz", l1),
	}, l1)

	l2 := dyn.Location{File: "file2", Line: 3, Column: 4}
	v2 := dyn.NewValue(map[string]dyn.Value{
		"bar": dyn.NewValue("qux", l2),
		"qux": dyn.NewValue("foo", l2),
	}, l2)

	// Merge v2 into v1.
	{
		out, err := Merge(v1, v2)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
			"bar": "qux",
			"qux": "foo",
		}, out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l1, l2}, out.YamlLocations())
		assert.Equal(t, []dyn.Location{l2, l1}, out.Get("bar").YamlLocations())
		assert.Equal(t, []dyn.Location{l1}, out.Get("foo").YamlLocations())
		assert.Equal(t, []dyn.Location{l2}, out.Get("qux").YamlLocations())
	}

	// Merge v1 into v2.
	{
		out, err := Merge(v2, v1)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
			"bar": "baz",
			"qux": "foo",
		}, out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l2, l1}, out.YamlLocations())
		assert.Equal(t, []dyn.Location{l1, l2}, out.Get("bar").YamlLocations())
		assert.Equal(t, []dyn.Location{l1}, out.Get("foo").YamlLocations())
		assert.Equal(t, []dyn.Location{l2}, out.Get("qux").YamlLocations())
	}
}

func TestMergeMapsNil(t *testing.T) {
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
	})

	// Merge nil into v.
	{
		out, err := Merge(v, dyn.NilValue)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
		}, out.AsAny())
	}

	// Merge v into nil.
	{
		out, err := Merge(dyn.NilValue, v)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
		}, out.AsAny())
	}
}

func TestMergeMapsError(t *testing.T) {
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
	})

	other := dyn.V("string")

	// Merge a string into v.
	{
		out, err := Merge(v, other)
		assert.EqualError(t, err, "cannot merge map with string")
		assert.Equal(t, dyn.NilValue, out)
	}
}

func TestMergeSequences(t *testing.T) {
	l1 := dyn.Location{File: "file1", Line: 1, Column: 2}
	v1 := dyn.NewValue([]dyn.Value{
		dyn.NewValue("bar", l1),
		dyn.NewValue("baz", l1),
	}, l1)

	l2 := dyn.Location{File: "file2", Line: 3, Column: 4}
	v2 := dyn.NewValue([]dyn.Value{
		dyn.NewValue("qux", l2),
		dyn.NewValue("foo", l2),
	}, l2)

	// Merge v2 into v1.
	{
		out, err := Merge(v1, v2)
		assert.NoError(t, err)
		assert.Equal(t, []any{
			"bar",
			"baz",
			"qux",
			"foo",
		}, out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l1, l2}, out.YamlLocations())
	}

	// Merge v1 into v2.
	{
		out, err := Merge(v2, v1)
		assert.NoError(t, err)
		assert.Equal(t, []any{
			"qux",
			"foo",
			"bar",
			"baz",
		}, out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l2, l1}, out.YamlLocations())
	}
}

func TestMergeSequencesNil(t *testing.T) {
	v := dyn.V([]dyn.Value{
		dyn.V("bar"),
	})

	// Merge nil into v.
	{
		out, err := Merge(v, dyn.NilValue)
		assert.NoError(t, err)
		assert.Equal(t, []any{
			"bar",
		}, out.AsAny())
	}

	// Merge v into nil.
	{
		out, err := Merge(dyn.NilValue, v)
		assert.NoError(t, err)
		assert.Equal(t, []any{
			"bar",
		}, out.AsAny())
	}
}

func TestMergeSequencesError(t *testing.T) {
	v := dyn.V([]dyn.Value{
		dyn.V("bar"),
	})

	other := dyn.V("string")

	// Merge a string into v.
	{
		out, err := Merge(v, other)
		assert.EqualError(t, err, "cannot merge sequence with string")
		assert.Equal(t, dyn.NilValue, out)
	}
}

func TestMergePrimitives(t *testing.T) {
	l1 := dyn.Location{File: "file1", Line: 1, Column: 2}
	l2 := dyn.Location{File: "file2", Line: 3, Column: 4}
	v1 := dyn.NewValue("bar", l1)
	v2 := dyn.NewValue("baz", l2)

	// Merge v2 into v1.
	{
		out, err := Merge(v1, v2)
		assert.NoError(t, err)
		assert.Equal(t, "baz", out.AsAny())

		// Location of both values should be preserved.
		assert.Equal(t, []dyn.Location{l2, l1}, out.YamlLocations())
	}

	// Merge v1 into v2.
	{
		out, err := Merge(v2, v1)
		assert.NoError(t, err)
		assert.Equal(t, "bar", out.AsAny())

		// Location of both values should be preserved.
		assert.Equal(t, []dyn.Location{l1, l2}, out.YamlLocations())
	}
}

func TestMergePrimitivesNil(t *testing.T) {
	v := dyn.V("bar")

	// Merge nil into v.
	{
		out, err := Merge(v, dyn.NilValue)
		assert.NoError(t, err)
		assert.Equal(t, "bar", out.AsAny())
	}

	// Merge v into nil.
	{
		out, err := Merge(dyn.NilValue, v)
		assert.NoError(t, err)
		assert.Equal(t, "bar", out.AsAny())
	}
}

func TestMergePrimitivesError(t *testing.T) {
	v := dyn.V("bar")
	other := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
	})

	// Merge a map into v.
	{
		out, err := Merge(v, other)
		assert.EqualError(t, err, "cannot merge string with map")
		assert.Equal(t, dyn.NilValue, out)
	}
}
