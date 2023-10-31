package merge

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
)

func TestMergeMaps(t *testing.T) {
	v1 := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V("baz"),
	})

	v2 := config.V(map[string]config.Value{
		"bar": config.V("qux"),
		"qux": config.V("foo"),
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
	v := config.V(map[string]config.Value{
		"foo": config.V("bar"),
	})

	// Merge nil into v.
	{
		out, err := Merge(v, config.NilValue)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
		}, out.AsAny())
	}

	// Merge v into nil.
	{
		out, err := Merge(config.NilValue, v)
		assert.NoError(t, err)
		assert.Equal(t, map[string]any{
			"foo": "bar",
		}, out.AsAny())
	}
}

func TestMergeMapsError(t *testing.T) {
	v := config.V(map[string]config.Value{
		"foo": config.V("bar"),
	})

	other := config.V("string")

	// Merge a string into v.
	{
		out, err := Merge(v, other)
		assert.EqualError(t, err, "cannot merge map with string")
		assert.Equal(t, config.NilValue, out)
	}
}

func TestMergeSequences(t *testing.T) {
	v1 := config.V([]config.Value{
		config.V("bar"),
		config.V("baz"),
	})

	v2 := config.V([]config.Value{
		config.V("qux"),
		config.V("foo"),
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
	v := config.V([]config.Value{
		config.V("bar"),
	})

	// Merge nil into v.
	{
		out, err := Merge(v, config.NilValue)
		assert.NoError(t, err)
		assert.Equal(t, []any{
			"bar",
		}, out.AsAny())
	}

	// Merge v into nil.
	{
		out, err := Merge(config.NilValue, v)
		assert.NoError(t, err)
		assert.Equal(t, []any{
			"bar",
		}, out.AsAny())
	}
}

func TestMergeSequencesError(t *testing.T) {
	v := config.V([]config.Value{
		config.V("bar"),
	})

	other := config.V("string")

	// Merge a string into v.
	{
		out, err := Merge(v, other)
		assert.EqualError(t, err, "cannot merge sequence with string")
		assert.Equal(t, config.NilValue, out)
	}
}

func TestMergePrimitives(t *testing.T) {
	v1 := config.V("bar")
	v2 := config.V("baz")

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
	v := config.V("bar")

	// Merge nil into v.
	{
		out, err := Merge(v, config.NilValue)
		assert.NoError(t, err)
		assert.Equal(t, "bar", out.AsAny())
	}

	// Merge v into nil.
	{
		out, err := Merge(config.NilValue, v)
		assert.NoError(t, err)
		assert.Equal(t, "bar", out.AsAny())
	}
}

func TestMergePrimitivesError(t *testing.T) {
	v := config.V("bar")
	other := config.V(map[string]config.Value{
		"foo": config.V("bar"),
	})

	// Merge a map into v.
	{
		out, err := Merge(v, other)
		assert.EqualError(t, err, "cannot merge string with map")
		assert.Equal(t, config.NilValue, out)
	}
}
