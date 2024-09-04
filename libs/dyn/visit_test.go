package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestVisitCallbackPathCopy(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{
		"foo": dyn.V(42),
		"bar": dyn.V(43),
	})

	var paths []dyn.Path

	// The callback should receive a copy of the path.
	// If the same underlying value is used, all collected paths will be the same.
	// This test uses `MapByPattern` to collect all paths in the map.
	// Visit itself doesn't have public functions and we exclusively use black-box testing for this package.
	_, _ = dyn.MapByPattern(vin, dyn.NewPattern(dyn.AnyKey()), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		paths = append(paths, p)
		return v, nil
	})

	// Verify that the paths retained their original values.
	var strings []string
	for _, p := range paths {
		strings = append(strings, p.String())
	}
	assert.ElementsMatch(t, strings, []string{
		"foo",
		"bar",
	})
}
