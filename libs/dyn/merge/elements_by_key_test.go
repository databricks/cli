package merge

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestElementByKey(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V(map[string]dyn.Value{
			"key":   dyn.V("foo"),
			"value": dyn.V(42),
		}),
		dyn.V(map[string]dyn.Value{
			"key":   dyn.V("bar"),
			"value": dyn.V(43),
		}),
		dyn.V(map[string]dyn.Value{
			// Use upper case key to test that the resulting element has its
			// key field assigned to the output of the key function.
			// The key function in this test returns the lower case version of the key.
			"key":   dyn.V("FOO"),
			"value": dyn.V(44),
		}),
	})

	keyFunc := func(v dyn.Value) string {
		return strings.ToLower(v.MustString())
	}

	vout, err := dyn.MapByPath(vin, dyn.EmptyPath, ElementsByKey("key", keyFunc))
	require.NoError(t, err)
	assert.Len(t, vout.MustSequence(), 2)
	assert.Equal(t,
		vout.Index(0).AsAny(),
		map[string]any{
			"key":   "foo",
			"value": 44,
		},
	)
	assert.Equal(t,
		vout.Index(1).AsAny(),
		map[string]any{
			"key":   "bar",
			"value": 43,
		},
	)
}

func TestElementByKeyWithOverride(t *testing.T) {
	vin := dyn.V([]dyn.Value{
		dyn.V(map[string]dyn.Value{
			"key":   dyn.V("foo"),
			"value": dyn.V(42),
		}),
		dyn.V(map[string]dyn.Value{
			"key":   dyn.V("bar"),
			"value": dyn.V(43),
		}),
		dyn.V(map[string]dyn.Value{
			"key":        dyn.V("foo"),
			"othervalue": dyn.V(44),
		}),
	})

	keyFunc := func(v dyn.Value) string {
		return strings.ToLower(v.MustString())
	}

	vout, err := dyn.MapByPath(vin, dyn.EmptyPath, ElementsByKeyWithOverride("key", keyFunc))
	require.NoError(t, err)
	assert.Len(t, vout.MustSequence(), 2)
	assert.Equal(t,
		vout.Index(0).AsAny(),
		map[string]any{
			"key":        "foo",
			"othervalue": 44,
		},
	)
	assert.Equal(t,
		vout.Index(1).AsAny(),
		map[string]any{
			"key":   "bar",
			"value": 43,
		},
	)
}
