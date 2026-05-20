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

func TestTranslatePathsGenieSpaces_FilePathRelativeSubDirectory(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "src", "my_space.geniespace.json"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				GenieSpaces: map[string]*resources.GenieSpace{
					"genie_space": {
						GenieSpaceConfig: resources.GenieSpaceConfig{
							Title: "My Genie Space",
						},
						FilePath: "../src/my_space.geniespace.json",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.genie_spaces", []dyn.Location{{
		File: filepath.Join(dir, "resources", "genie_space.yml"),
	}})

	// Genie space paths reuse the dashboard translator; there is no separate
	// genie_space mutator. The dashboard translator walks all resource types
	// that need path translation, so calling it covers genie_spaces too.
	diags := bundle.ApplySeq(t.Context(), b, mutator.NormalizePaths(), mutator.TranslatePathsDashboards())
	require.NoError(t, diags.Error())

	assert.Equal(
		t,
		filepath.ToSlash(filepath.Join("src", "my_space.geniespace.json")),
		b.Config.Resources.GenieSpaces["genie_space"].FilePath,
	)
}
