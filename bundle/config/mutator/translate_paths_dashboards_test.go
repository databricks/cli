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
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslatePathsDashboards_FilePathRelativeSubDirectory(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "src", "my_dashboard.lvdash.json"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"dashboard": {
						DashboardConfig: resources.DashboardConfig{
							Dashboard: dashboards.Dashboard{
								DisplayName: "My Dashboard",
							},
						},
						FilePath: "../src/my_dashboard.lvdash.json",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.dashboards", []dyn.Location{{
		File: filepath.Join(dir, "resources/dashboard.yml"),
	}})

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// Assert that the file path for the dashboard has been converted to its local absolute path.
	assert.Equal(
		t,
		filepath.ToSlash(filepath.Join("src", "my_dashboard.lvdash.json")),
		b.Config.Resources.Dashboards["dashboard"].FilePath,
	)
}
