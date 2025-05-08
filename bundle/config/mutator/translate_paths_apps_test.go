package mutator_test

import (
	"context"
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

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// Assert that the file path for the app has been converted to its local absolute path.
	assert.Equal(
		t,
		"/bundle/files/src/app",
		b.Config.Resources.Apps["app"].SourceCodePath,
	)
}
