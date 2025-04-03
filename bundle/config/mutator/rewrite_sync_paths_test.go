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
)

func TestRewriteSyncPathsRelative(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: ".",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					".",
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

	bundletest.SetLocation(b, "sync.paths[0]", []dyn.Location{{File: "./databricks.yml"}})
	bundletest.SetLocation(b, "sync.paths[1]", []dyn.Location{{File: "./databricks.yml"}})
	bundletest.SetLocation(b, "sync.include[0]", []dyn.Location{{File: "./file.yml"}})
	bundletest.SetLocation(b, "sync.include[1]", []dyn.Location{{File: "./a/file.yml"}})
	bundletest.SetLocation(b, "sync.exclude[0]", []dyn.Location{{File: "./a/b/file.yml"}})
	bundletest.SetLocation(b, "sync.exclude[1]", []dyn.Location{{File: "./a/b/c/file.yml"}})

	diags := bundle.Apply(context.Background(), b, mutator.RewriteSyncPaths())
	assert.NoError(t, diags.Error())

	assert.Equal(t, filepath.Clean("."), b.Config.Sync.Paths[0])
	assert.Equal(t, filepath.Clean("../common"), b.Config.Sync.Paths[1])
	assert.Equal(t, "foo", b.Config.Sync.Include[0])
	assert.Equal(t, "a/bar", b.Config.Sync.Include[1])
	assert.Equal(t, "a/b/baz", b.Config.Sync.Exclude[0])
	assert.Equal(t, "a/b/c/qux", b.Config.Sync.Exclude[1])
}

func TestRewriteSyncPathsAbsolute(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath: "/tmp/dir",
		Config: config.Root{
			Sync: config.Sync{
				Paths: []string{
					".",
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

	bundletest.SetLocation(b, "sync.paths[0]", []dyn.Location{{File: "/tmp/dir/databricks.yml"}})
	bundletest.SetLocation(b, "sync.paths[1]", []dyn.Location{{File: "/tmp/dir/databricks.yml"}})
	bundletest.SetLocation(b, "sync.include[0]", []dyn.Location{{File: "/tmp/dir/file.yml"}})
	bundletest.SetLocation(b, "sync.include[1]", []dyn.Location{{File: "/tmp/dir/a/file.yml"}})
	bundletest.SetLocation(b, "sync.exclude[0]", []dyn.Location{{File: "/tmp/dir/a/b/file.yml"}})
	bundletest.SetLocation(b, "sync.exclude[1]", []dyn.Location{{File: "/tmp/dir/a/b/c/file.yml"}})

	diags := bundle.Apply(context.Background(), b, mutator.RewriteSyncPaths())
	assert.NoError(t, diags.Error())

	assert.Equal(t, filepath.Clean("."), b.Config.Sync.Paths[0])
	assert.Equal(t, filepath.Clean("../common"), b.Config.Sync.Paths[1])
	assert.Equal(t, "foo", b.Config.Sync.Include[0])
	assert.Equal(t, "a/bar", b.Config.Sync.Include[1])
	assert.Equal(t, "a/b/baz", b.Config.Sync.Exclude[0])
	assert.Equal(t, "a/b/c/qux", b.Config.Sync.Exclude[1])
}

func TestRewriteSyncPathsErrorPaths(t *testing.T) {
	t.Run("no sync block", func(t *testing.T) {
		b := &bundle.Bundle{
			BundleRootPath: ".",
		}

		diags := bundle.Apply(context.Background(), b, mutator.RewriteSyncPaths())
		assert.NoError(t, diags.Error())
	})

	t.Run("empty include/exclude blocks", func(t *testing.T) {
		b := &bundle.Bundle{
			BundleRootPath: ".",
			Config: config.Root{
				Sync: config.Sync{
					Include: []string{},
					Exclude: []string{},
				},
			},
		}

		diags := bundle.Apply(context.Background(), b, mutator.RewriteSyncPaths())
		assert.NoError(t, diags.Error())
	})
}
