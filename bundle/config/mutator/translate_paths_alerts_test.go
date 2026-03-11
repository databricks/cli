package mutator_test

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslatePathsAlerts_FilePathRelativeSubDirectory(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "src", "my_alert.dbalert.json"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: "/bundle",
			},
			Resources: config.Resources{
				Alerts: map[string]*resources.Alert{
					"alert": {
						FilePath: "../src/my_alert.dbalert.json",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.alerts", []dyn.Location{{
		File: filepath.Join(dir, "resources/alert.yml"),
	}})

	diags := bundle.ApplySeq(t.Context(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	assert.Equal(
		t,
		"/bundle/src/my_alert.dbalert.json",
		b.Config.Resources.Alerts["alert"].FilePath,
	)
}
