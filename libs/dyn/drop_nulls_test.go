package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestDropNulls(t *testing.T) {
	loc := diag.Location{File: "file", Line: 1, Column: 2}
	v := dyn.NewValue(map[string]dyn.Value{
		"null": dyn.NilValue,
		"map": dyn.V(map[string]dyn.Value{
			"null":  dyn.NilValue,
			"value": dyn.V("a"),
		}),
		"list": dyn.V([]dyn.Value{
			dyn.NilValue,
			dyn.V(map[string]dyn.Value{"null": dyn.NilValue}),
		}),
	}, []diag.Location{loc})

	out := dyn.DropNulls(v)
	assert.Equal(t, map[string]any{
		"map":  map[string]any{"value": "a"},
		"list": []any{nil, map[string]any{}},
	}, out.AsAny())
	assert.Equal(t, []diag.Location{loc}, out.Locations())
}

func TestDropNullsNil(t *testing.T) {
	assert.Equal(t, dyn.NilValue, dyn.DropNulls(dyn.NilValue))
}
