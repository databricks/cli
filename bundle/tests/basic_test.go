package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	b := load(t, "./basic")
	assert.Equal(t, "basic", b.Config.Bundle.Name)
}
