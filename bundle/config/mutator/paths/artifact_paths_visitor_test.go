package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactPathsVisitor(t *testing.T) {
	root := config.Root{
		Artifacts: config.Artifacts{
			"artifact0": &config.Artifact{
				Path: "foo.whl",
			},
		},
	}

	actual := visitArtifactPaths(t, root)
	expected := []dyn.Path{
		dyn.MustPathFromString("artifacts.artifact0.path"),
	}

	assert.ElementsMatch(t, expected, actual)
}

func visitArtifactPaths(t *testing.T, root config.Root) []dyn.Path {
	var actual []dyn.Path
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return VisitArtifactPaths(value, func(p dyn.Path, mode TranslateMode, v dyn.Value) (dyn.Value, error) {
			actual = append(actual, p)
			return v, nil
		})
	})
	require.NoError(t, err)
	return actual
}
