package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
)

func TestArtifactPathsVisitor(t *testing.T) {
	root := config.Root{
		Artifacts: config.Artifacts{
			"artifact0": &config.Artifact{
				Path: "foo.whl",
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitArtifactPaths)
	expected := structpath.MustParsePaths("artifacts.artifact0.path")

	assert.ElementsMatch(t, expected, actual)
}
