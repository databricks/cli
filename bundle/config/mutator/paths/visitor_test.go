package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/require"
)

// collectVisitedPaths is a helper function that collects all visited paths for testing
func collectVisitedPaths(t *testing.T, root config.Root, visitFn func(value structvar.View, fn VisitFunc) error) []*structpath.PathNode {
	var actual []*structpath.PathNode
	err := visitFn(root.View(), func(p *structpath.PathNode, mode TranslateMode, v structvar.View) error {
		actual = append(actual, p)
		return nil
	})
	require.NoError(t, err)
	return actual
}
