package dynvar_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

func TestDefaultLookup(t *testing.T) {
	lookup := dynvar.DefaultLookup(dyn.V(map[string]dyn.Value{
		"a": dyn.V("a"),
		"b": dyn.V("b"),
	}))

	v1, err := lookup(dyn.NewPath(dyn.Key("a")))
	assert.NoError(t, err)
	assert.Equal(t, dyn.V("a"), v1)

	v2, err := lookup(dyn.NewPath(dyn.Key("b")))
	assert.NoError(t, err)
	assert.Equal(t, dyn.V("b"), v2)

	_, err = lookup(dyn.NewPath(dyn.Key("c")))
	assert.True(t, dyn.IsNoSuchKeyError(err))
}
