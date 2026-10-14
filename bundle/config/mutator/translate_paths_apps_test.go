package mutator_test

import (
	"github.com/databricks/cli/libs/utils"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
							Name:           new("My App"),
							SourceCodePath: new("../src/app"),
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.apps", []diag.Location{{
		File: filepath.Join(dir, "resources/app.yml"),
	}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// Assert that the file path for the app has been converted to its local absolute path.
	assert.Equal(
		t,
		"/bundle/files/src/app",
		utils.Deref(b.Config.Resources.Apps["app"].SourceCodePath),
	)
}
