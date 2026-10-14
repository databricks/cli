package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/stretchr/testify/assert"
)

func TestAppPathsVisitor(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Apps: map[string]*resources.App{
				"app0": {
					App: apps.App{
						SourceCodePath: new("foo"),
					},
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
