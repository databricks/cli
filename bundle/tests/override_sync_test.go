package config_tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideSyncTarget(t *testing.T) {
	b := load(t, "./override_sync")
	assert.ElementsMatch(t, []string{filepath.Clean("src/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync", "development")
	assert.ElementsMatch(t, []string{filepath.Clean("src/*"), filepath.Clean("tests/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{filepath.Clean("dist")}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync", "staging")
	assert.ElementsMatch(t, []string{filepath.Clean("src/*"), filepath.Clean("fixtures/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync", "prod")
	assert.ElementsMatch(t, []string{filepath.Clean("src/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)
}

func TestOverrideSyncTargetNoRootSync(t *testing.T) {
	b := load(t, "./override_sync_no_root")
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync_no_root", "development")
	assert.ElementsMatch(t, []string{filepath.Clean("tests/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{filepath.Clean("dist")}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync_no_root", "staging")
	assert.ElementsMatch(t, []string{filepath.Clean("fixtures/*")}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync_no_root", "prod")
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)
}
