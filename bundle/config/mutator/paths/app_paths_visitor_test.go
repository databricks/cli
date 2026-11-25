package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
)

func TestAppPathsVisitor(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Apps: map[string]*resources.App{
				"app0": {
					App: apps.App{
						SourceCodePath: "foo",
					},
					ThumbnailPath: "thumbnail.png",
				},
			},
		},
	}

	actual := collectVisitedPaths(t, root)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.apps.app0.source_code_path"),
		dyn.MustPathFromString("resources.apps.app0.thumbnail_path"),
	}

	assert.ElementsMatch(t, expected, actual)
}
