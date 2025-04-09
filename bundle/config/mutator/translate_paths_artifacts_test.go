package mutator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslatePathsArtifacts_InsideSyncRoot(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "bundle")
	lib := filepath.Join(dir, "my_lib")
	_ = os.MkdirAll(lib, 0o755)
	_ = os.MkdirAll(dir, 0o755)

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Artifacts: map[string]*config.Artifact{
				"my_artifact": {
					Type: "wheel",

					// Assume this is defined in a subdir to the sync root.
					Path: "../my_lib",
				},
			},
		},
	}

	bundletest.SetLocation(b, "artifacts", []dyn.Location{{
		File: filepath.Join(dir, "config/artifacts.yml"),
	}})

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// Assert that the artifact path has been converted to a local absolute path.
	assert.Equal(t, filepath.ToSlash(lib), b.Config.Artifacts["my_artifact"].Path)
}

func TestTranslatePathsArtifacts_OutsideSyncRoot(t *testing.T) {
	tmp := t.TempDir()
	lib := filepath.Join(tmp, "my_lib")
	dir := filepath.Join(tmp, "bundle")
	_ = os.MkdirAll(lib, 0o755)
	_ = os.MkdirAll(dir, 0o755)

	b := &bundle.Bundle{
		SyncRootPath:   dir,
		BundleRootPath: dir,
		SyncRoot:       vfs.MustNew(dir),
		Config: config.Root{
			Artifacts: map[string]*config.Artifact{
				"my_artifact": {
					Type: "wheel",

					// Assume this is defined in a subdir of the bundle root.
					Path: "../../my_lib",
				},
			},
		},
	}

	bundletest.SetLocation(b, "artifacts", []dyn.Location{{
		File: filepath.Join(dir, "config/artifacts.yml"),
	}})

	diags := bundle.ApplySeq(context.Background(), b, mutator.NormalizePaths(), mutator.TranslatePaths())
	require.NoError(t, diags.Error())

	// Assert that the artifact path has been converted to a local absolute path.
	assert.Equal(t, filepath.ToSlash(lib), b.Config.Artifacts["my_artifact"].Path)
}
