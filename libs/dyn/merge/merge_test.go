package merge

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestMergeMaps(t *testing.T) {
	v1 := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	v2 := dyn.V(map[string]dyn.Value{
		"bar": dyn.V("qux"),
		"qux": dyn.V("foo"),
	})

	// Merge v2 into v1.
	{
		out, err := Merge(v1, v2)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
			"bar": "qux",
			"qux": "foo",
		}, out.AsAny())
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
	v1 := dyn.V([]dyn.Value{
		dyn.V("bar"),
		dyn.V("baz"),
	})

	v2 := dyn.V([]dyn.Value{
		dyn.V("qux"),
		dyn.V("foo"),
	})

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
	v1 := dyn.V("bar")
	v2 := dyn.V("baz")

	// Merge v2 into v1.
	{
		out, err := Merge(v1, v2)
		assert.NoError(t, err)
		assert.Equal(t, "baz", out.AsAny())
	}

	// Merge v1 into v2.
	{
		out, err := Merge(v2, v1)
		assert.NoError(t, err)
		assert.Equal(t, "bar", out.AsAny())
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
