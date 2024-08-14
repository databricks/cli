package config_tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/phases"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPythonWheelBuild(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/python_wheel")
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build()))
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel/my_test_code/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Equal(t, 1, len(matches))

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildAutoDetect(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/python_wheel_no_artifact")
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build()))
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel_no_artifact/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Equal(t, 1, len(matches))

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildAutoDetectWithNotebookTask(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/python_wheel_no_artifact_notebook")
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build()))
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel_no_artifact_notebook/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Equal(t, 1, len(matches))

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelWithDBFSLib(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/python_wheel_dbfs_lib")
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build()))
	require.NoError(t, diags.Error())

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildNoBuildJustUpload(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/python_wheel_no_artifact_no_setup")
	require.NoError(t, err)

	b.Config.Workspace.ArtifactPath = "/foo/bar"

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("my_test_code-0.0.1-py3-none-any.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil)

	u := libraries.UploadWithClient(mockFiler)
	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build(), libraries.ExpandGlobReferences(), u))
	require.NoError(t, diags.Error())
	require.Empty(t, diags)

	require.Equal(t, "/Workspace/foo/bar/.internal/my_test_code-0.0.1-py3-none-any.whl", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[0].Libraries[0].Whl)
}

func TestPythonWheelBuildWithEnvironmentKey(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/environment_key")
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build()))
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/environment_key/my_test_code/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Equal(t, 1, len(matches))

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildMultiple(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/python_wheel_multiple")
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build()))
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel_multiple/my_test_code/dist/my_test_code*.whl")
	require.NoError(t, err)
	require.Equal(t, 2, len(matches))

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelNoBuild(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./python_wheel/python_wheel_no_build")
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, bundle.Seq(phases.Load(), phases.Build()))
	require.NoError(t, diags.Error())

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}
