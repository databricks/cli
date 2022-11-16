package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncludeOverride(t *testing.T) {
	root := load(t, "./include_override")
	assert.Empty(t, root.Resources.Jobs)
}
