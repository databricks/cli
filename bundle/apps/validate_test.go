package apps

import (
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
						App: apps.App{
							Name: "app1",
						},
						SourceCodePath: "./app1",
					},
					"app2": {
						App: apps.App{
							Name: "app2",
						},
						SourceCodePath: "./app1",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.TranslatePaths(), Validate())
	require.Len(t, diags, 1)
	require.Equal(t, "Duplicate app source code path", diags[0].Summary)
	require.Contains(t, diags[0].Detail, "has the same source code path as app resource")
}

func TestAppsValidateBothSourceCodePathAndGitSource(t *testing.T) {
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
						App: apps.App{
							Name: "app1",
						},
						SourceCodePath: "./app1",
						GitSource: &apps.GitSource{
							Branch: "main",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.TranslatePaths(), Validate())
	require.Len(t, diags, 1)
	require.Equal(t, "Both source_code_path and git_source fields are set", diags[0].Summary)
	require.Contains(t, diags[0].Detail, "should have either source_code_path or git_source field, not both")
}
