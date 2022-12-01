package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncludeOverride(t *testing.T) {
	b := load(t, "./include_override")
	assert.Empty(t, b.Config.Resources.Workflows)
}
