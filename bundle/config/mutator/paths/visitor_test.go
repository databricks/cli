package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

// collectVisitedPaths is a helper function that collects all visited paths for testing
func collectVisitedPaths(t *testing.T, root config.Root, visitFn func(value dyn.Value, fn VisitFunc) (dyn.Value, error)) []dyn.Path {
	var actual []dyn.Path
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return visitFn(value, func(p dyn.Path, mode TranslateMode, v dyn.Value) (dyn.Value, error) {
			actual = append(actual, p)
			return v, nil
		})
	})
	require.NoError(t, err)
	return actual
}
