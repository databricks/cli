package mutator_test

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/knowledge"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslatePathsPages_FilePathRelativeSubDirectory(t *testing.T) {
	dir := t.TempDir()
	touchEmptyFile(t, filepath.Join(dir, "src", "chargeback.md"))

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Resources: config.Resources{
				Pages: map[string]*resources.Page{
					"page": {
						Page: knowledge.Page{
							Domain:      "domains/1234",
							DisplayName: "Chargeback",
						},
						FilePath: "../src/chargeback.md",
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.pages", []diag.Location{{
		File: filepath.Join(dir, "resources", "page.yml"),
	}})

	// Page paths are translated together with dashboard and Genie space paths.
	diags := bundle.ApplySeq(t.Context(), b, mutator.NormalizePaths(), mutator.TranslatePathsDashboards())
	require.NoError(t, diags.Error())

	// The path is relative to the file that defines the page, and is rewritten to be
	// relative to the bundle sync root.
	assert.Equal(
		t,
		filepath.ToSlash(filepath.Join("src", "chargeback.md")),
		b.Config.Resources.Pages["page"].FilePath,
	)
}
