package config_tests

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

func TestIncludeDefault(t *testing.T) {
	b := load(t, "./include_default")

	// Test that both workflows were loaded.
	keys := maps.Keys(b.Config.Resources.Workflows)
	sort.Strings(keys)
	assert.Equal(t, []string{"my_first_workflow", "my_second_workflow"}, keys)
	assert.Equal(t, "1", b.Config.Resources.Workflows["my_first_workflow"].ID)
	assert.Equal(t, "2", b.Config.Resources.Workflows["my_second_workflow"].ID)
}
