package bundle

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/stretchr/testify/require"
)

func TestBundlePythonWheelBuild(t *testing.T) {
	b, err := bundle.Load("./python_wheel")
	require.NoError(t, err)

	m := phases.Build()
	err = m.Apply(context.Background(), b)
	require.NoError(t, err)

	matches, err := filepath.Glob("python_wheel/my_test_code/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Equal(t, 1, len(matches))
}
