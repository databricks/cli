package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/require"
)

// collectVisitedPaths is a helper function that collects all visited paths for testing
func collectVisitedPaths(t *testing.T, root config.Root, visitFn func(root *config.Root, fn VisitFunc) error) []*structpath.PathNode {
	var actual []*structpath.PathNode
	err := visitFn(&root, func(p *structpath.PathNode, mode TranslateMode, s *string) error {
		actual = append(actual, p)
		return nil
	})
	require.NoError(t, err)
	return actual
}
