package dyn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisitCallbackPathCopy(t *testing.T) {
	vin := V(map[string]Value{
		"foo": V(42),
		"bar": V(43),
	})

	var paths []Path

	// The callback should receive a copy of the path.
	// If the same underlying value is used, all collected paths will be the same.
	_, _ = visit(vin, EmptyPath, NewPattern(AnyKey()), visitOptions{
		fn: func(p Path, v Value) (Value, error) {
			paths = append(paths, p)
			return v, nil
		},
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
