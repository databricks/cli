package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideSyncDevTarget(t *testing.T) {
	b := load(t, "./override_sync")
	assert.ElementsMatch(t, []string{"src/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync", "development")
	assert.ElementsMatch(t, []string{"src/*", "tests/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{"dist"}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync", "staging")
	assert.ElementsMatch(t, []string{"src/*", "fixtures/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)

	b = loadTarget(t, "./override_sync", "prod")
	assert.ElementsMatch(t, []string{"src/*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{}, b.Config.Sync.Exclude)
}
