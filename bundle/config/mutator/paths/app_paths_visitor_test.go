package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestAppPathsVisitor(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Apps: map[string]*resources.App{
				"app0": {
					SourceCodePath: "foo",
				},
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitAppPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.apps.app0.source_code_path"),
	}

	assert.ElementsMatch(t, expected, actual)
}
