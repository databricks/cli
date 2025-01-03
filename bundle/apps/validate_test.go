package apps

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/require"
)

func TestAppsValidate(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.Touch(t, tmpDir, "app1", "app.yml")
	testutil.Touch(t, tmpDir, "app2", "app.py")

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		SyncRootPath:   tmpDir,
		SyncRoot:       vfs.MustNew(tmpDir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/foo/bar/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"app1": {
						App: &apps.App{
							Name: "app1",
						},
						SourceCodePath: "./app1",
					},
					"app2": {
						App: &apps.App{
							Name: "app2",
						},
						SourceCodePath: "./app2",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	diags := bundle.Apply(context.Background(), b, bundle.Seq(mutator.TranslatePaths(), Validate()))
	require.Len(t, diags, 1)
	require.Equal(t, "app.yml detected", diags[0].Summary)
	require.Contains(t, diags[0].Detail, "app.yml and use 'config' property for app resource")
}

func TestAppsValidateSameSourcePath(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.Touch(t, tmpDir, "app1", "app.py")

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		SyncRootPath:   tmpDir,
		SyncRoot:       vfs.MustNew(tmpDir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/foo/bar/",
			},
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"app1": {
						App: &apps.App{
							Name: "app1",
						},
						SourceCodePath: "./app1",
					},
					"app2": {
						App: &apps.App{
							Name: "app2",
						},
						SourceCodePath: "./app1",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	diags := bundle.Apply(context.Background(), b, bundle.Seq(mutator.TranslatePaths(), Validate()))
	require.Len(t, diags, 1)
	require.Equal(t, "Duplicate app source code path", diags[0].Summary)
	require.Contains(t, diags[0].Detail, "has the same source code path as app resource")
}
