package mutator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTranslatePathsApps_SharedDirectoryWithArtifact reproduces the bug from
// https://github.com/databricks/cli/issues/4924 where the shared `seen` cache
// in translateContext caused an artifact's local absolute path to be returned
// for an app's source_code_path when both point to the same directory.
func TestTranslatePathsApps_SharedDirectoryWithArtifact(t *testing.T) {
	dir := t.TempDir()
	// Create the shared directory that both the artifact and the app reference.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src", "app"), 0o755))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle/files",
			},
			Artifacts: map[string]*config.Artifact{
				"my_artifact": {
					Type: config.ArtifactPythonWheel,
					// Points to the same local directory as the app below.
					Path: "src/app",
				},
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app": {
						App: apps.App{
							Name: "My App",
						},
						// Points to the same local directory as the artifact above.
						SourceCodePath: "src/app",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "artifacts", []dyn.Location{{File: filepath.Join(dir, "databricks.yml")}})
	bundletest.SetLocation(b, "resources.apps", []dyn.Location{{File: filepath.Join(dir, "databricks.yml")}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// The artifact path must be a local absolute path.
	assert.Equal(t, filepath.ToSlash(filepath.Join(dir, "src/app")), b.Config.Artifacts["my_artifact"].Path)

	// The app source_code_path must be a remote workspace path, not the local absolute path.
	assert.Equal(t, "/bundle/files/src/app", b.Config.Resources.Apps["my_app"].SourceCodePath)
}

func TestTranslatePathsApps_FilePathRelativeSubDirectory(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "src", "app", "app.py"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle/files",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"app": {
						App: apps.App{
							Name: "My App",
						},
						SourceCodePath: "../src/app",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.apps", []dyn.Location{{
		File: filepath.Join(dir, "resources/app.yml"),
	}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// Assert that the file path for the app has been converted to its local absolute path.
	assert.Equal(
		t,
		"/bundle/files/src/app",
		b.Config.Resources.Apps["app"].SourceCodePath,
	)
}
