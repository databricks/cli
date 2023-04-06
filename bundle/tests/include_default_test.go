package config_tests

import (
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
	assert.Equal(t, "1", b.Config.Resources.Jobs["my_first_job"].ID)
	assert.Equal(t, "my_first_job/resource.yml", b.Config.Resources.Jobs["my_first_job"].ConfigFilePath)
	assert.Equal(t, "2", b.Config.Resources.Jobs["my_second_job"].ID)
	assert.Equal(t, "my_second_job/resource.yml", b.Config.Resources.Jobs["my_second_job"].ConfigFilePath)
}
