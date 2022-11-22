package config_tests

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

func TestIncludeDefault(t *testing.T) {
	root := load(t, "./include_default")

	// Test that both jobs were loaded.
	keys := maps.Keys(root.Resources.Jobs)
	sort.Strings(keys)
	assert.Equal(t, []string{"my_first_job", "my_second_job"}, keys)
	assert.Equal(t, "1", root.Resources.Jobs["my_first_job"].ID)
	assert.Equal(t, "2", root.Resources.Jobs["my_second_job"].ID)
}
