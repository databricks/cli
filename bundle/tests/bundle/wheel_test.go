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
	t.Skip("Skipping test until fixing Python installation on GitHub Windows environment")

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

func TestBundlePythonWheelBuildNoBuildJustUpload(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel_no_artifact_no_setup")
	require.NoError(t, err)

	m := phases.Build()
	err = m.Apply(ctx, b)
	require.NoError(t, err)

	match := libraries.MatchWithArtifacts()
	err = match.Apply(ctx, b)
	require.ErrorContains(t, err, "./non-existing/*.whl")

	require.NotZero(t, len(b.Config.Artifacts))

	artifact := b.Config.Artifacts["my_test_code-0.0.1-py3-none-any.whl"]
	require.NotNil(t, artifact)
	require.Empty(t, artifact.BuildCommand)
	require.Contains(t, artifact.Files[0].Source, filepath.Join(
		b.Config.Path,
		"package",
		"my_test_code-0.0.1-py3-none-any.whl",
	))
	require.True(t, artifact.Files[0].NeedsUpload())
}
