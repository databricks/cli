package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
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

	actual := collectVisitedPaths(t, root, VisitAppPaths)
	expected := structpath.MustParsePaths(
		"resources.apps.app0.source_code_path",
		"resources.apps.app0.thumbnail_path",
	)

	assert.ElementsMatch(t, expected, actual)
}
