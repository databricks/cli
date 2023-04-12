package config_tests

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

func TestIncludeDefault(t *testing.T) {
	b := load(t, "./include_default")

	// Test that both jobs were loaded.
	keys := maps.Keys(b.Config.Resources.Jobs)
	sort.Strings(keys)
	assert.Equal(t, []string{"my_first_job", "my_second_job"}, keys)

	first := b.Config.Resources.Jobs["my_first_job"]
	assert.Equal(t, "1", first.ID)
	assert.Equal(t, "include_default/my_first_job/resource.yml", filepath.ToSlash(first.ConfigFilePath))

	second := b.Config.Resources.Jobs["my_second_job"]
	assert.Equal(t, "2", second.ID)
	assert.Equal(t, "include_default/my_second_job/resource.yml", filepath.ToSlash(second.ConfigFilePath))
}
