package mutator_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncInferRoot_NominalAbsolute(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/some/dir",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					".",
				},
				Include: []string{
					"foo",
					"bar",
				},
				Exclude: []string{
					"baz",
					"qux",
				},
			},
		},
	}

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SyncInferRoot())
	assert.NoError(t, diags.Error())
	assert.Equal(t, filepath.FromSlash("/tmp/some/dir"), b.SyncRootPath)

	// Check that the paths are unchanged.
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.Equal(t, []string{"foo", "bar"}, b.Config.Sync.Include)
	assert.Equal(t, []string{"baz", "qux"}, b.Config.Sync.Exclude)
}

func TestSyncInferRoot_NominalRelative(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "./some/dir",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					".",
				},
				Include: []string{
					"foo",
					"bar",
				},
				Exclude: []string{
					"baz",
					"qux",
				},
			},
		},
	}

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SyncInferRoot())
	assert.NoError(t, diags.Error())
	assert.Equal(t, filepath.FromSlash("some/dir"), b.SyncRootPath)

	// Check that the paths are unchanged.
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.Equal(t, []string{"foo", "bar"}, b.Config.Sync.Include)
	assert.Equal(t, []string{"baz", "qux"}, b.Config.Sync.Exclude)
}

func TestSyncInferRoot_ParentDirectory(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/some/dir",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					"../common",
				},
				Include: []string{
					"foo",
					"bar",
				},
				Exclude: []string{
					"baz",
					"qux",
				},
			},
		},
	}

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SyncInferRoot())
	assert.NoError(t, diags.Error())
	assert.Equal(t, filepath.FromSlash("/tmp/some"), b.SyncRootPath)

	// Check that the paths are updated.
	assert.Equal(t, []string{"common"}, b.Config.Sync.Paths)
	assert.Equal(t, []string{"dir/foo", "dir/bar"}, b.Config.Sync.Include)
	assert.Equal(t, []string{"dir/baz", "dir/qux"}, b.Config.Sync.Exclude)
}

func TestSyncInferRoot_ManyParentDirectories(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/some/dir/that/is/very/deeply/nested",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					"../../../../../../common",
				},
				Include: []string{
					"foo",
					"bar",
				},
				Exclude: []string{
					"baz",
					"qux",
				},
			},
		},
	}

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SyncInferRoot())
	assert.NoError(t, diags.Error())
	assert.Equal(t, filepath.FromSlash("/tmp/some"), b.SyncRootPath)

	// Check that the paths are updated.
	assert.Equal(t, []string{"common"}, b.Config.Sync.Paths)
	assert.Equal(t, []string{
		"dir/that/is/very/deeply/nested/foo",
		"dir/that/is/very/deeply/nested/bar",
	}, b.Config.Sync.Include)
	assert.Equal(t, []string{
		"dir/that/is/very/deeply/nested/baz",
		"dir/that/is/very/deeply/nested/qux",
	}, b.Config.Sync.Exclude)
}

func TestSyncInferRoot_MultiplePaths(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/some/bundle/root",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					"./foo",
					"../common",
					"./bar",
					"../../baz",
				},
			},
		},
	}

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SyncInferRoot())
	assert.NoError(t, diags.Error())
	assert.Equal(t, filepath.FromSlash("/tmp/some"), b.SyncRootPath)

	// Check that the paths are updated.
	assert.Equal(t, filepath.FromSlash("bundle/root/foo"), b.Config.Sync.Paths[0])
	assert.Equal(t, filepath.FromSlash("bundle/common"), b.Config.Sync.Paths[1])
	assert.Equal(t, filepath.FromSlash("bundle/root/bar"), b.Config.Sync.Paths[2])
	assert.Equal(t, filepath.FromSlash("baz"), b.Config.Sync.Paths[3])
}

func TestSyncInferRoot_Error(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/some/dir",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					"../../../../error",
					"../../../thisworks",
					"../../../../../error",
				},
			},
		},
	}

	bundletest.SetLocation(b, "sync.paths", []dyn.Location{{File: "databricks.yml"}})

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, mutator.SyncInferRoot())
	require.Len(t, diags, 2)
	assert.Equal(t, `invalid sync path "../../../../error"`, diags[0].Summary)
	assert.Equal(t, "databricks.yml:0:0", diags[0].Locations[0].String())
	assert.Equal(t, "sync.paths[0]", diags[0].Paths[0].String())
	assert.Equal(t, `invalid sync path "../../../../../error"`, diags[1].Summary)
	assert.Equal(t, "databricks.yml:0:0", diags[1].Locations[0].String())
	assert.Equal(t, "sync.paths[2]", diags[1].Paths[0].String())
}
