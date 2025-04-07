package config_tests

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/stretchr/testify/assert"
)

func TestSyncOverride(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/override", "development")
	assert.Equal(t, filepath.FromSlash("sync/override"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{"src/*", "tests/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{"dist"}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override", "staging")
	assert.Equal(t, filepath.FromSlash("sync/override"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{"src/*", "fixtures/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override", "prod")
	assert.Equal(t, filepath.FromSlash("sync/override"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{"src/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)
}

func TestSyncOverrideNoRootSync(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/override_no_root", "development")
	assert.Equal(t, filepath.FromSlash("sync/override_no_root"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{"tests/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{"dist"}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override_no_root", "staging")
	assert.Equal(t, filepath.FromSlash("sync/override_no_root"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{"fixtures/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override_no_root", "prod")
	assert.Equal(t, filepath.FromSlash("sync/override_no_root"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)
}

func TestSyncNil(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/nil", "development")
	assert.Equal(t, filepath.FromSlash("sync/nil"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.Nil(t, b.Config.Sync.Include)
	assert.Nil(t, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/nil", "staging")
	assert.Equal(t, filepath.FromSlash("sync/nil"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{"tests/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{"dist"}, b.Config.Sync.Exclude)
}

func TestSyncNilRoot(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/nil_root", "development")
	assert.Equal(t, filepath.FromSlash("sync/nil_root"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.Nil(t, b.Config.Sync.Include)
	assert.Nil(t, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/nil_root", "staging")
	assert.Equal(t, filepath.FromSlash("sync/nil_root"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)
	assert.ElementsMatch(t, []string{"tests/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{"dist"}, b.Config.Sync.Exclude)
}

func TestSyncPaths(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/paths", "development")
	assert.Equal(t, filepath.FromSlash("sync/paths"), b.SyncRootPath)
	assert.Equal(t, []string{"src", "development"}, b.Config.Sync.Paths)

	b = loadTarget(t, "./sync/paths", "staging")
	assert.Equal(t, filepath.FromSlash("sync/paths"), b.SyncRootPath)
	assert.Equal(t, []string{"src", "staging"}, b.Config.Sync.Paths)
}

func TestSyncPathsNoRoot(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/paths_no_root", "development")
	assert.Equal(t, filepath.FromSlash("sync/paths_no_root"), b.SyncRootPath)
	assert.ElementsMatch(t, []string{"development"}, b.Config.Sync.Paths)

	b = loadTarget(t, "./sync/paths_no_root", "staging")
	assert.Equal(t, filepath.FromSlash("sync/paths_no_root"), b.SyncRootPath)
	assert.ElementsMatch(t, []string{"staging"}, b.Config.Sync.Paths)

	// If not set at all, it defaults to "."
	b = loadTarget(t, "./sync/paths_no_root", "undefined")
	assert.Equal(t, filepath.FromSlash("sync/paths_no_root"), b.SyncRootPath)
	assert.Equal(t, []string{"."}, b.Config.Sync.Paths)

	// If set to nil, it won't sync anything.
	b = loadTarget(t, "./sync/paths_no_root", "nil")
	assert.Equal(t, filepath.FromSlash("sync/paths_no_root"), b.SyncRootPath)
	assert.Empty(t, b.Config.Sync.Paths)

	// If set to an empty sequence, it won't sync anything.
	b = loadTarget(t, "./sync/paths_no_root", "empty")
	assert.Equal(t, filepath.FromSlash("sync/paths_no_root"), b.SyncRootPath)
	assert.Empty(t, b.Config.Sync.Paths)
}

func TestSyncSharedCode(t *testing.T) {
	b := loadTarget(t, "./sync/shared_code/bundle", "default")
	assert.Equal(t, filepath.FromSlash("sync/shared_code"), b.SyncRootPath)
	assert.ElementsMatch(t, []string{"common", "bundle"}, b.Config.Sync.Paths)
}
