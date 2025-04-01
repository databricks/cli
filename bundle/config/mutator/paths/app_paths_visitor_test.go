package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
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

	actual := visitAppPaths(t, root)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.apps.app0.source_code_path"),
	}

	assert.ElementsMatch(t, expected, actual)
}

func visitAppPaths(t *testing.T, root config.Root) []dyn.Path {
	var actual []dyn.Path
	err := root.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return VisitAppPaths(value, func(p dyn.Path, mode TranslateMode, v dyn.Value) (dyn.Value, error) {
			actual = append(actual, p)
			return v, nil
		})
	})
	require.NoError(t, err)
	return actual
}
