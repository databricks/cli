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
	assert.ElementsMatch(t, []string{filepath.FromSlash("src/*"), filepath.FromSlash("tests/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{filepath.FromSlash("dist")}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override", "staging")
	assert.ElementsMatch(t, []string{filepath.FromSlash("src/*"), filepath.FromSlash("fixtures/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override", "prod")
	assert.ElementsMatch(t, []string{filepath.FromSlash("src/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)
}

func TestSyncOverrideNoRootSync(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/override_no_root", "development")
	assert.ElementsMatch(t, []string{filepath.FromSlash("tests/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{filepath.FromSlash("dist")}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override_no_root", "staging")
	assert.ElementsMatch(t, []string{filepath.FromSlash("fixtures/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/override_no_root", "prod")
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)
}

func TestSyncNil(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/nil", "development")
	assert.Nil(t, b.Config.Sync.Include)
	assert.Nil(t, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/nil", "staging")
	assert.ElementsMatch(t, []string{filepath.FromSlash("tests/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{filepath.FromSlash("dist")}, b.Config.Sync.Exclude)
}

func TestSyncNilRoot(t *testing.T) {
	var b *bundle.Bundle

	b = loadTarget(t, "./sync/nil_root", "development")
	assert.Nil(t, b.Config.Sync.Include)
	assert.Nil(t, b.Config.Sync.Exclude)

	b = loadTarget(t, "./sync/nil_root", "staging")
	assert.ElementsMatch(t, []string{filepath.FromSlash("tests/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{filepath.FromSlash("dist")}, b.Config.Sync.Exclude)
}
