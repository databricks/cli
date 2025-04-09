package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
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

	actual := collectVisitedPaths(t, root, VisitDashboardPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.dashboards.dashboard0.file_path"),
	}

	assert.ElementsMatch(t, expected, actual)
}
