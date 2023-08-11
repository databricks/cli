package bundle

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/phases"
	"github.com/stretchr/testify/require"
)

func TestBundlePythonWheelBuild(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel")
	require.NoError(t, err)

	m := phases.Build()
	err = m.Apply(ctx, b)
	require.NoError(t, err)

	matches, err := filepath.Glob("python_wheel/my_test_code/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Equal(t, 1, len(matches))

	match := libraries.MatchWithArtifacts()
	err = match.Apply(ctx, b)
	require.NoError(t, err)
}

func TestBundlePythonWheelBuildAutoDetect(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel_no_artifact")
	require.NoError(t, err)

	m := phases.Build()
	err = m.Apply(ctx, b)
	require.NoError(t, err)

	matches, err := filepath.Glob("python_wheel/my_test_code/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Equal(t, 1, len(matches))

	match := libraries.MatchWithArtifacts()
	err = match.Apply(ctx, b)
	require.NoError(t, err)
}

func TestBundlePythonWheelWithDBFSLib(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel_dbfs_lib")
	require.NoError(t, err)

	m := phases.Build()
	err = m.Apply(ctx, b)
	require.NoError(t, err)

	match := libraries.MatchWithArtifacts()
	err = match.Apply(ctx, b)
	require.NoError(t, err)
}
