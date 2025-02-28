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

func TestPythonWheelBuildNoBuildJustUpload(t *testing.T) {
	b := loadTarget(t, "./python_wheel/python_wheel_no_artifact_no_setup", "default")

	ctx := context.Background()
	diags := phases.Build(ctx, b)
	require.NoError(t, diags.Error())

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.EXPECT().Write(
		mock.Anything,
		filepath.Join("my_test_code-0.0.1-py3-none-any.whl"),
		mock.AnythingOfType("*os.File"),
		filer.OverwriteIfExists,
		filer.CreateParentDirectories,
	).Return(nil)

	diags = bundle.ApplySeq(ctx, b,
		libraries.ExpandGlobReferences(),
		libraries.UploadWithClient(mockFiler),
	)
	require.NoError(t, diags.Error())
	require.Empty(t, diags)
	require.Equal(t, "/Workspace/foo/bar/.internal/my_test_code-0.0.1-py3-none-any.whl", b.Config.Resources.Jobs["test_job"].JobSettings.Tasks[0].Libraries[0].Whl)
}
