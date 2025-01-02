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
	b := loadTarget(t, "./python_wheel/python_wheel", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel/my_test_code/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Len(t, matches, 1)

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildAutoDetect(t *testing.T) {
	b := loadTarget(t, "./python_wheel/python_wheel_no_artifact", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel_no_artifact/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Len(t, matches, 1)

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildAutoDetectWithNotebookTask(t *testing.T) {
	b := loadTarget(t, "./python_wheel/python_wheel_no_artifact_notebook", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel_no_artifact_notebook/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Len(t, matches, 1)

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelWithDBFSLib(t *testing.T) {
	b := loadTarget(t, "./python_wheel/python_wheel_dbfs_lib", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildNoBuildJustUpload(t *testing.T) {
	b := loadTarget(t, "./python_wheel/python_wheel_no_artifact_no_setup", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("my_test_code-0.0.1-py3-none-any.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil)

	diags = bundle.Apply(ctx, b, bundle.Seq(
		libraries.ExpandGlobReferences(),
		libraries.UploadWithClient(mockFiler),
	))
	require.NoError(t, diags.Error())
	require.Empty(t, diags)
	require.Equal(t, "/Workspace/foo/bar/.internal/my_test_code-0.0.1-py3-none-any.whl", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[0].Libraries[0].Whl)
}

func TestPythonWheelBuildWithEnvironmentKey(t *testing.T) {
	b := loadTarget(t, "./python_wheel/environment_key", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/environment_key/my_test_code/dist/my_test_code-*.whl")
	require.NoError(t, err)
	require.Len(t, matches, 1)

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelBuildMultiple(t *testing.T) {
	b := loadTarget(t, "./python_wheel/python_wheel_multiple", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	matches, err := filepath.Glob("./python_wheel/python_wheel_multiple/my_test_code/dist/my_test_code*.whl")
	require.NoError(t, err)
	require.Len(t, matches, 2)

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}

func TestPythonWheelNoBuild(t *testing.T) {
	b := loadTarget(t, "./python_wheel/python_wheel_no_build", "default")

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, phases.Build())
	require.NoError(t, diags.Error())

	match := libraries.ExpandGlobReferences()
	diags = bundle.Apply(ctx, b, match)
	require.NoError(t, diags.Error())
}
