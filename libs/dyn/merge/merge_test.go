package merge

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestMergeMaps(t *testing.T) {
	l1 := dyn.Location{File: "file1", Line: 1, Column: 2}
	v1 := dyn.NewValue(map[string]dyn.Value{
		"foo": dyn.NewValue("bar", []dyn.Location{l1}),
		"bar": dyn.NewValue("baz", []dyn.Location{l1}),
	}, []dyn.Location{l1})

	l2 := dyn.Location{File: "file2", Line: 3, Column: 4}
	v2 := dyn.NewValue(map[string]dyn.Value{
		"bar": dyn.NewValue("qux", []dyn.Location{l2}),
		"qux": dyn.NewValue("foo", []dyn.Location{l2}),
	}, []dyn.Location{l2})

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
		assert.Equal(t, []dyn.Location{l1, l2}, out.Locations())
		assert.Equal(t, []dyn.Location{l2, l1}, out.Get("bar").Locations())
		assert.Equal(t, []dyn.Location{l1}, out.Get("foo").Locations())
		assert.Equal(t, []dyn.Location{l2}, out.Get("qux").Locations())

		// Location of the merged value should be the location of v1.
		assert.Equal(t, l1, out.Location())

		// Value of bar is "qux" which comes from v2. This .Location() should
		// return the location of v2.
		assert.Equal(t, l2, out.Get("bar").Location())

		// Original locations of keys that were not overwritten should be preserved.
		assert.Equal(t, l1, out.Get("foo").Location())
		assert.Equal(t, l2, out.Get("qux").Location())
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
		assert.Equal(t, []dyn.Location{l2, l1}, out.Locations())
		assert.Equal(t, []dyn.Location{l1, l2}, out.Get("bar").Locations())
		assert.Equal(t, []dyn.Location{l1}, out.Get("foo").Locations())
		assert.Equal(t, []dyn.Location{l2}, out.Get("qux").Locations())

		// Location of the merged value should be the location of v2.
		assert.Equal(t, l2, out.Location())

		// Value of bar is "baz" which comes from v1. This .Location() should
		// return the location of v1.
		assert.Equal(t, l1, out.Get("bar").Location())

		// Original locations of keys that were not overwritten should be preserved.
		assert.Equal(t, l1, out.Get("foo").Location())
		assert.Equal(t, l2, out.Get("qux").Location())
	}
}

func TestMergeMapsNil(t *testing.T) {
	l := dyn.Location{File: "file", Line: 1, Column: 2}
	v := dyn.NewValue(map[string]dyn.Value{
		"foo": dyn.V("bar"),
	}, []dyn.Location{l})

	nilL := dyn.Location{File: "file", Line: 3, Column: 4}
	nilV := dyn.NewValue(nil, []dyn.Location{nilL})

	// Merge nil into v.
	{
		out, err := Merge(v, nilV)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
		}, out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l, nilL}, out.Locations())

		// Location of the non-nil value should be returned by .Location().
		assert.Equal(t, l, out.Location())
	}

	// Merge v into nil.
	{
		out, err := Merge(nilV, v)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
		}, out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l, nilL}, out.Locations())

		// Location of the non-nil value should be returned by .Location().
		assert.Equal(t, l, out.Location())
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
		assert.Equal(t, dyn.InvalidValue, out)
	}
}

func TestMergeSequences(t *testing.T) {
	l1 := dyn.Location{File: "file1", Line: 1, Column: 2}
	v1 := dyn.NewValue([]dyn.Value{
		dyn.NewValue("bar", []dyn.Location{l1}),
		dyn.NewValue("baz", []dyn.Location{l1}),
	}, []dyn.Location{l1})

	l2 := dyn.Location{File: "file2", Line: 3, Column: 4}
	l3 := dyn.Location{File: "file3", Line: 5, Column: 6}
	v2 := dyn.NewValue([]dyn.Value{
		dyn.NewValue("qux", []dyn.Location{l2}),
		dyn.NewValue("foo", []dyn.Location{l3}),
	}, []dyn.Location{l2, l3})

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
		assert.Equal(t, []dyn.Location{l1, l2, l3}, out.Locations())

		// Location of the merged value should be the location of v1.
		assert.Equal(t, l1, out.Location())

		// Location of the individual values should be preserved.
		assert.Equal(t, l1, out.Index(0).Location()) // "bar"
		assert.Equal(t, l1, out.Index(1).Location()) // "baz"
		assert.Equal(t, l2, out.Index(2).Location()) // "qux"
		assert.Equal(t, l3, out.Index(3).Location()) // "foo"
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
		assert.Equal(t, []dyn.Location{l2, l3, l1}, out.Locations())

		// Location of the merged value should be the location of v2.
		assert.Equal(t, l2, out.Location())

		// Location of the individual values should be preserved.
		assert.Equal(t, l2, out.Index(0).Location()) // "qux"
		assert.Equal(t, l3, out.Index(1).Location()) // "foo"
		assert.Equal(t, l1, out.Index(2).Location()) // "bar"
		assert.Equal(t, l1, out.Index(3).Location()) // "baz"
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
		assert.Equal(t, dyn.InvalidValue, out)
	}
}

func TestMergePrimitives(t *testing.T) {
	l1 := dyn.Location{File: "file1", Line: 1, Column: 2}
	l2 := dyn.Location{File: "file2", Line: 3, Column: 4}
	v1 := dyn.NewValue("bar", []dyn.Location{l1})
	v2 := dyn.NewValue("baz", []dyn.Location{l2})

	// Merge v2 into v1.
	{
		out, err := Merge(v1, v2)
		assert.NoError(t, err)
		assert.Equal(t, "baz", out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l2, l1}, out.Locations())

		// Location of the merged value should be the location of v2, the second value.
		assert.Equal(t, l2, out.Location())
	}

	// Merge v1 into v2.
	{
		out, err := Merge(v2, v1)
		assert.NoError(t, err)
		assert.Equal(t, "bar", out.AsAny())

		// Locations of both values should be preserved.
		assert.Equal(t, []dyn.Location{l1, l2}, out.Locations())

		// Location of the merged value should be the location of v1, the second value.
		assert.Equal(t, l1, out.Location())
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
		assert.Equal(t, dyn.InvalidValue, out)
	}
}
