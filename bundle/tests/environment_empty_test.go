package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvironmentEmpty(t *testing.T) {
	b := loadEnvironment(t, "./environment_empty", "development")
	assert.Equal(t, "development", b.Config.Bundle.Environment)
}
