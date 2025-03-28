package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestVisitDashboardPaths(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Dashboards: map[string]*resources.Dashboard{
				"dashboard0": {
					FilePath: "foo.lvdash.json",
				},
			},
		},
	}

	actual := visitDashboardPaths(t, root)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.dashboards.dashboard0.file_path"),
	}

	assert.ElementsMatch(t, expected, actual)
}

func visitDashboardPaths(t *testing.T, root config.Root) []dyn.Path {
	var actual []dyn.Path
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return VisitDashboardPaths(value, func(p dyn.Path, mode TranslateMode, v dyn.Value) (dyn.Value, error) {
			actual = append(actual, p)
			return v, nil
		})
	})
	require.NoError(t, err)
	return actual
}
