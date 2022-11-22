package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	root := load(t, "./basic")
	assert.Equal(t, "basic", root.Bundle.Name)
}
